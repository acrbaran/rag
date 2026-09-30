import { onUnmounted, ref, type Ref } from 'vue'
import { post } from '@/utils/request'
import { readStoredPtyId, writeStoredPtyId } from '@/utils/sandboxPtyId'

export type SandboxTerminalStatus =
  /** Sorgulama, oturum sandbox'ının duraklatıldığını buldu; uyandırma kullanıcı onayı gerektirir.*/
  | 'paused'
  | 'connecting'
  | 'ready'
  /** Sorgulama, oturum için çalışan sandbox olmadığını buldu; oluşturma kullanıcı onayı gerektirir.*/
  | 'needs_provision'
  | 'exited'
  /** Kullanıcı oluşturmayı onayladı, ancak backend'in oluşturacak yeri yok (agent için sandbox backend'i yapılandırılmamış).*/
  | 'no_sandbox'
  | 'unsupported'
  | 'idle'
  | 'unauthorized'
  | 'error'

export type SandboxTerminalControlFrame = {
  type: string
  code?: string
  message?: string
  pty_id?: number
  backend?: string
  exit_code?: number | null
  cols?: number
  rows?: number
}

const RECONNECT_BASE_DELAY_MS = 1000
const RECONNECT_MAX_DELAY_MS = 30000
const APP_PING_INTERVAL_MS = 25000
const PENDING_OUTPUT_MAX_BYTES = 1024 * 1024
// Sunucu, tek bir girdi karesi için 4 KiB sınır koyar (terminalMaxInputBytes); gorilla sınır aşıldığında
// bağlantıyı doğrudan kapatır; xterm ise yapıştırılan metnin tamamını onData'ye verdiğinden, bir betik yapıştırmak bağlantıyı düşürür.
// Önce baytlara bölüp sonra gönderin: PTY bir bayt akışıdır; çok baytlı bir karakterin ortasından kesilse bile özgün sırayla yeniden birleştirilir.
const INPUT_FRAME_MAX_BYTES = 2048

function resolveWsBase(): string {
  const base = (import.meta.env.BASE_URL || '/').replace(/\/+$/, '')
  const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
  return `${protocol}://${window.location.host}${base}`
}

async function mintTerminalTicket(sessionId: string): Promise<string> {
  const res = await post<{ success?: boolean; data?: { ticket?: string } }>(
    `/api/v1/sessions/${encodeURIComponent(sessionId)}/sandbox/terminal-ticket`,
    {},
  )
  const ticket = res?.data?.ticket
  if (!ticket) {
    throw new Error('missing terminal ticket')
  }
  return ticket
}

/**
 * provisionAttempted, SANDBOX_NOT_BOUND anlamını belirler: oluşturma niyeti yoksa yalnızca "henüz
 * sandbox yok, bir tane oluşturulsun mu?" demektir; niyet varken de başarısız olursa gerçekten oluşturacak yer yoktur.
 */
function statusFromErrorCode(
  code: string | undefined,
  provisionAttempted: boolean,
): SandboxTerminalStatus {
  if (code === 'SANDBOX_NOT_BOUND') {
    return provisionAttempted ? 'no_sandbox' : 'needs_provision'
  }
  if (code === 'SANDBOX_PAUSED') return 'paused'
  if (code === 'TERMINAL_UNSUPPORTED') return 'unsupported'
  if (code === 'IDLE_DISCONNECTED') return 'idle'
  if (code === 'AUTH_REVOKED') return 'unauthorized'
  return 'error'
}

export type SandboxTerminalSession = {
  status: Ref<SandboxTerminalStatus>
  /**
   * SandboxTerminal.vue tarafından enjekte edilir: PTY çıktısı xterm'e yazılır.
   * handler kaydedilmeden önce gelen ikili çerçeveler önce kuyruğa alınır; böylece bash isteminin xterm
   * bağlanmadan önce atılması önlenir. `null` geçirmek, kaldırma sırasında arabelleğe almayı yeniden başlatır.
   */
  onOutput: (handler: ((data: Uint8Array) => void) | null) => void
  /**
   * Bağlanır ve terminali açar; zaten bağlanıyorsa idempotenttir.
   *
   * provision, "bu kullanıcının açık eylemidir; arka ucun sanal alan oluşturmasına veya uyandırmasına izin ver" anlamına gelir. Yalnızca
   * düğmeye tıklanınca true geçirilmelidir: sanal alan oluşturmak/uyandırmak faturalandırılan gerçek altyapıdır. Panel açılırken yapılan
   * otomatik bağlantıda asla gönderilmez: çalışan sanal alan doğrudan bağlanır; duraklatılmış veya henüz oluşturulmamışsa kullanıcı onayına geri dönülür.
   */
  connect: (options?: { provision?: boolean; cols?: number; rows?: number }) => void
  /** Klavye girdisini gönderir (`xterm onData` ham dizesi).*/
  sendInput: (data: string) => void
  /** Terminal boyutunu eşitler; ready sonrasında çağrılır.*/
  resize: (cols: number, rows: number) => void
  /** Bağlantıyı keser ve yeniden bağlanmayı durdurur (bileşen kaldırılırken / oturum değiştirirken çağrılır).*/
  dispose: () => void
}

/**
 * Üç parametrenin de canlı ref olması gerekir (`toRef(props, …)`); `ref(props.x)` gibi bir
 * anlık görüntü olamazlar: her `openSocket` çağrısı bunları yeniden okur; kullanıcı agent değiştirdikten sonra (paylaşılan kaynak alanı dahil)
 * sonraki bağlantı, yeni agent'in sanal alan yapılandırmasına göre sanal alan oluşturabilir.
 */
export function useSandboxTerminal(
  sessionId: Ref<string>,
  agentId: Ref<string | undefined>,
  agentSourceTenantId: Ref<string | number | null | undefined> = ref(undefined),
): SandboxTerminalSession {
  const status = ref<SandboxTerminalStatus>('connecting')

  let ws: WebSocket | null = null
  let opening = false
  let outputHandler: ((data: Uint8Array) => void) | null = null
  let pendingOutput: Uint8Array[] = []
  let pendingOutputBytes = 0
  let disposed = false
  let reconnectAttempt = 0
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let pingTimer: ReturnType<typeof setInterval> | null = null
  let pendingResize: { cols: number; rows: number } | null = null
  let pendingGeometry: { cols: number; rows: number } | null = null
  // Yenileme / paneli kapatma bu closure'u kaldırır; PID `sessionStorage` içine yazılır, yeniden bağlandığında ancak o zaman `pty_id` taşınabilir.
  let lastPid: number | null = readStoredPtyId(sessionId.value)
  // Geçerli bağlantının oluşturma niyeti taşıyıp taşımadığı. Yalnızca `connect({ provision: true })` bunu true yapar.
  let allowProvision = false

  const textEncoder = new TextEncoder()

  function rememberPid(next: number | null) {
    lastPid = next
    writeStoredPtyId(sessionId.value, next)
  }

  function clearReconnectTimer() {
    if (reconnectTimer !== null) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
  }

  function clearPingTimer() {
    if (pingTimer !== null) {
      clearInterval(pingTimer)
      pingTimer = null
    }
  }

  function scheduleReconnect() {
    if (disposed) return
    clearReconnectTimer()
    // Otomatik yeniden bağlanma geri alınmış bir sanal alanı canlandırmamalıdır: yalnızca kullanıcı tıklaması oluşturma niyeti taşır. Bağlantı kesintisi sırasında
    // sanal alan geri alınırsa, yeniden bağlanma `needs_provision` alır ve yeniden oluşturulup oluşturulmayacağına kullanıcı karar verir.
    allowProvision = false
    const delay = Math.min(
      RECONNECT_BASE_DELAY_MS * 2 ** reconnectAttempt,
      RECONNECT_MAX_DELAY_MS,
    )
    reconnectAttempt += 1
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      void openSocket()
    }, delay)
  }

  async function openSocket() {
    if (disposed || opening || ws) return
    const sid = sessionId.value
    if (!sid) return

    opening = true
    status.value = 'connecting'
    try {
      const ticket = await mintTerminalTicket(sid)
      if (disposed) return
      const query = new URLSearchParams({ ticket })
      if (allowProvision) {
        query.set('provision', '1')
        // `agent_id` yalnızca oluşturmaya izin verildiğinde anlamlıdır: ilk sanal alan oluşturulurken kullanılan yapılandırma kaynağıdır.
        const agent = agentId.value
        if (agent && agent !== 'builtin-quick-answer') {
          query.set('agent_id', agent)
        }
        const sourceTenant = agentSourceTenantId.value
        if (sourceTenant != null && String(sourceTenant).trim() !== '') {
          query.set('agent_source_tenant_id', String(sourceTenant).trim())
        }
      }
      if (lastPid && lastPid > 0) {
        query.set('pty_id', String(lastPid))
      }
      if (pendingGeometry) {
        query.set('cols', String(pendingGeometry.cols))
        query.set('rows', String(pendingGeometry.rows))
      }
      const socket = new WebSocket(
        `${resolveWsBase()}/api/v1/sessions/${encodeURIComponent(sid)}/sandbox/terminal?${query.toString()}`,
      )
      if (disposed) {
        socket.close()
        return
      }
      ws = socket
    } catch {
      ws = null
      if (disposed) return
      status.value = 'error'
      scheduleReconnect()
      return
    } finally {
      opening = false
    }
    if (!ws) return
    ws.binaryType = 'arraybuffer'

    ws.onopen = () => {
      clearPingTimer()
      pingTimer = setInterval(() => {
        if (ws && ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: 'ping' }))
        }
      }, APP_PING_INTERVAL_MS)
    }

    ws.onmessage = (event) => {
      if (typeof event.data === 'string') {
        handleControlFrame(event.data)
        return
      }
      const data =
        event.data instanceof ArrayBuffer ? new Uint8Array(event.data) : new Uint8Array(0)
      deliverOutput(data)
    }

    ws.onclose = (event) => {
      clearPingTimer()
      ws = null
      if (disposed) return
      const reason = typeof event.reason === 'string' ? event.reason : ''
      if (
        event.code === 1008
        || reason === 'SANDBOX_NOT_BOUND'
        || reason === 'SANDBOX_PAUSED'
        || reason === 'TERMINAL_UNSUPPORTED'
        || reason === 'IDLE_DISCONNECTED'
        || reason === 'AUTH_REVOKED'
      ) {
        if (reason) {
          status.value = statusFromErrorCode(reason, allowProvision)
        } else if (status.value === 'ready' || status.value === 'connecting') {
          status.value = 'error'
        }
        // `idle` / `unauthorized` / `paused`: kabuk hâlâ sanal alanın içindedir; yeniden bağlanabilmek için PID korunmalıdır.
        // Sanal alan artık yoksa veya arka uç desteklemiyorsa: PID artık geçersizdir; sonraki Create çağrısının ölü süreçle çakışmaması için temizleyin.
        if (status.value !== 'idle' && status.value !== 'unauthorized' && status.value !== 'paused') {
          rememberPid(null)
        }
        return
      }
      if (
        status.value !== 'needs_provision'
        && status.value !== 'paused'
        && status.value !== 'no_sandbox'
        && status.value !== 'unsupported'
        && status.value !== 'exited'
        && status.value !== 'idle'
        && status.value !== 'unauthorized'
      ) {
        status.value = 'error'
        scheduleReconnect()
      }
    }

    ws.onerror = () => {
      // `onclose` daha sonra tetiklenir; işlemi orada tek noktadan yürütün.
    }
  }

  function handleControlFrame(raw: string) {
    let frame: SandboxTerminalControlFrame
    try {
      frame = JSON.parse(raw)
    } catch {
      return
    }
    switch (frame.type) {
      case 'ready':
        status.value = 'ready'
        rememberPid(typeof frame.pty_id === 'number' ? frame.pty_id : null)
        reconnectAttempt = 0
        // Oluşturma niyeti burada sona erer. Yalnızca `SANDBOX_NOT_BOUND` açıklamak için kullanılır: bağlanmadan önce "henüz
        // sanal alan yok, oluşturulsun mu?" anlamındadır; bağlandıktan sonra tekrar alınırsa kesinlikle "sanal alan geri alındı" demektir. Sıfırlanmaması
        // geri almayı `no_sandbox` ("agent için arka uç yapılandırılmamış") diye yanlış yorumlar; metin gerçek nedenle ilgisiz olur.
        allowProvision = false
        if (pendingResize) {
          sendResize(pendingResize.cols, pendingResize.rows)
          pendingResize = null
        }
        break
      case 'exited': {
        status.value = 'exited'
        rememberPid(null)
        break
      }
      case 'error': {
        status.value = statusFromErrorCode(frame.code, allowProvision)
        if (status.value !== 'idle' && status.value !== 'unauthorized' && status.value !== 'paused') {
          rememberPid(null)
        }
        break
      }
      default:
        break
    }
  }

  function deliverOutput(data: Uint8Array) {
    if (data.length === 0) return
    if (outputHandler) {
      outputHandler(data)
      return
    }
    pendingOutput.push(data)
    pendingOutputBytes += data.length
    while (pendingOutputBytes > PENDING_OUTPUT_MAX_BYTES && pendingOutput.length > 0) {
      const dropped = pendingOutput.shift()
      if (dropped) pendingOutputBytes -= dropped.length
    }
  }

  function sendRaw(frame: SandboxTerminalControlFrame) {
    ws?.send(JSON.stringify(frame))
  }

  function sendResize(cols: number, rows: number) {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      pendingResize = { cols, rows }
      return
    }
    sendRaw({ type: 'resize', cols, rows })
  }

  const session: SandboxTerminalSession = {
    status,
    onOutput(handler) {
      outputHandler = handler
      if (!handler) return
      const queued = pendingOutput
      pendingOutput = []
      pendingOutputBytes = 0
      for (const chunk of queued) {
        handler(chunk)
      }
    },
    connect(options) {
      if (disposed || ws || opening) return
      // Bekleyen otomatik yeniden bağlanmayı iptal edin; aksi hâlde daha sonra `provision=false` ile tekrar arama yapar
      // ve kullanıcının bu tıklamasıyla bağlantı için yarışır.
      clearReconnectTimer()
      allowProvision = options?.provision === true
      reconnectAttempt = 0
      const cols = options?.cols
      const rows = options?.rows
      pendingGeometry =
        cols && rows && cols > 0 && rows > 0 ? { cols, rows } : null
      void openSocket()
    },
    sendInput(data) {
      if (!ws || ws.readyState !== WebSocket.OPEN) return
      const bytes = textEncoder.encode(data)
      for (let offset = 0; offset < bytes.length; offset += INPUT_FRAME_MAX_BYTES) {
        ws.send(bytes.subarray(offset, offset + INPUT_FRAME_MAX_BYTES))
      }
    },
    resize: sendResize,
    dispose() {
      disposed = true
      clearReconnectTimer()
      clearPingTimer()
      if (ws) {
        const socket = ws
        ws = null
        socket.onclose = null
        socket.onerror = null
        socket.onmessage = null
        socket.close()
      }
    },
  }

  onUnmounted(() => session.dispose())

  return session
}
