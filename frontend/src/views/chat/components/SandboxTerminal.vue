<template>
    <div ref="containerRef" class="sandbox-terminal" :class="{ 'is-dark': isDarkTheme }"
        @mousedown="focusTerminal">
        <div v-if="status !== 'ready'" class="sandbox-terminal__overlay">
            <div class="sandbox-terminal__overlay-card">
                <t-icon v-if="status === 'connecting'" name="loading" size="24px"
                    class="sandbox-terminal__spinner" />
                <t-icon
                    v-else-if="status === 'paused' || status === 'needs_provision' || status === 'no_sandbox'"
                    name="terminal" size="28px" />
                <t-icon v-else-if="status === 'unsupported'" name="error-circle" size="28px" />
                <t-icon v-else-if="status === 'idle'" name="time" size="28px" />
                <t-icon v-else name="cloud" size="28px" />
                <p class="sandbox-terminal__overlay-text">{{ statusText }}</p>
                <t-button v-if="actionLabel" size="small"
                    :theme="status === 'needs_provision' ? 'primary' : 'default'" variant="outline"
                    @click.stop="start">
                    {{ actionLabel }}
                </t-button>
            </div>
        </div>
        <div ref="terminalHost" class="sandbox-terminal__host"></div>
    </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, toRef, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import { useSandboxTerminal, type SandboxTerminalStatus } from '@/composables/useSandboxTerminal';
import { useTheme } from '@/composables/useTheme';
import { createPtyEchoPredictor } from '@/utils/ptyEchoPredictor';
import {
    PTY_PROMPT_NUDGE_DELAY_MS,
    xtermBufferLooksEmpty,
} from '@/utils/ptyPromptNudge';

const props = defineProps<{
    sessionId: string;
    /** Geçerli oturumda seçili agent: ilk bağlantıda arka uç, yapılandırmasına göre sandbox'ı otomatik oluşturur.*/
    agentId?: string;
    /** Paylaşılan agent'ın kaynak alanı; sohbet isteğindeki agent_source_tenant_id ile aynıdır.*/
    agentSourceTenantId?: string | number | null;
}>();

const { t } = useI18n();

const containerRef = ref<HTMLElement | null>(null);
const terminalHost = ref<HTMLElement | null>(null);

// Tema, useTheme içindeki paylaşılan ref'ten türetilmelidir; `theme-mode` DOM özniteliği okunamaz: DOM öznitelikleri
// reaktif veri kaynağı değildir; bu şekilde yazılan computed'in hiçbir bağımlılığı olmaz, bir kez değerlendirildikten sonra kalıcı olarak önbelleğe alınır,
// aşağıdaki watch asla tetiklenmez ve terminal renkleri ilk bağlanma anındaki görünümde kalır.
const { currentTheme } = useTheme();
const systemPrefersDark = ref(prefersDarkQuery()?.matches === true);
const isDarkTheme = computed(() =>
    currentTheme.value === 'system' ? systemPrefersDark.value : currentTheme.value === 'dark',
);

function prefersDarkQuery(): MediaQueryList | null {
    if (typeof window === 'undefined' || !window.matchMedia) return null;
    return window.matchMedia('(prefers-color-scheme: dark)');
}

// `system` modunda işletim sisteminin anlık değişimlerini de izlemek gerekir; useTheme'in genel dinleyicisi yalnızca DOM
// özniteliğini yazar, etkin açık/koyu değerini dışarı vermez; bu nedenle burada ayrıca bir abonelik oluşturulur.
const prefersDark = prefersDarkQuery();
const onSystemThemeChange = (event: MediaQueryListEvent) => {
    systemPrefersDark.value = event.matches;
};
prefersDark?.addEventListener('change', onSystemThemeChange);

// ANSI palette: ls --color uses the usual dircolors mapping (dir=blue,
// exec=green, link=cyan). Only the green slots stay Rethra brand so
// user@host (01;32) matches the product color; path (01;34) stays blue
// like directories.
function xtermTheme(dark: boolean) {
    return dark
        ? {
            background: '#1a1a1a',
            foreground: '#e6e6e6',
            cursor: '#e6e6e6',
            cursorAccent: '#1a1a1a',
            selectionBackground: '#3a3a3a',
            red: '#c64751',
            brightRed: '#de6670',
            green: '#06b04d',
            brightGreen: '#07c05f',
            yellow: '#c4a000',
            brightYellow: '#fce94f',
            blue: '#3465a4',
            brightBlue: '#729fcf',
            magenta: '#75507b',
            brightMagenta: '#ad7fa8',
            cyan: '#06989a',
            brightCyan: '#34e2e2',
        }
        : {
            background: '#ffffff',
            foreground: '#242424',
            cursor: '#242424',
            cursorAccent: '#ffffff',
            selectionBackground: '#d0d7de',
            red: '#e34d59',
            brightRed: '#f36d78',
            green: '#06b04d',
            brightGreen: '#07c05f',
            yellow: '#c4a000',
            brightYellow: '#c4a000',
            blue: '#3465a4',
            brightBlue: '#729fcf',
            magenta: '#75507b',
            brightMagenta: '#ad7fa8',
            cyan: '#06989a',
            brightCyan: '#34e2e2',
        };
}

// ref(props.x) yerine toRef kullanın: ikincisi anlık görüntüdür; agent değiştirildikten sonra yeniden bağlantı hâlâ eski agent'ın
// sandbox yapılandırmasıyla sandbox oluşturur. sessionId, üst bileşenin :key yeniden oluşturmasıyla korunur; agentId korunmaz.
const terminal = useSandboxTerminal(
    toRef(props, 'sessionId'),
    toRef(props, 'agentId'),
    toRef(props, 'agentSourceTenantId'),
);
const { status } = terminal;

const statusText = computed(() => {
    switch (status.value as SandboxTerminalStatus) {
        case 'paused':
            return t('chat.sandbox.paused');
        case 'connecting':
            return t('chat.sandbox.connecting');
        case 'needs_provision':
            return t('chat.sandbox.needsProvision');
        case 'no_sandbox':
            return t('chat.sandbox.noSandbox');
        case 'unsupported':
            return t('chat.sandbox.unsupported');
        case 'exited':
            return t('chat.sandbox.sessionEnded');
        case 'idle':
            return t('chat.sandbox.idleDisconnected');
        case 'unauthorized':
            return t('chat.sandbox.authRevoked');
        default:
            return t('chat.sandbox.disconnected');
    }
});

// Kaplama düğmesi: duraklatılmışsa "Terminali başlat", henüz oluşturulmamışsa "Oluştur ve başlat" kullanılır. Yalnızca tıklama
// oluşturmaya veya uyandırmaya izin verir; panel açılırken yapılan lookup bu iki işlemi gerçekleştirmez.
const actionLabel = computed(() => {
    switch (status.value as SandboxTerminalStatus) {
        case 'paused':
            return t('chat.sandbox.start');
        case 'needs_provision':
            return t('chat.sandbox.createAndStart');
        case 'error':
        case 'exited':
        case 'idle':
        case 'unauthorized':
        // no_sandbox, "agent için sandbox arka ucu yapılandırılmamış, oluşturulacak yer yok" demektir. Kullanıcı yapılandırdıktan sonra
        // paneli kapatıp yeniden açmak zorunda kalmadan doğrudan yeniden deneyebilmelidir; aksi hâlde hiçbir işlem giriş noktası olmayan bir çıkmaz oluşur.
        case 'no_sandbox':
            return t('chat.sandbox.retry');
        default:
            return '';
    }
});

// xterm örneği yalnızca ready sonrasında bir kez bağlanır; overlay durumunu bunun üzerinde gösterir.
let xterm: Terminal | null = null;
let fitAddon: FitAddon | null = null;
let echo: ReturnType<typeof createPtyEchoPredictor> | null = null;
let resizeObserver: ResizeObserver | null = null;
let resizeDebounce: ReturnType<typeof setTimeout> | null = null;
let promptNudgeTimer: ReturnType<typeof setTimeout> | null = null;
let unmounted = false;

function writeToXterm(chunk: string | Uint8Array) {
    xterm?.write(chunk);
}

function focusTerminal() {
    if (status.value !== 'ready') return;
    xterm?.focus();
}

function mountTerminal() {
    if (!terminalHost.value || xterm) return;
    xterm = new Terminal({
        cursorBlink: true,
        cursorStyle: 'bar',
        cursorInactiveStyle: 'outline',
        fontSize: 13,
        fontFamily: "'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace",
        theme: xtermTheme(isDarkTheme.value),
        scrollback: 5000,
    });
    echo = createPtyEchoPredictor(writeToXterm);
    fitAddon = new FitAddon();
    xterm.loadAddon(fitAddon);
    xterm.open(terminalHost.value);
    xterm.onData((data) => {
        echo?.onLocal(data);
        terminal.sendInput(data);
    });

    resizeObserver = new ResizeObserver(() => {
        if (resizeDebounce) clearTimeout(resizeDebounce);
        resizeDebounce = setTimeout(() => applyFit(), 100);
    });
    resizeObserver.observe(containerRef.value || terminalHost.value);
    // Fit BEFORE flushing buffered PTY bytes. FitAddon.fit() calls
    // _renderService.clear() when the default 80x24 becomes the panel size,
    // which would wipe a prompt painted a moment earlier and leave only the
    // cursor until the next keystroke.
    applyFit();
    terminal.onOutput((data) => echo?.onRemote(data));
    schedulePromptNudge();
    void nextTick(() => {
        requestAnimationFrame(() => fitAndFocus());
    });
}

function unmountTerminal() {
    terminal.onOutput(null);
    echo = null;
    resizeObserver?.disconnect();
    resizeObserver = null;
    if (resizeDebounce) clearTimeout(resizeDebounce);
    resizeDebounce = null;
    if (promptNudgeTimer) clearTimeout(promptNudgeTimer);
    promptNudgeTimer = null;
    xterm?.dispose();
    xterm = null;
    fitAddon = null;
}

// Kaplama üzerindeki sandbox oluşturabilen veya uyandırabilen tek giriş noktası. provision: true bunun kullanıcının açık
// onayı olduğunu belirtir. Bileşen bağlanırken yalnızca lookup yapılır: çalışan sandbox'a doğrudan bağlanılır; duraklatılmış veya henüz oluşturulmamışsa
// kaplamada tıklama beklenir.
function start() {
    unmountTerminal();
    const { cols, rows } = estimatePtySize();
    terminal.connect({ provision: true, cols, rows });
}

function connectLookup() {
    const { cols, rows } = estimatePtySize();
    terminal.connect({ provision: false, cols, rows });
}

onMounted(() => {
    connectLookup();
});

watch(isDarkTheme, (dark) => {
    if (xterm) xterm.options.theme = xtermTheme(dark);
});

// ready sonrasında xterm'i bağla; ready olmayan durumdan çıkınca örneği sıfırla (yeniden bağlantı = yeni PTY).
watch(status, (next, prev) => {
    if (next === 'ready' && prev !== 'ready') {
        requestAnimationFrame(() => {
            // Bileşen ready ile sonraki kare arasında kaldırılmış olabilir (paneli kapatma / oturum değiştirme). Doğrulanmazsa
            // yok edilmiş bileşen üzerinde yeni bir Terminal oluşturulur ve sahipsiz bir düğüm observe edilir; bu
            // örneği watch bile kurtaramaz.
            if (unmounted) return;
            mountTerminal();
        });
    } else if (prev === 'ready' && next !== 'ready') {
        unmountTerminal();
    }
});

// watch(status) yalnızca "durumun ready'den çıkmasını" kapsar; bileşenin kendisinin yok edilmesini kapsamaz: paneli kapatma
// SandboxSidePanel'in v-if'inden, oturum değiştirme ise :key yeniden oluşturmasından geçer; her iki yolda da status baştan sona değişmez,
// watch tetiklenmiyor. Bu kanca olmadan, panel her açılıp kapandığında bir xterm örneği atlanır
// (renderer + 5000 satır scrollback) ve bir ResizeObserver.
onBeforeUnmount(() => {
    unmounted = true;
    prefersDark?.removeEventListener('change', onSystemThemeChange);
    unmountTerminal();
});

function estimatePtySize() {
    const el = containerRef.value;
    const width = Math.max(0, (el?.clientWidth ?? 0) - 16);
    const height = Math.max(0, (el?.clientHeight ?? 0) - 16);
    return {
        cols: Math.max(20, Math.floor(width / 8) || 80),
        rows: Math.max(8, Math.floor(height / 17) || 24),
    };
}

function xtermVisibleBufferEmpty(): boolean {
    if (!xterm) return true;
    const buf = xterm.buffer.active;
    const origin = buf.viewportY;
    return xtermBufferLooksEmpty(
        (row) => buf.getLine(origin + row)?.translateToString(true),
        xterm.rows,
    );
}

// Pty.Connect does not replay a prompt bash already printed. A same-size
// resize is a no-op; flipping rows by 1 sends SIGWINCH so readline (and
// TUIs) redraw. Do not inject Ctrl-L or Enter: that would go to whatever
// is running in a reattached PTY.
function schedulePromptNudge() {
    if (promptNudgeTimer) clearTimeout(promptNudgeTimer);
    promptNudgeTimer = setTimeout(() => {
        promptNudgeTimer = null;
        if (unmounted || !xterm || status.value !== 'ready') return;
        if (!xtermVisibleBufferEmpty()) return;
        const cols = Math.max(2, xterm.cols);
        const rows = Math.max(2, xterm.rows);
        terminal.resize(cols, rows - 1);
        terminal.resize(cols, rows);
    }, PTY_PROMPT_NUDGE_DELAY_MS);
}

// v-show terminali gizlediğinde kapsayıcı 0×0 olur. FitAddon yine de 2×1 hesaplayıp PTY'ye gönderebilir,
// Terminal sekmesine geri dönüldüğünde bash hâlâ bu boyutlarda kalır ve bağlı değilmiş gibi görünür. Boyut yeterli değilse işlem yapma.
function containerHasPtySize() {
    const el = containerRef.value;
    return !!el && el.clientWidth >= 20 && el.clientHeight >= 20;
}

function applyFit() {
    if (!xterm || !containerHasPtySize()) return;
    try {
        fitAddon?.fit();
    } catch {
        // fit, kapsayıcı boyutu 0 olduğunda hata fırlatır; yok saymak yeterlidir.
        return;
    }
    if (xterm.cols < 2 || xterm.rows < 2) return;
    terminal.resize(xterm.cols, xterm.rows);
    xterm.refresh(0, xterm.rows - 1);
}

function fitAndFocus() {
    // v-show henüz açıldığında nextTick içindeki yerleşim tamamlanmamış olabilir; fit için iki kare daha bekle.
    requestAnimationFrame(() => {
        requestAnimationFrame(() => {
            if (unmounted) return;
            applyFit();
            xterm?.focus();
        });
    });
}

defineExpose({
    focus: fitAndFocus,
});
</script>

<style scoped lang="less">
.sandbox-terminal {
    position: relative;
    height: 100%;
    min-height: 0;
    display: flex;
    flex-direction: column;
    background: var(--td-bg-color-container);
    border-radius: var(--app-radius-md);
    overflow: hidden;
    cursor: text;
}

.sandbox-terminal__host {
    flex: 1;
    min-height: 0;
    padding: 8px;
    cursor: text;

    :deep(.xterm) {
        height: 100%;
        cursor: text;
    }

    :deep(.xterm-viewport) {
        overflow-y: auto;
    }

    :deep(.xterm-helper-textarea) {
        // xterm klavye girdisini gizli bir textarea ile alır; imlecin yanıp sönmesi için odak alabilmelidir.
        pointer-events: auto;
    }
}

.sandbox-terminal__overlay {
    position: absolute;
    inset: 0;
    z-index: 2;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--td-bg-color-container);
    cursor: default;
}

.sandbox-terminal__overlay-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 10px;
    max-width: 280px;
    padding: 20px;
    text-align: center;
    color: var(--td-text-color-placeholder);
}

.sandbox-terminal__overlay-text {
    margin: 0;
    font-size: var(--app-text-md);
    line-height: 1.6;
    white-space: pre-line;
}

.sandbox-terminal__spinner {
    animation: wk-spin 0.9s linear infinite;
}

</style>
