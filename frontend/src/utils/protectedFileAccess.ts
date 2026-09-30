/**
 * Korunan dosyaların (`provider://` / `resource://` vb.) erişim bağlamı.
 *
 * Arka uç, dosya vekilini erişim öznesine göre ayırır; kimlik doğrulama modelleri birbirinden farklıdır:
 *   - `/files` → oturum Bearer + X-Tenant-ID
 *   - `/api/v1/knowledge-bases/:id/files` → bilgi tabanı erişim izni (kiracılar arası paylaşılan kitaplık)
 *   - `/api/v1/sessions/:id/messages/:mid/files` → oturum mesajı sahipliği + paylaşılan akıllı ajan izni
 *   - `/api/v1/embed/:channel_id/files` → gömülü ziyaretçinin Embed token'ı
 *
 * Hangi vekilin seçileceği, görüntüyü hangi bileşenin işlediğine değil, mevcut isteğin kimlik doğrulama düzlemine bağlıdır.
 * Burada bu karar tek bir doğruluk kaynağında toplanır; işleme bileşenlerinin yalnızca kapsam bildirmesi yeterlidir, artık her biri ayrı ayrı URL oluşturmaz.
 */

export const PROVIDER_SCHEME_PATTERN = 'resource|local|minio|cos|tos|s3|oss|ks3|obs';

const PROVIDER_FILE_SCHEME_RE = new RegExp(`^(${PROVIDER_SCHEME_PATTERN}):\\/\\/\\S+$`, 'i');
const STORAGE_BACKEND_FILE_SCHEME_RE = new RegExp(
  `^storage:\\/\\/[0-9A-Za-z_-]+\\/(${PROVIDER_SCHEME_PATTERN}):\\/\\/\\S+$`,
  'i',
);

const KB_FILE_PROXY_PATH_RE = /^\/api\/v1\/knowledge-bases\/[^/]+\/files$/;
const EMBED_FILE_PROXY_PATH_RE = /^\/api\/v1\/embed\/[^/]+\/files$/;
const MESSAGE_FILE_PROXY_PATH_RE = /^\/api\/v1\/sessions\/[^/]+\/messages\/[^/]+\/files$/;

export type ProtectedFileAccessContext =
  /** Oturum açmış kullanıcı: Bearer + seçili kiracı.*/
  | { mode: 'tenant' }
  /** Gömülü ziyaretçi: yalnızca Embed token taşır; Bearer ve kiracı bağlamı yoktur.*/
  | { mode: 'embed'; channelId: string; token: string }
  /** Bilgi tabanı kapsamı: oturum açmış kullanıcı, paylaşılan kitaplıktaki başka kiracılara ait nesneleri okur.*/
  | { mode: 'knowledgeBase'; kbId: string }
  /** Mesaj kapsamı: oturum açmış kullanıcı, paylaşılan akıllı ajan yanıtlarındaki kaynak alanı kaynaklarını okur.*/
  | { mode: 'message'; sessionId: string; messageId: string };

export interface ProtectedFileRequest {
  url: string;
  headers: Record<string, string>;
}

const TENANT_ACCESS: ProtectedFileAccessContext = { mode: 'tenant' };

interface ProtectedFileAccessState {
  current: ProtectedFileAccessContext;
}

// Blob önbelleği gibi `window` üzerinde tutulur: Vite sıcak güncellemesi modülü değiştirir ancak belgeyi yeniden oluşturmaz,
// modül düzeyindeki değişkenler gömülü uygulama başlatılırken kaydedilen bağlamı kaybeder.
const accessState: ProtectedFileAccessState = (() => {
  const fresh = (): ProtectedFileAccessState => ({ current: TENANT_ACCESS });
  if (typeof window === 'undefined') return fresh();
  const scope = window as typeof window & {
    __rethraProtectedFileAccessV1__?: ProtectedFileAccessState;
  };
  scope.__rethraProtectedFileAccessV1__ ||= fresh();
  return scope.__rethraProtectedFileAccessV1__;
})();

/**
 * Mevcut belge için varsayılan erişim bağlamını kaydeder. Uygulama giriş noktası tarafından bir kez çağrılır (gömülü uygulama
 * channelId/token alındıktan sonra kaydeder); bundan sonra tüm korumalı dosya istekleri otomatik olarak ilgili vekilden geçer.
 */
export function setDefaultProtectedFileAccess(
  access: ProtectedFileAccessContext | null,
): void {
  accessState.current = access ?? TENANT_ACCESS;
}

export function getDefaultProtectedFileAccess(): ProtectedFileAccessContext {
  return accessState.current;
}

/**
 * Varsayılan bağlamı ve bileşen tarafından aktarılan kapsamı birleştirir.
 *
 * Varsayılan bağlam kimlik doğrulama düzlemini taşır (gömülü ziyaretçi veya oturum açmış kullanıcı); bileşen düzeyindeki override yalnızca
 * aynı düzlem içinde kapsamı ayrıntılandırabilir. Gömülü ziyaretçinin Bearer'ı yoktur; `knowledgeBase` override'ının
 * embed düzlemini ezmesine izin verilirse istek oturum gerektiren vekile gider ve 401 döner.
 */
export function resolveProtectedFileAccess(
  override?: ProtectedFileAccessContext | null,
): ProtectedFileAccessContext {
  const fallback = accessState.current;
  if (fallback.mode === 'embed') return fallback;
  if (!override) return fallback;
  if (override.mode === 'knowledgeBase' && !override.kbId.trim()) return fallback;
  if (override.mode === 'message' && (!override.sessionId.trim() || !override.messageId.trim())) {
    return fallback;
  }
  return override;
}

/** Vekil üzerinden alınması gereken bir depolama yolu olup olmadığı (`provider://` veya `storage://<backend>/provider://`).*/
export function isProviderFileURL(url: string): boolean {
  const trimmed = url.trim();
  return PROVIDER_FILE_SCHEME_RE.test(trimmed) || STORAGE_BACKEND_FILE_SCHEME_RE.test(trimmed);
}

/** Korumalı dosya vekillerinden birinin yolu olup olmadığı.*/
export function isProtectedFileProxyPath(pathname: string): boolean {
  return (
    pathname === '/files'
    || KB_FILE_PROXY_PATH_RE.test(pathname)
    || MESSAGE_FILE_PROXY_PATH_RE.test(pathname)
    || EMBED_FILE_PROXY_PATH_RE.test(pathname)
  );
}

function tenantRequestHeaders(): Record<string, string> {
  const headers: Record<string, string> = {};
  try {
    const token = (localStorage.getItem('rethra_token') || '').trim();
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const selectedTenantId = (localStorage.getItem('rethra_selected_tenant_id') || '').trim();
    if (selectedTenantId) {
      // Always attach when a selected tenant is set. Same rationale as
      // utils/request.ts / api/chat/streame.ts: the
      // "selectedTenantId === defaultTenantId → skip" short-circuit
      // silently drops the header whenever any code path writes the
      // active tenant into rethra_tenant, leaving authenticated file
      // fetches landing on the home tenant.
      headers['X-Tenant-ID'] = selectedTenantId;
    }
  } catch {
    // ignore localStorage read errors
  }
  return headers;
}

/**
 * Bir depolama yolu için vekil isteği oluşturur. null dönmesi, mevcut bağlamın bu isteği başlatamadığını belirtir
 * (depolama yolu değildir veya gömülü bağlam henüz token'ı almamıştır); çağıran taraf atlamalı ve daha sonra yeniden denemelidir.
 */
export function buildProtectedFileRequest(
  sourceURL: string,
  access: ProtectedFileAccessContext,
): ProtectedFileRequest | null {
  const filePath = sourceURL.trim();
  if (!isProviderFileURL(filePath)) return null;

  const query = new URLSearchParams({ file_path: filePath }).toString();

  if (access.mode === 'embed') {
    const channelId = access.channelId.trim();
    const token = access.token.trim();
    // Gömülü ziyaretçinin tek kimlik bilgisi Embed token'dır; eksikken /files yoluna geri dönmek yalnızca 401 verir,
    // bootstrap tamamlandıktan sonraki bir sonraki hydration işlemini beklemek için atlamak daha iyidir.
    if (!channelId || !token) return null;
    return {
      url: `/api/v1/embed/${encodeURIComponent(channelId)}/files?${query}`,
      headers: { Authorization: `Embed ${token}` },
    };
  }

  if (access.mode === 'knowledgeBase') {
    return {
      url: `/api/v1/knowledge-bases/${encodeURIComponent(access.kbId.trim())}/files?${query}`,
      headers: tenantRequestHeaders(),
    };
  }

  if (access.mode === 'message') {
    return {
      url: `/api/v1/sessions/${encodeURIComponent(access.sessionId.trim())}/messages/${encodeURIComponent(access.messageId.trim())}/files?${query}`,
      headers: tenantRequestHeaders(),
    };
  }

  return { url: `/files?${query}`, headers: tenantRequestHeaders() };
}
