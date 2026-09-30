// src/utils/request.js
import axios from "axios";
import { generateRandomString, MAX_FILE_SIZE_MB, MAX_SKILL_BUNDLE_SIZE_MB } from "./index";
import i18n from '@/i18n'
import { getApiBaseUrl } from './api-base';
import { isSkillBundleUploadUrl } from './uploadLimit';
import { isTimeoutError, uploadTimeoutMs } from './requestTimeouts';
import {
  forceReloginRedirect,
  isEmbedPage,
  refreshAccessTokenShared,
} from './authRefresh';

export { forceReloginRedirect, refreshAccessTokenShared };

const t = (key: string) => i18n.global.t(key)

// API temel URL'si
const BASE_URL = getApiBaseUrl();

/**
 * Response payload augmented with the HTTP status code.
 *
 * `$httpStatus` lets callers distinguish outcomes that share a success shape.
 * Defined as a non-enumerable property, so it stays invisible to object spread,
 * JSON.stringify and Object.keys and never leaks into downstream payloads.
 *
 * Guaranteed only for JSON responses (objects/arrays). Blob, string and SSE
 * stream responses do not carry it at runtime, so only read `$httpStatus`
 * when the payload is known to be an object.
 */
export type WithStatus<T> = T & {
  /** HTTP status code of the response. Non-enumerable. See {@link WithStatus}. */
  readonly $httpStatus: number
};

const HTTP_STATUS_KEY = '$httpStatus';

/**
 * Attach the non-enumerable `$httpStatus` property to a response payload
 * in place and return it. Primitives pass through untouched.
 * See {@link WithStatus} for where the property is guaranteed.
 */
function withHttpStatus<T>(data: T, status: number): T {
  if (data !== null && typeof data === 'object') {
    Object.defineProperty(data, HTTP_STATUS_KEY, {
      value: status,
      enumerable: false,
      configurable: true,
      writable: false,
    });
  }
  return data;
}

// Axios örneği oluştur
const instance = axios.create({
  baseURL: BASE_URL, // Yapılandırılmış API temel URL'sini kullanır
  timeout: 30000, // İstek zaman aşımı süresi
  headers: {
    "Content-Type": "application/json",
    "X-Request-ID": `${generateRandomString(12)}`,
  },
});

// Mevcut kullanıcı dilini al (`Accept-Language` başlığı için)
export function getCurrentLanguage(): string {
  return i18n.global.locale?.value || 'tr-TR'
}


instance.interceptors.request.use(
  (config) => {
    const existingAuth = config.headers?.Authorization ?? config.headers?.authorization;
    const isEmbedAuth = typeof existingAuth === 'string' && existingAuth.startsWith('Embed ');
    const isEmbedPath = typeof config.url === 'string' && config.url.includes('/api/v1/embed/');

    // Gömülü kanalda Embed token kullan; yerel JWT ile üzerine yazma (aksi halde hata ayıklama sayfası 401 döner)
    if (!isEmbedAuth) {
      const token = localStorage.getItem('rethra_token');
      if (token) {
        config.headers["Authorization"] = `Bearer ${token}`;
      }
    }
    
    // Kullanıcı dil tercihini ekle
    config.headers["Accept-Language"] = getCurrentLanguage();
    
    // Alanlar arası erişim istek başlığını ekle: `setSelectedTenant` etkin alanı bir kez yazdıysa,
    // her isteğe `X-Tenant-ID` eklenmelidir. İlk sürümler short-circuit yapıyordu
    // başlık boyutunu azaltmak için "`selectedTenantId === defaultTenantId` olduğunda ekleme",
    // ancak bu optimizasyon, `rethra_tenant` değerini etkin alan olarak yazan herhangi bir kod tarafından (OIDC
    // geri çağrısı, `UserMenu loadUserInfo`, router hydrate) tetiklenir ve sonraki isteklerde
    // başlığın sessizce kaybolmasına neden olur; ön yüz "değiştirdiğini" gösterirken gerçekte hâlâ home alanında çalışır — "değişimden sonra yalnızca ilk istek grubu `X-Tenant-ID` taşır" durumunu
    // kalıcı hâle getirir.
    // Arka uçtaki `IsTenantAccessible`, başlığın home alanını (kendi alanını) işaret etmesine zaten izin veriyor,
    // bu nedenle koşulsuz eklemek yeni bir risk oluşturmaz.
    if (!isEmbedAuth && !isEmbedPath) {
      const selectedTenantId = localStorage.getItem('rethra_selected_tenant_id');
      if (selectedTenantId) {
        config.headers["X-Tenant-ID"] = selectedTenantId;
      }
    }
    
    config.headers["X-Request-ID"] = `${generateRandomString(12)}`;
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Share-link endpoints (/auth/invitations/lookup, /auth/register-by-invite)
// are reachable by anonymous users opening an invite link. A 401 from these
// must surface to the page (e.g. expired token), not trigger the
// refresh-then-redirect-to-login flow (issue #1617). '/auth/register' already
// covers '/auth/register-by-invite' via substring match.
const PUBLIC_AUTH_PATHS = ['/auth/login', '/auth/register', '/auth/oidc/', '/auth/invitations/lookup', '/api/v1/embed/'];

function isPublicAuthRequest(url?: string): boolean {
  if (!url) return false;
  return PUBLIC_AUTH_PATHS.some(p => url.includes(p));
}

instance.interceptors.response.use(
  (response) => {
    // İş mantığı durum koduna göre işleme mantığı
    const { status, data } = response;
    if (status >= 200 && status < 300) {
      return withHttpStatus(data, status);
    } else {
      return Promise.reject(withHttpStatus(data, status));
    }
  },
  async (error: any) => {
    const originalRequest = error.config;
    
    if (!error.response) {
      // A timeout and an unreachable server both arrive without a response, but
      // telling someone whose upload timed out to "check your connection" sends
      // them after the wrong problem.
      return Promise.reject({
        message: t(isTimeoutError(error) ? 'error.requestTimeout' : 'error.networkError'),
      });
    }

    // Dosya indirme başarısız olduğunda sunucu yine JSON döndürür; Blob tarafından gizlenmesini önlemek için önce hata bilgisini geri yükle.
    // `Content-Type` bağımlılığı yoktur: ağ geçidi hatayı `text/plain` veya boş tür olarak değiştirebilir.
    if (typeof Blob !== 'undefined' && error.response.data instanceof Blob) {
      try {
        const text = (await error.response.data.text()).trim();
        if (text.startsWith('{') || text.startsWith('[') || error.response.data.type.includes('json')) {
          error.response.data = JSON.parse(text);
        }
      } catch {
        // Geçersiz JSON için mevcut hata işleme kullanılmaya devam edilir.
      }
    }
    
    // Genel arayüzlerin (`login` / `register` / `oidc`) 401 hataları refresh mantığına girmez, hata doğrudan döndürülür
    if ((error.response.status === 401 || error.response.status === 403) && isPublicAuthRequest(originalRequest?.url)) {
      const { status, data } = error.response;
      const msg = typeof data === 'object'
        ? (typeof data?.error === 'string' ? data.error : (data?.error?.message || data?.message))
        : data;
      return Promise.reject(withHttpStatus({ status, message: msg || t('error.invalidCredentials') }, status));
    }

    // Embed hata ayıklama sayfası/bileşeni: JWT yoksa doğrudan reddet, refresh → `/login` yoluna gitme
    if (error.response.status === 401 && isEmbedPage()) {
      const { status, data } = error.response;
      const msg = typeof data === 'object'
        ? (typeof data?.error === 'string' ? data.error : (data?.error?.message || data?.message))
        : data;
      return Promise.reject(withHttpStatus({ status, message: msg || t('error.invalidCredentials') }, status));
    }

    // 401 hatasıysa ve token yenileme isteği değilse token'ı yenilemeyi dene
    if (error.response.status === 401 && !originalRequest._retry && !originalRequest.url?.includes('/auth/refresh')) {
      originalRequest._retry = true;
      try {
        const token = await refreshAccessTokenShared({
          messages: {
            pleaseRelogin: t('error.pleaseRelogin'),
            tokenRefreshFailed: t('error.tokenRefreshFailed'),
          },
        });
        originalRequest.headers['Authorization'] = 'Bearer ' + token;
        return instance(originalRequest);
      } catch (refreshError) {
        // refreshAccessTokenShared already cleared credentials and redirected.
        return Promise.reject(refreshError);
      }
    }
    
    // Nginx 413 `Request Entity Too Large` durumunu işle
    const ERR_ENTITY_TOO_LARGE = 413;
    if (error.response.status === ERR_ENTITY_TOO_LARGE) {
      const skillUpload = isSkillBundleUploadUrl(error.config?.url)
      return Promise.reject(withHttpStatus({
        status: ERR_ENTITY_TOO_LARGE,
        message: skillUpload
          ? i18n.global.t('settings.sandbox.skillBundleTooLarge', { size: MAX_SKILL_BUNDLE_SIZE_MB })
          : i18n.global.t('error.fileSizeExceeded', { size: MAX_FILE_SIZE_MB }),
        success: false
      }, ERR_ENTITY_TOO_LARGE));
    }

    const { status, data } = error.response;
    // HTTP durum kodunu da fırlat; üst katmanın 401 gibi durumları değerlendirmesini kolaylaştır
    // Arka uç dönüş biçimi: { success: false, error: { code, message, details } }
    // Ön yüzün error?.message ile almasını kolaylaştırmak için error.message değerini en üst düzey message olarak çıkar.
    let errorMessage: string | undefined;
    if (typeof data === 'object') {
      if (typeof data?.error === 'string') {
        errorMessage = data.error;
      } else if (data?.error?.message) {
        errorMessage = data.error.message;
      } else {
        errorMessage = data?.message;
      }
    } else if (typeof data === 'string') {
      errorMessage = data;
    }
    return Promise.reject(withHttpStatus({
      status,
      message: errorMessage,
      ...(typeof data === 'object' ? data : {}) 
    }, status));
  }
);

export function get<T = any>(url: string, config?: any): Promise<WithStatus<T>> {
  return instance.get<T>(url, config) as unknown as Promise<WithStatus<T>>;
}

export async function getDown(url: string): Promise<Blob> {
  const res = await instance.get<Blob>(url, {
    responseType: "blob",
  }) as unknown as Blob;
  return res
}

export function postUpload(
  url: string,
  data = {},
  onUploadProgress?: (progressEvent: any) => void,
  config: any = {},
): Promise<WithStatus<any>> {
  return instance.post(url, data, {
    // Uploads are bounded by transfer time, not by the 30s default that suits
    // JSON calls. Derive the budget from the payload so a deployment raising
    // MAX_FILE_SIZE_MB doesn't silently abort its own uploads; an explicit
    // `config.timeout` still wins.
    timeout: uploadTimeoutMs(data),
    ...config,
    headers: {
      "Content-Type": "multipart/form-data",
      "X-Request-ID": `${generateRandomString(12)}`,
      ...(config.headers || {}),
    },
    onUploadProgress: onUploadProgress || config.onUploadProgress,
  }) as unknown as Promise<any>;
}

export function postChat<T = any>(url: string, data = {}): Promise<T> {
  // SSE stream: body is a string, so no `$httpStatus` is attached (see WithStatus).
  return instance.post(url, data, {
    headers: {
      "Content-Type": "text/event-stream;charset=utf-8",
      "X-Request-ID": `${generateRandomString(12)}`,
    },
  }) as unknown as Promise<T>;
}

export function post<T = any>(url: string, data = {}, config?: any): Promise<WithStatus<T>> {
  return instance.post<T>(url, data, config) as unknown as Promise<WithStatus<T>>;
}

export function put<T = any>(url: string, data = {}, config?: any): Promise<WithStatus<T>> {
  return instance.put<T>(url, data, config) as unknown as Promise<WithStatus<T>>;
}

export function patch<T = any>(url: string, data = {}, config?: any): Promise<WithStatus<T>> {
  return instance.patch<T>(url, data, config) as unknown as Promise<WithStatus<T>>;
}

export function del<T = any>(url: string, data?: any): Promise<WithStatus<T>> {
  return instance.delete<T>(url, { data }) as unknown as Promise<WithStatus<T>>;
}
