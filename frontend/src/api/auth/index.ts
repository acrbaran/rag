import { post, get, put } from '@/utils/request'
import i18n from '@/i18n'

const t = (key: string) => i18n.global.t(key)

// Kullanıcı giriş API'si
export interface LoginRequest {
  email: string
  password: string
}

export interface LoginResponse {
  success: boolean
  message?: string
  user?: {
    id: string
    username: string
    first_name?: string
    last_name?: string
    phone?: string
    email: string
    avatar?: string
    tenant_id: number
    can_access_all_tenants?: boolean
    is_system_admin?: boolean
    is_active: boolean
    created_at: string
    updated_at: string
  }
  tenant?: {
    id: number
    name: string
    description: string
    status: string
    business: string
    storage_quota: number
    storage_used: number
    created_at: string
    updated_at: string
  } | null
  // active_tenant mirrors `tenant` for endpoints that distinguish home
  // tenant from current tenant (e.g. /auth/register-by-invite). Only
  // one of `tenant` / `active_tenant` is populated by any given endpoint.
  active_tenant?: {
    id: number
    name: string
    description?: string
    status?: string
    business?: string
    storage_quota?: number
    storage_used?: number
    created_at?: string
    updated_at?: string
  } | null
  memberships?: MembershipInfo[]
  token?: string
  refresh_token?: string
}

export interface OIDCAuthURLResponse {
  success: boolean
  authorization_url?: string
  state?: string
  message?: string
}

export interface OIDCConfigResponse {
  success: boolean
  enabled: boolean
  provider_display_name?: string
  message?: string
}

// Kullanıcı kayıt API'si
export interface RegisterRequest {
  username: string
  first_name: string
  last_name: string
  phone: string
  email: string
  password: string
}

export interface RegisterResponse {
  success: boolean
  message?: string
  data?: {
    user: {
      id: string
      username: string
      email: string
    }
    tenant: {
      id: string
      name: string
    }
  }
}

// Kullanıcı tercihleri (arka uç types.UserPreferences ile uyumlu; alanların isteğe bağlı olması = açıkça ayarlanmamış olmaları).
// Yeni key eklerken unutmayın: arka uç service.UpdateUserPreferences bunu merge dalında da
// işlemelidir; ön uç çağıranlar gerektiğinde okur / varsayılan değere geri döner.
export interface UserPreferences {
  browser_search_instructions?: string | null
  // last_active_tenant_id, «yenileme / cihaz değiştirme / yeniden girişten sonra son alana dön» davranışını kalıcı kılar
  // ; arka uç Login / RefreshToken sırasında membership geçerliliğini doğruladıktan sonra bunu kullanmaya devam eder,
  // aksi halde home'a döner ve bu alan temizlenir. PATCH'e 0 göndermek «tercihi temizle» anlamına gelir.
  last_active_tenant_id?: number | null
  // oidc_only_login değerinin true olması, hesabın OIDC tarafından otomatik açıldığı ve kullanıcının henüz bilinen bir parola belirlemediği anlamına gelir.
  oidc_only_login?: boolean
  // gallery, görsel kitaplığının kişisel durumunu kaydeder: arama etkin modu ve alan bazında açma/kapama (özellik kimliğine göre).
  // Her değişiklikte bu blok tamamen üzerine yazılır; arka uç mode ve status değerlerini doğrular.
  gallery?: {
    mode?: 'all' | 'custom'
    status?: Record<string, string>
  }
}

// Kullanıcı bilgisi API'si
export interface UserInfo {
  id: string
  username: string
  first_name?: string
  last_name?: string
  phone?: string
  email: string
  avatar?: string
  tenant_id: string
  can_access_all_tenants?: boolean
  preferences?: UserPreferences
  is_system_admin?: boolean
  created_at: string
  updated_at: string
}

/**
 * Arka ucun döndürdüğü user JSON'unu ön uç UserInfo biçimine dönüştürür.
 *
 * Geçmişte birden fazla bağımsız setUser çağrısı kendi alan beyaz listesini elle yazıyordu; her yeni user alanında
 * eşzamanlama gerekiyordu — aksi hâlde alan sessizce filtreleniyordu. is_system_admin kullanıma alındığında,
 * bir yerde kopyalama unutulduğu için 「Sistem Yönetimi」 girişi görünmüyordu; bu fabrikanın amacı benzer
 * kopyalama unutmalarının tekrar yaşanmasını önlemektir. **Yeni user alanlarını yalnızca burada değiştirin**.
 *
 * fallbackTenantId, tenant_id eksik olduğunda kullanılan yedek kaynaktır —
 *   - /auth/me ara sıra tenant olmadan yalnızca user döndürdüğünde de yedek kullanılır
 * Çağıran, gerektiğinde iletir; iletilmezse boş dize korunur (geçmiş davranışla uyumlu).
 *
 * Alan okumaları `=== true` ile yapılır, `|| false` ile değil; ara sıra boolean olmayan
 * türler (arka uç bir gün 1/0 veya dize gönderirse) sıkı biçimde sınırlandırılır, böylece truthy dizelerin
 * yanlışlıkla yetki verilmiş sayılması önlenir.
 */
export function userInfoFromApi(
  u: any,
  fallbackTenantId?: string | number | null,
): UserInfo {
  const rawTenantId =
    u?.tenant_id !== undefined && u?.tenant_id !== null && u.tenant_id !== ''
      ? u.tenant_id
      : fallbackTenantId ?? ''
  const tid = Number(rawTenantId) > 0 ? rawTenantId : ''
  return {
    id: u?.id || '',
    username: u?.username || '',
    first_name: u?.first_name || '',
    last_name: u?.last_name || '',
    phone: u?.phone || '',
    email: u?.email || '',
    avatar: u?.avatar,
    tenant_id: String(tid) || '',
    can_access_all_tenants: u?.can_access_all_tenants === true,
    is_system_admin: u?.is_system_admin === true,
    preferences: u?.preferences,
    created_at: u?.created_at || new Date().toISOString(),
    updated_at: u?.updated_at || new Date().toISOString(),
  }
}

// Alan bilgisi API'si
export interface TenantInfo {
  id: string
  name: string
  description?: string
  status?: string
  business?: string
  owner_id: string
  storage_quota?: number
  storage_used?: number
  created_at: string
  updated_at: string
  knowledge_bases?: KnowledgeBaseInfo[]
}

// Bilgi tabanı bilgisi API'si
export interface KnowledgeBaseInfo {
  id: string
  name: string
  description: string
  tenant_id: string
  // creator_id is the user id of whoever originally created the KB.
  // Set by PR 5 of the multi-tenant RBAC series; nullable for legacy
  // KBs created before that migration backfilled the column.
  creator_id?: string
  // creator_name, arka uç list API'si tarafından toplu olarak doldurulur (önce username, yoksa email),
  // yalnızca liste kartı kaynak rozeti için kullanılır; eksik olması çözümlenemediğini gösterir (silinmiş / eski veri).
  creator_name?: string
  created_at: string
  updated_at: string
  document_count?: number
  chunk_count?: number
}

// Model bilgisi API'si
export interface ModelInfo {
  id: string
  name: string
  type: string
  source: string
  description?: string
  is_default?: boolean
  created_at: string
  updated_at: string
}

/**
 * Kullanıcı girişi
 */
export async function login(data: LoginRequest): Promise<LoginResponse> {
  try {
    const response = await post('/api/v1/auth/login', data)
    return response as unknown as LoginResponse
  } catch (error: any) {
    return {
      success: false,
      message: error.message || t('error.auth.loginFailed')
    }
  }
}

/**
 * OIDC giriş yönlendirme adresini al
 */
export async function getOIDCAuthorizationURL(redirectURI: string): Promise<OIDCAuthURLResponse> {
  try {
    const response = await get(`/api/v1/auth/oidc/url?redirect_uri=${encodeURIComponent(redirectURI)}`)
    return response as unknown as OIDCAuthURLResponse
  } catch (error: any) {
    return {
      success: false,
      message: error.message || t('error.auth.loginFailed')
    }
  }
}

/**
 * OIDC giriş yapılandırmasını al
 */
export async function getOIDCConfig(): Promise<OIDCConfigResponse> {
  try {
    const response = await get('/api/v1/auth/oidc/config')
    return response as unknown as OIDCConfigResponse
  } catch (error: any) {
    return {
      success: false,
      enabled: false,
      message: error.message || t('error.auth.loginFailed')
    }
  }
}

/**
 * Kimlik doğrulama yapılandırmasını al (yalnızca ön uç oluşturma için gereken herkese açık alanları, örneğin kayıt modunu döndürür).
 *
 * Arka uç, kendi kendine kayda izin verilip verilmediğini `auth.registration_mode` ile kontrol eder:
 *   - "self_serve"  mevcut kendi kendine kayıt girişini korur (varsayılan)
 *   - "invite_only" kaydı kapatır, yönetici daveti gerektirir
 *
 * Başarısızlık durumunda self_serve'e geri dönerek API hatasının kayıt girişini doğrudan ortadan kaldırmasını önler.
 */
export interface AuthConfigResponse {
  success: boolean
  registration_mode: 'self_serve' | 'invite_only' | string
  complex_password_enabled: boolean
}

export async function getAuthConfig(): Promise<AuthConfigResponse> {
  try {
    const response = await get('/api/v1/auth/config')
    return response as unknown as AuthConfigResponse
  } catch {
    return { success: false, registration_mode: 'self_serve', complex_password_enabled: false }
  }
}

/**
 * Kullanıcı kaydı
 */
export async function register(data: RegisterRequest): Promise<RegisterResponse> {
  try {
    const response = await post('/api/v1/auth/register', data)
    return response as unknown as RegisterResponse
  } catch (error: any) {
    return {
      success: false,
      message: error.message || t('error.auth.registerFailed')
    }
  }
}

/**
 * Membership row returned alongside /auth/me. Mirrors the LoginResponse
 * shape so the frontend can refresh `currentTenantRole` on every page
 * load — without it, role changes after login (e.g. an Owner demoting
 * us in a peer tenant) stay invisible until the user logs out and back
 * in.
 */
export interface MembershipInfo {
  tenant_id: number
  tenant_name?: string
  role: string
}

/**
 * Mevcut kullanıcı bilgilerini al
 */
export interface AuthCapabilities {
  can_create_tenant: boolean
  auto_accept_invitation: boolean
}

export async function getCurrentUser(): Promise<{ success: boolean; data?: { user: UserInfo; tenant?: TenantInfo | null; memberships?: MembershipInfo[]; tenant_required?: boolean; capabilities?: AuthCapabilities; preference_defaults?: { browser_search_instructions: string } }; message?: string }> {
  try {
    const response = await get('/api/v1/auth/me')
    return response as unknown as { success: boolean; data?: { user: UserInfo; tenant?: TenantInfo | null; memberships?: MembershipInfo[]; tenant_required?: boolean; capabilities?: AuthCapabilities; preference_defaults?: { browser_search_instructions: string } }; message?: string }
  } catch (error: any) {
    return {
      success: false,
      message: error.message || t('error.auth.getUserFailed')
    }
  }
}

/**
 * Mevcut kullanıcının tercihlerini güncelle (PATCH anlamı: yalnızca değiştirilecek alanları gönder, arka uç yalnızca gönderilen key'lerin üzerine yazar,
 * Diğer key'ler değişmeden kalır). Arka uç güncellenmiş tam preferences nesnesini döndürür.
 */
export async function updateMyPreferences(
  patch: Partial<UserPreferences>,
): Promise<{ success: boolean; data?: UserPreferences; message?: string }> {
  try {
    const response = await put('/api/v1/auth/me/preferences', patch)
    return response as unknown as { success: boolean; data?: UserPreferences; message?: string }
  } catch (error: any) {
    return {
      success: false,
      message: error.message || t('error.auth.updatePreferencesFailed'),
    }
  }
}

/**
 * Mevcut alan bilgilerini al
 */
export async function updateMyProfile(profile: {
  first_name: string
  last_name: string
  phone: string
}): Promise<{ success: boolean; data?: { user: any }; message?: string }> {
  try {
    const response = await put('/api/v1/auth/me/profile', profile)
    return response as unknown as { success: boolean; data?: { user: any }; message?: string }
  } catch (error: any) {
    return {
      success: false,
      message: error.message || t('userProfile.personal.failed'),
    }
  }
}

/**
 * Mevcut alan bilgilerini al
 */
export async function getCurrentTenant(): Promise<{ success: boolean; data?: TenantInfo; message?: string }> {
  try {
    const response = await get('/api/v1/auth/tenant')
    return response as unknown as { success: boolean; data?: TenantInfo; message?: string }
  } catch (error: any) {
    return {
      success: false,
      message: error.message || t('error.auth.getTenantFailed')
    }
  }
}

/**
 * Token'ı yenile
 */
export async function refreshToken(refreshToken: string): Promise<{ success: boolean; data?: { token: string; refreshToken: string }; message?: string }> {
  try {
    const response: any = await post('/api/v1/auth/refresh', { refreshToken })
    if (response && response.success) {
      if (response.access_token || response.refresh_token) {
        return {
          success: true,
          data: {
            token: response.access_token,
            refreshToken: response.refresh_token,
          }
        }
      }
    }

    // Diğer durumlarda doğrudan ham mesajı döndür
    return {
      success: false,
      message: response?.message || t('error.auth.refreshTokenFailed')
    }
  } catch (error: any) {
    return {
      success: false,
      message: error.message || t('error.auth.refreshTokenFailed')
    }
  }
}

/**
 * Kullanıcı çıkışı
 */
export async function logout(): Promise<{ success: boolean; message?: string }> {
  try {
    await post('/api/v1/auth/logout', {})
    return {
      success: true
    }
  } catch (error: any) {
    return {
      success: false,
      message: error.message || t('error.auth.logoutFailed')
    }
  }
}

/**
 * Presence heartbeat: keeps an open tab counted as online in the system-admin
 * users table even when the user is not otherwise making requests.
 */
export async function sendPresenceHeartbeat(): Promise<void> {
  await post('/api/v1/auth/heartbeat', {})
}

export interface ChangePasswordRequest {
  old_password: string
  new_password: string
}

/** Map change-password API failures to localized UI strings. */
export function resolveChangePasswordError(error: any): string {
  const details =
    typeof error?.error?.details === 'string'
      ? error.error.details
      : typeof error?.details === 'string'
        ? error.details
        : ''
  switch (details) {
    case 'invalid_old_password':
      return t('userProfile.changePassword.failed')
    case 'password_policy':
      return t('userProfile.changePassword.policyFailed')
    case 'same_password':
      return t('userProfile.changePassword.sameAsCurrent')
    default:
      return error?.message || t('userProfile.changePassword.failed')
  }
}

/**
 * Self-service password rotation. On success the backend revokes every
 * outstanding session for the caller, so the client should clear local
 * auth state and send the user back to /login.
 */
export async function changePassword(
  data: ChangePasswordRequest,
): Promise<{ success: boolean; message?: string }> {
  try {
    const response = await post('/api/v1/auth/change-password', data)
    return response as unknown as { success: boolean; message?: string }
  } catch (error: any) {
    return {
      success: false,
      message: resolveChangePasswordError(error),
    }
  }
}

/**
 * Token geçerliliğini doğrula
 */
export async function validateToken(): Promise<{ success: boolean; valid?: boolean; message?: string }> {
  try {
    const response = await get('/api/v1/auth/validate')
    return response as unknown as { success: boolean; valid?: boolean; message?: string }
  } catch (error: any) {
    return {
      success: false,
      valid: false,
      message: error.message || t('error.auth.validateTokenFailed')
    }
  }
}





// ---- share-link registration --------------------------------------------

// InviteLookup is the public projection of a share-link row used by
// /register?token=xxx — enough to render the registration page header
// ("X invited you to Y") without leaking sensitive inviter fields.
export interface InviteLookup {
  tenant_id: number
  tenant_name?: string
  role: string
  expires_at: string
}

export interface InviteLookupResponse {
  success: boolean
  data?: InviteLookup
  message?: string
}

export interface RegisterByInviteRequest {
  token: string
  email: string
  username: string
  first_name: string
  last_name: string
  phone: string
  password: string
}

/**
 * Resolve a share-link token (no auth) into the context the
 * registration page needs (tenant name, role, expiry). Returns 410
 * when the link is invalid / revoked / expired.
 *
 * Uses POST + body (rather than GET + path) so the plaintext token
 * never appears in access logs, browser history, or tracing spans.
 */
export async function getInvitationByToken(token: string): Promise<InviteLookupResponse> {
  try {
    const response = await post(`/api/v1/auth/invitations/lookup`, { token })
    return response as unknown as InviteLookupResponse
  } catch (error: any) {
    return { success: false, message: error.message || '' }
  }
}

/**
 * Complete registration via a share-link token. The invitee supplies
 * their own email — the token is the authorisation, not an identity
 * lock.
 */
export async function registerByInvite(data: RegisterByInviteRequest): Promise<LoginResponse> {
  try {
    const response = await post('/api/v1/auth/register-by-invite', data)
    return response as unknown as LoginResponse
  } catch (error: any) {
    return { success: false, message: error.message || t('error.auth.registerFailed') }
  }
}
