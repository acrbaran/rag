// Tenant-switch navigation helper.
//
// Switching the active tenant always lands the user on the platform's KB
// list. Önceden «geçerli yolda reload» uygulanıyor ve bazı hassas yollar KB listesine geri dönüyordu; ancak
// resource id içermeyen sayfalar (ayarlar, Agent listesi vb.) bile reload sonrasında yeni alanda
// karşılık gelen veri bulunmadığından çoğu zaman boş durum gösteriyordu; deneyim sabit ana sayfaya gitmekten pek farklı değildi — bu yüzden doğrudan KB listesine yönlendirme standardize edildi,
// tek bir full navigation ile tüm store / SSE / istekleri baştan sıfırlamak için.

import { updateMyPreferences } from '@/api/auth'

const SAFE_FALLBACK_PATH = '/platform/knowledge-bases'

/**
 * Tenant değişiminden sonra gidilecek URL'yi döndürür. Şu anda her zaman KB listesini
 * giriş sayfası olarak döndürür; işlev imzası, gelecekte rotaya göre özel işlem gerektiğinde bir genişletme noktası bırakmak için korunur.
 */
export function tenantSwitchTargetPath(_currentPath: string): string {
  return SAFE_FALLBACK_PATH
}

/**
 * Değişim sonrası yönlendirmeyi gerçekleştirir. Her zaman KB listesine yönlendirir.
 */
export function navigateAfterTenantSwitch(): void {
  window.location.href = tenantSwitchTargetPath(window.location.pathname)
}

// Geçiş başarılı olduktan sonra toast mesajını hard reload üzerinden taşımak için: çağıran taraf reload öncesinde bilgiyi şuraya koyar
// `sessionStorage`; `App.vue` başlatılırken bunu bir kez consume eder ve ardından gösterir. Reload öncesinde doğrudan
// `NotifyPlugin` çağrılırsa silinir; okunacak zaman bile kalmaz.
const PENDING_TOAST_KEY = 'rethra_pending_tenant_switch_toast'

export interface PendingTenantSwitchToast {
  name: string
  /** İnsan tarafından okunabilir rol etiketi; ör. "Sahip" / "Owner". Rol rozetinin metnini belirler.*/
  role?: string
  /** Raw role enum, e.g. "owner". Drives the role chip color + icon. */
  roleEnum?: string
}

export function stashTenantSwitchToast(payload: PendingTenantSwitchToast): void {
  try {
    sessionStorage.setItem(PENDING_TOAST_KEY, JSON.stringify(payload))
  } catch {
    // `sessionStorage` yazımı başarısız olursa (gizli mod vb.) sessizce vazgeçilir; toast yalnızca ek bir kolaylıktır.
  }
}

export function consumePendingTenantSwitchToast(): PendingTenantSwitchToast | null {
  try {
    const raw = sessionStorage.getItem(PENDING_TOAST_KEY)
    if (!raw) return null
    sessionStorage.removeItem(PENDING_TOAST_KEY)
    const parsed = JSON.parse(raw)
    if (parsed && typeof parsed.name === 'string') {
      return {
        name: parsed.name,
        role: typeof parsed.role === 'string' ? parsed.role : undefined,
        roleEnum: typeof parsed.roleEnum === 'string' ? parsed.roleEnum : undefined,
      }
    }
    return null
  } catch {
    return null
  }
}

/**
 * Persist the user's "last active tenant" preference server-side so a
 * fresh login (new device, new refresh token, cleared browser) drops
 * them back into this workspace instead of always bouncing to their
 * home tenant.
 *
 * Conventions:
 *   - Pass the target tenant id when switching to a peer tenant.
 *   - Pass `null` (which sends `0`) when switching back to the home
 *     tenant — that clears the preference and reverts the user to the
 *     "always start at home" default.
 *
 * Fire-and-forget: callers usually trigger a full-page reload right
 * after; this returns the in-flight promise so callers can race it
 * against a small budget if they want best-effort completion before
 * navigation. Failures are logged but never thrown — losing one
 * persist is recoverable on the next switch.
 */
export function persistLastActiveTenantPreference(
  tenantId: number | null,
): Promise<void> {
  const payload = { last_active_tenant_id: tenantId == null ? 0 : tenantId }
  return updateMyPreferences(payload)
    .then((res) => {
      if (!res.success) {
        console.warn('persistLastActiveTenantPreference: server rejected update', res.message)
      }
    })
    .catch((err) => {
      console.warn('persistLastActiveTenantPreference: network/serialization error', err)
    })
}
