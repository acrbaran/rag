import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { UserInfo, TenantInfo, KnowledgeBaseInfo } from '@/api/auth'
import { userInfoFromApi } from '@/api/auth'
import type { TenantInfo as TenantInfoFromAPI } from '@/api/tenant'
import i18n from '@/i18n'
import { reloadFontFromStorage } from '@/composables/useFont'
import { reloadThemeFromStorage } from '@/composables/useTheme'
import { resetMigrationLatch } from '@/composables/preferenceStorage'
import { BUILTIN_QUICK_ANSWER_ID } from '@/api/agent'
import { useChatResourcesStore } from '@/stores/chatResources'
import { useEditorResourcesStore } from '@/stores/editorResources'
import { useOrganizationStore } from '@/stores/organization'
import { createSequencedRefresh } from '@/stores/sequencedRefresh'

/** Çıkış sırasında Pinia içindeki alan düzeyi kaynak önbelleği atılır; böylece SPA ile yeniden girişte önceki hesabın verileri yeniden kullanılmaz.*/
function clearSessionResourceCaches() {
  useChatResourcesStore().invalidate()
  useEditorResourcesStore().invalidate()
  useOrganizationStore().clearState()
}

// Per-user UI preferences are namespaced by user id in localStorage.
// Reload them whenever the active user changes.
function reloadUserPreferences() {
  // Reset the latch so migration runs once for the new active user.
  resetMigrationLatch()
  reloadFontFromStorage()
  reloadThemeFromStorage()
}

export const useAuthStore = defineStore('auth', () => {
  // Durum
  const user = ref<UserInfo | null>(null)
  const tenant = ref<TenantInfo | null>(null)
  const token = ref<string>('')
  const refreshToken = ref<string>('')
  const knowledgeBases = ref<KnowledgeBaseInfo[]>([])
  const currentKnowledgeBase = ref<KnowledgeBaseInfo | null>(null)
  const selectedTenantId = ref<number | null>(null)
  const selectedTenantName = ref<string | null>(null)
  const allTenants = ref<TenantInfoFromAPI[]>([])
  // memberships lists every tenant the user can authenticate into,
  // along with their role in each. Populated from /auth/login response.
  // v1 deployments will typically have length 1; the field is wired now
  // so PR 3 can render a tenant-switcher UI without a store migration.
  const memberships = ref<Array<{ tenant_id: number; tenant_name?: string; role: string }>>([])
  const isLiteMode = ref(false)
  // pendingInvitationCount is the number of pending tenant invitations
  // addressed to the current user. Renders as a badge next to the
  // avatar; updated by fetchPendingInvitationCount, which runs after
  // login, after tenant switch, and on a polling interval. Polling
  // (vs SSE) is fine — the count is checked rarely and a 1-2 minute
  // staleness window is acceptable for an inbox indicator.
  const pendingInvitationCount = ref<number>(0)
  // Authoritative deployment capability returned by /auth/me. Defaults to
  // false (fail-closed): we hide the "create workspace" affordance until
  // /auth/me confirms the deployment allows it, so an invitation-only
  // deployment never briefly flashes a create action the backend would
  // then reject with 403/2005.
  const canCreateTenant = ref(false)

  const autoAcceptInvitation = ref(false)

  // Hesaplanmış özellik
  const isLoggedIn = computed(() => {
    return !!token.value && !!user.value
  })

  const hasValidTenant = computed(() => {
    return !!tenant.value && !!tenant.value.id
  })

  const currentTenantId = computed(() => {
    return tenant.value?.id || ''
  })

  // currentTenantName resolves the active tenant's display name across the
  // app (sidebar, breadcrumbs, headers). It MUST follow the same active-
  // tenant rule as currentTenantRole / effectiveTenantId: the user's
  // selected override takes precedence over their home tenant. Reading
  // tenant.value.name alone would render the home tenant's name even after
  // the user switched, which is the bug TenantSelector already worked
  // around inline; this computed centralises that fallback ladder.
  const currentTenantName = computed(() => {
    const tid =
      selectedTenantId.value !== null && selectedTenantId.value !== undefined
        ? String(selectedTenantId.value)
        : tenant.value?.id
        ? String(tenant.value.id)
        : ''
    if (!tid) return ''
    if (selectedTenantId.value && selectedTenantName.value) {
      return selectedTenantName.value
    }
    const fromMembership = memberships.value.find((m) => String(m.tenant_id) === tid)
    if (fromMembership?.tenant_name) return fromMembership.tenant_name
    if (tenant.value && String(tenant.value.id) === tid) return tenant.value.name || ''
    return ''
  })

  const currentUserId = computed(() => {
    return user.value?.id || ''
  })

  const canAccessAllTenants = computed(() => {
    return user.value?.can_access_all_tenants || false
  })

  // isSystemAdmin reflects the platform-wide system-administrator flag
  // (User.IsSystemAdmin on the server). It is independent of per-tenant
  // Owner/Admin/Contributor/Viewer roles — a system admin can manage
  // global settings, built-in models, and other system admins regardless
  // of which tenant they're currently scoped into.
  //
  // SECURITY: same caveat as currentTenantRole — this value is hydrated
  // from localStorage on reload and is therefore tamper-prone client-side.
  // Use it ONLY to gate UI visibility (menu entries, route guards). All
  // real authorisation lives in the server-side RequireSystemAdmin
  // middleware (see internal/middleware/rbac.go). A user who flips this
  // bit in DevTools will get a 403 the moment they hit a guarded endpoint.
  const isSystemAdmin = computed(() => {
    return user.value?.is_system_admin === true
  })

  // currentTenantRole returns the user's role in the active tenant
  // (defaulting to '' when memberships have not been loaded). Used by
  // role-aware UI gating; PR 2 wires backend enforcement, PR 3 uses
  // this for menu/button visibility.
  //
  // It MUST read effectiveTenantId, not tenant.value.id. tenant.value is
  // the user's home tenant (set once on login from /auth/me's tenant
  // field); selectedTenantId is the active override applied by the
  // tenant switcher. Reading the wrong field used to leak Owner-level
  // UI to a Viewer who switched into a tenant where they were a Viewer
  // — every gate above (`hasRole(...)`) returned the home-tenant role.
  //
  // SECURITY: This value is read from localStorage (`rethra_memberships`)
  // and therefore MUST be treated as UI-rendering-only — any user can
  // tamper with localStorage and grant themselves "owner" here. All real
  // authorisation decisions live on the server (auth middleware resolves
  // the role from tenant_members on every request). Never branch
  // security-sensitive logic on this value alone.
  const currentTenantRole = computed(() => {
    const tid =
      selectedTenantId.value !== null && selectedTenantId.value !== undefined
        ? String(selectedTenantId.value)
        : tenant.value?.id
        ? String(tenant.value.id)
        : ''
    if (!tid) return ''
    const match = memberships.value.find((m) => String(m.tenant_id) === tid)
    if (match?.role) return match.role
    // Cross-tenant superuser visiting a tenant they're not a member of:
    // backend auth.go resolveTenantRole step2 grants a temporary Admin
    // role without writing tenant_members. Mirror that here so mutation
    // UIs aren't hidden in tenants the superuser switched into. Never
    // surface Owner — the backend caps the temporary grant at Admin too
    // (Owner-only ops like tenant deletion stay reserved for a real
    // Owner inside the target tenant).
    if (canAccessAllTenants.value) return 'admin'
    return ''
  })

  // hasRole answers "is the current tenant role at least <min>?", used by
  // role-aware UI gating across KB / Agent / settings views. The numeric
  // ordering (viewer < contributor < admin < owner) mirrors the server-side
  // matrix in middleware/rbac.go so a v-if here lines up with the 403 the
  // backend would return.
  //
  // SECURITY: shares the same caveat as currentTenantRole — derived from
  // localStorage, MUST be treated as UI-rendering-only. Never rely on
  // hasRole() for security decisions; the server is the source of truth.
  const ROLE_LEVEL: Record<string, number> = {
    viewer: 10,
    contributor: 20,
    admin: 30,
    owner: 40,
  }
  const hasRole = (min: 'viewer' | 'contributor' | 'admin' | 'owner'): boolean => {
    return (ROLE_LEVEL[currentTenantRole.value] ?? 0) >= ROLE_LEVEL[min]
  }

  const effectiveTenantId = computed(() => {
    // Başka bir alan seçildiyse seçilen alanın ID'sini, aksi halde kullanıcının varsayılan alan ID'sini kullan
    return selectedTenantId.value || (tenant.value?.id ? Number(tenant.value.id) : null)
  })

  // İşlem yöntemleri
  const setUser = (userData: UserInfo) => {
    const previousId = user.value?.id
    user.value = userData
    // localStorage'a kaydet
    localStorage.setItem('rethra_user', JSON.stringify(userData))
    if (previousId !== userData.id) {
      reloadUserPreferences()
    }
  }

  const setTenant = (tenantData: TenantInfo | null) => {
    tenant.value = tenantData
    if (tenantData) {
      localStorage.setItem('rethra_tenant', JSON.stringify(tenantData))
    } else {
      localStorage.removeItem('rethra_tenant')
      setSelectedTenant(null, null)
    }
  }

  const setToken = (tokenValue: string) => {
    token.value = tokenValue
    localStorage.setItem('rethra_token', tokenValue)
  }

  const setRefreshToken = (refreshTokenValue: string) => {
    refreshToken.value = refreshTokenValue
    localStorage.setItem('rethra_refresh_token', refreshTokenValue)
  }

  const setKnowledgeBases = (kbList: KnowledgeBaseInfo[]) => {
    // Girdinin bir dizi olduğundan emin ol
    knowledgeBases.value = Array.isArray(kbList) ? kbList : []
    localStorage.setItem('rethra_knowledge_bases', JSON.stringify(knowledgeBases.value))
  }

  const setCurrentKnowledgeBase = (kb: KnowledgeBaseInfo | null) => {
    currentKnowledgeBase.value = kb
    if (kb) {
      localStorage.setItem('rethra_current_kb', JSON.stringify(kb))
    } else {
      localStorage.removeItem('rethra_current_kb')
    }
  }

  // Wipe chat / KB selections that were saved under the previous tenant.
  // These keys are NOT tenant-scoped in storage; after a tenant switch they
  // would otherwise be reloaded verbatim and the chat input would post under
  // the new tenant with an Agent / model id that only existed in the old
  // tenant — backend 403s or "model not found". Called from setSelectedTenant
  // only on an actual tenant change, so logout / init paths are not touched.
  const clearTenantScopedClientState = () => {
    try {
      localStorage.removeItem('rethra_last_chat_model_id')
      localStorage.removeItem('rethra_current_kb')
      const raw = localStorage.getItem('Rethra_settings')
      if (raw) {
        const parsed = JSON.parse(raw)
        if (parsed && typeof parsed === 'object') {
          parsed.selectedAgentId = BUILTIN_QUICK_ANSWER_ID
          parsed.selectedAgentSourceTenantId = null
          parsed.isAgentEnabled = false
          if (parsed.conversationModels && typeof parsed.conversationModels === 'object') {
            parsed.conversationModels.summaryModelId = ''
            parsed.conversationModels.rerankModelId = ''
            parsed.conversationModels.selectedChatModelId = ''
          }
          parsed.selectedKnowledgeBases = []
          parsed.selectedFiles = []
          parsed.selectedFileKbMap = {}
          parsed.knowledgeBaseId = ''
          localStorage.setItem('Rethra_settings', JSON.stringify(parsed))
        }
      }
    } catch (e) {
      // localStorage may be disabled or contain malformed JSON — best effort.
      console.warn('[auth] failed to clear tenant-scoped client state', e)
    }
  }

  const setSelectedTenant = (tenantId: number | null, tenantName: string | null = null) => {
    const previousTenantId = selectedTenantId.value
    const tenantChanged = previousTenantId !== tenantId
    selectedTenantId.value = tenantId
    selectedTenantName.value = tenantName
    if (tenantId !== null) {
      localStorage.setItem('rethra_selected_tenant_id', String(tenantId))
      if (tenantName) {
        localStorage.setItem('rethra_selected_tenant_name', tenantName)
      }
    } else {
      localStorage.removeItem('rethra_selected_tenant_id')
      localStorage.removeItem('rethra_selected_tenant_name')
    }
    if (tenantChanged) {
      clearTenantScopedClientState()
    }
  }

  const setAllTenants = (tenants: TenantInfoFromAPI[]) => {
    allTenants.value = tenants
  }

  const setMemberships = (
    list: Array<{ tenant_id: number; tenant_name?: string; role: string }>
  ) => {
    memberships.value = Array.isArray(list) ? list : []
    localStorage.setItem('rethra_memberships', JSON.stringify(memberships.value))
  }

  // setPendingInvitationCount is the explicit setter used by the
  // polling composable below. Keeping the mutation behind a setter
  // matches the pattern of every other piece of auth state and lets
  // future devtools / strict-mode checks intercept the write.
  const setPendingInvitationCount = (n: number) => {
    pendingInvitationCount.value = Number.isFinite(n) && n >= 0 ? Math.floor(n) : 0
  }

  const setCanCreateTenant = (allowed: boolean) => {
    canCreateTenant.value = allowed
  }

  const setAutoAcceptInvitation = (enabled: boolean) => {
    autoAcceptInvitation.value = enabled
  }

  // fetchPendingInvitationCount hits the dedicated /me/invitations/
  // pending-count endpoint and updates the store. Errors are
  // swallowed — the badge degrades to its last-known value instead
  // of nagging the user with a toast. The caller can choose to fire
  // and forget (login flow) or await (manual refresh button).
  const fetchPendingInvitationCount = async () => {
    // Import lazily to avoid circular module ordering between auth
    // store, axios interceptor, and i18n bootstrapping. The store is
    // imported by very early modules in main.ts; a top-level api
    // import would tighten that graph unnecessarily.
    try {
      const { getMyPendingInvitationCount } = await import('@/api/tenant/invitations')
      const resp = await getMyPendingInvitationCount()
      if (resp.success && resp.data) {
        setPendingInvitationCount(resp.data.pending_count)
      }
    } catch {
      // best-effort; keep last known value
    }
  }

  // Reconcile user / home tenant / memberships with GET /api/v1/auth/me.
  // Login only populated memberships once; SPA navigations skip
  // hydrateSessionFromToken when isLoggedIn is already true — so inviting
  // flows or revokes would leave the sidebar switcher stale until reload.
  //
  // İstek sıralaması: refreshFromAuthMe bir yenilemedir; sonuç, bu çağrıdan sonra gönderilen bir istekten gelmelidir
  // (Davet kabul edildikten / alan oluşturulduktan / alan bilgileri değiştirildikten sonra çağrılır; kenar çubuğu üzerine gelindiğinde daha önce gönderilmiş isteğin yanıtını almamalıdır);
  // ensureAuthMe, 'ilk ekran ortak kullanımı' içindir; devam eden bir istek varsa doğrudan yeniden kullanılır. Çıkış yapmak, devam eden isteği geçersiz kılar;
  // geç gelen yanıt artık oturumu geri yazmaz.
  // Bu oturumun /auth/me ile kalibre edilip edilmediği. Giriş yanıtı / otomatik başlatma yalnızca giriş anındaki anlık görüntüyü yazar;
  // daha sonra platform düzenine ilk girişte yine bir kez kalibrasyon gerekir; sonrasında yazma işlemleri açıkça refresh çağırır.
  let authMeSynced = false

  const authMeRequest = createSequencedRefresh<boolean>(async (isCurrent) => {
    try {
      const { getCurrentUser } = await import('@/api/auth')
      const response = await getCurrentUser()
      if (!isCurrent()) return false
      return applyAuthMeResponse(response)
    } catch {
      return false
    }
  })

  const refreshFromAuthMe = (): Promise<boolean> => authMeRequest.refresh()

  /** Oturum içinde zaten kalibre edildiyse doğrudan döner; aksi halde devam eden /auth/me yeniden kullanılır, yoksa bir kez çekilir.*/
  const ensureAuthMe = (): Promise<boolean> => {
    if (authMeSynced) return Promise.resolve(true)
    return authMeRequest.share()
  }

  type AuthMeResponse = Awaited<ReturnType<typeof import('@/api/auth')['getCurrentUser']>>
  const applyAuthMeResponse = (response: AuthMeResponse): boolean => {
    const u = response.data?.user
    if (!response.success || !u) return false

    setUser(userInfoFromApi(u, response.data?.tenant?.id))

    const tenantSnapshot = response.data?.tenant
    if (tenantSnapshot) {
      setTenant({
        id: String(tenantSnapshot.id) || '',
        name: tenantSnapshot.name || '',
        owner_id: tenantSnapshot.owner_id || u.id || '',
        description: tenantSnapshot.description,
        status: tenantSnapshot.status,
        business: tenantSnapshot.business,
        storage_quota: tenantSnapshot.storage_quota,
        storage_used: tenantSnapshot.storage_used,
        created_at: tenantSnapshot.created_at || new Date().toISOString(),
        updated_at: tenantSnapshot.updated_at || new Date().toISOString(),
      })
    } else {
      setTenant(null)
    }

    const list = response.data?.memberships
    if (Array.isArray(list)) {
      setMemberships(list)
    }

    const createCapability = response.data?.capabilities?.can_create_tenant
    if (typeof createCapability === 'boolean') {
      setCanCreateTenant(createCapability)
    }

    setAutoAcceptInvitation(response.data?.capabilities?.auto_accept_invitation === true)

    authMeSynced = true
    return true
  }

  // token ile alana katılır, üye ilişkilerini yeniler ve yeni alana geçer. token geçersizse ok:false döner (istisna fırlatmaz);
  // bildirim ve yönlendirme çağırana bırakılır (store router'a dokunmaz).
  const acceptInvitationByTokenAndRefresh = async (
    token: string,
  ): Promise<{ ok: boolean; tenantId?: number; tenantName?: string }> => {
    try {
      const { acceptInvitationByToken } = await import('@/api/tenant/invitations')
      const resp = await acceptInvitationByToken(token)
      if (!resp.success || !resp.data?.membership) {
        return { ok: false }
      }
      const tenantId = resp.data.membership.tenant_id
      const tenantName = resp.data.tenant_name
      // Üye ilişkilerini yeniler ve az önce katılınan alana geçer.
      await refreshFromAuthMe()
      setSelectedTenant(tenantId, tenantName ?? null)
      return { ok: true, tenantId, tenantName }
    } catch {
      return { ok: false }
    }
  }

  const getSelectedTenant = () => {
    return selectedTenantId.value
  }

  const setLiteMode = (value: boolean) => {
    isLiteMode.value = value
    if (value) {
      localStorage.setItem('rethra_lite_mode', 'true')
    } else {
      localStorage.removeItem('rethra_lite_mode')
    }
  }

  const logout = () => {
    // Durumu temizle
    user.value = null
    tenant.value = null
    token.value = ''
    refreshToken.value = ''
    knowledgeBases.value = []
    currentKnowledgeBase.value = null
    selectedTenantId.value = null
    selectedTenantName.value = null
    allTenants.value = []
    memberships.value = []
    pendingInvitationCount.value = 0
    canCreateTenant.value = false
    autoAcceptInvitation.value = false
    authMeSynced = false
    // Devam eden /auth/me geçersiz kılınır: çıkıştan sonra dönen yanıt setUser / setTenant çağrılarını yapamaz.
    authMeRequest.invalidate()
    clearSessionResourceCaches()

    // localStorage'ı temizle
    localStorage.removeItem('rethra_user')
    localStorage.removeItem('rethra_tenant')
    localStorage.removeItem('rethra_token')
    localStorage.removeItem('rethra_refresh_token')
    localStorage.removeItem('rethra_knowledge_bases')
    localStorage.removeItem('rethra_current_kb')
    localStorage.removeItem('rethra_selected_tenant_id')
    localStorage.removeItem('rethra_selected_tenant_name')
    localStorage.removeItem('rethra_memberships')
    localStorage.removeItem('rethra_lite_mode')
    isLiteMode.value = false
    try {
      sessionStorage.removeItem('rethra_lite_last_path')
    } catch {
      /* ignore */
    }
    reloadUserPreferences()
  }

  const initFromStorage = () => {
    // Durumu localStorage'dan geri yükle
    const storedUser = localStorage.getItem('rethra_user')
    const storedTenant = localStorage.getItem('rethra_tenant')
    const storedToken = localStorage.getItem('rethra_token')
    const storedRefreshToken = localStorage.getItem('rethra_refresh_token')
    const storedKnowledgeBases = localStorage.getItem('rethra_knowledge_bases')
    const storedCurrentKb = localStorage.getItem('rethra_current_kb')
    const storedSelectedTenantId = localStorage.getItem('rethra_selected_tenant_id')
    const storedSelectedTenantName = localStorage.getItem('rethra_selected_tenant_name')

    if (storedUser) {
      try {
        // Eski localStorage'ı (yeni alanlar eksik olabilir, örneğin
        // is_system_admin) userInfoFromApi üzerinden bir kez normalize eder; böylece 'yeni bir alan ekledim ama eski giriş durumu
        // giriş yanıtı tarafından hiç işlenmediği için alan sonsuza dek undefined kalıyor' açmazından kaçınılır.
        // Bu, '4 yerde kopyalamayı unutma' dışında, özellikle sayfa yenileme akışı için olan 5. gizli giriş noktasıdır.
        user.value = userInfoFromApi(JSON.parse(storedUser))
      } catch (e) {
        console.error(i18n.global.t('authStore.errors.parseUserFailed'), e)
      }
    }

    if (storedTenant) {
      try {
        tenant.value = JSON.parse(storedTenant)
      } catch (e) {
        console.error(i18n.global.t('authStore.errors.parseTenantFailed'), e)
      }
    }

    if (storedToken) {
      token.value = storedToken
    }

    if (storedRefreshToken) {
      refreshToken.value = storedRefreshToken
    }

    if (storedKnowledgeBases) {
      try {
        const parsed = JSON.parse(storedKnowledgeBases)
        knowledgeBases.value = Array.isArray(parsed) ? parsed : []
      } catch (e) {
        console.error(i18n.global.t('authStore.errors.parseKnowledgeBasesFailed'), e)
        knowledgeBases.value = []
      }
    }

    if (storedCurrentKb) {
      try {
        currentKnowledgeBase.value = JSON.parse(storedCurrentKb)
      } catch (e) {
        console.error(i18n.global.t('authStore.errors.parseCurrentKnowledgeBaseFailed'), e)
      }
    }

    if (storedSelectedTenantId) {
      try {
        selectedTenantId.value = Number(storedSelectedTenantId)
        if (storedSelectedTenantName) {
          selectedTenantName.value = storedSelectedTenantName
        }
      } catch (e) {
        console.error('Failed to parse selected tenant ID', e)
        selectedTenantId.value = null
        selectedTenantName.value = null
      }
    }

    const storedMemberships = localStorage.getItem('rethra_memberships')
    if (storedMemberships) {
      try {
        const parsed = JSON.parse(storedMemberships)
        memberships.value = Array.isArray(parsed) ? parsed : []
      } catch (e) {
        console.error('Failed to parse memberships', e)
        memberships.value = []
      }
    }

    isLiteMode.value = localStorage.getItem('rethra_lite_mode') === 'true'
  }

  // Başlatılırken durumu localStorage'dan geri yükle
  initFromStorage()

  return {
    // Durum
    user,
    tenant,
    token,
    refreshToken,
    knowledgeBases,
    currentKnowledgeBase,
    selectedTenantId,
    selectedTenantName,
    allTenants,
    memberships,
    pendingInvitationCount,
    canCreateTenant,
    autoAcceptInvitation,

    // Hesaplanan özellikler
    isLoggedIn,
    hasValidTenant,
    currentTenantId,
    currentTenantName,
    currentUserId,
    canAccessAllTenants,
    isSystemAdmin,
    currentTenantRole,
    hasRole,
    effectiveTenantId,
    isLiteMode,

    // Yöntemler
    setUser,
    setTenant,
    setToken,
    setRefreshToken,
    setKnowledgeBases,
    setCurrentKnowledgeBase,
    setSelectedTenant,
    setAllTenants,
    setMemberships,
    setPendingInvitationCount,
    setCanCreateTenant,
    setAutoAcceptInvitation,
    fetchPendingInvitationCount,
    refreshFromAuthMe,
    ensureAuthMe,
    acceptInvitationByTokenAndRefresh,
    getSelectedTenant,
    setLiteMode,
    logout,
    initFromStorage
  }
})
