import { createRouter, createWebHistory } from 'vue-router'
import { defineComponent } from 'vue'
import type { RouteLocationNormalized } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useDeploymentCapabilitiesStore } from '@/stores/deploymentCapabilities'
import type { DeploymentCapabilityKey } from '@/config/deploymentCapabilities'
import { MessagePlugin } from 'tdesign-vue-next'
import i18n from '@/i18n'
import { normalizeSettingsSection } from '@/config/settingsRoute'
import { isToolboxSection, toolboxLocation } from '@/config/toolbox'
import { SYSTEM_ADMIN_MANAGEMENT_TABS } from '@/config/settingsAccess'

/** Lite / masaüstü WebView zorla yenilendiğinde yalnızca `/` açılabilir; geri yüklemek için son sayfayı session içinde hatırla */
const LITE_LAST_PATH_KEY = 'rethra_lite_last_path'

// views/platform/index.vue always mounts the settings modal and opens it when
// the path is /platform/settings, so this route only has to own the URL.
// Rendering Settings.vue here would mount a second, independent copy.
const SettingsRouteOutlet = defineComponent({ name: 'SettingsRouteOutlet', render: () => null })

function isLiteEdition(authStore: ReturnType<typeof useAuthStore>) {
  return authStore.isLiteMode || localStorage.getItem('rethra_lite_mode') === 'true'
}

function isLiteSpaDefaultEntry(to: RouteLocationNormalized) {
  return (
    to.path === '/' ||
    to.path === '/platform' ||
    to.path === '/platform/knowledge-bases' ||
    to.name === 'knowledgeBaseList'
  )
}

function isSafeLiteRestoreTarget(path: string) {
  return path.startsWith('/platform/') && !path.startsWith('/platform/organizations')
}

function hasPendingOIDCCallback() {
  if (typeof window === 'undefined') return false
  const hash = window.location.hash || ''
  return hash.includes('oidc_result=') || hash.includes('oidc_error=')
}

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: "/",
      redirect: "/platform/knowledge-bases",
    },
    {
      path: "/login",
      name: "login",
      component: () => import("../views/auth/Login.vue"),
      meta: { requiresAuth: false, requiresInit: false }
    },
    // Embed chat is a separate entry (embed.html + embed-main.ts), not this SPA.
    {
      path: "/register",
      name: "registerByInvite",
      // Share-link landing page reuses the Login form: the same Vue
      // component renders both modes and detects ?token=xxx on mount
      // to switch into invite-register flow. Avoids a parallel page
      // that would duplicate the OIDC / language-switch / styling
      // surface for one extra field.
      component: () => import("../views/auth/Login.vue"),
      meta: { requiresAuth: false, requiresInit: false }
    },
    {
      path: "/onboarding/workspace",
      name: "workspaceOnboarding",
      component: () => import("../views/auth/WorkspaceOnboarding.vue"),
      meta: { requiresAuth: true, requiresInit: false, requiresTenant: false }
    },
    {
      path: "/join",
      name: "joinOrganization",
      // Kuruluş listesi sayfasına yönlendir ve code parametresini invite_code olarak dönüştür
      redirect: (to) => {
        const code = to.query.code as string
        return {
          path: '/platform/organizations',
          query: code ? { invite_code: code } : {}
        }
      },
      meta: { requiresInit: true, requiresAuth: true }
    },
    {
      path: "/knowledgeBase",
      name: "home",
      component: () => import("../views/knowledge/KnowledgeBase.vue"),
      meta: { requiresInit: true, requiresAuth: true }
    },
    {
      path: "/platform",
      name: "Platform",
      redirect: "/platform/knowledge-bases",
      component: () => import("../views/platform/index.vue"),
      meta: { requiresInit: true, requiresAuth: true },
      children: [
        {
          path: "tenant",
          redirect: "/platform/settings"
        },
        {
          path: "settings",
          name: "settings",
          component: SettingsRouteOutlet,
          meta: { requiresInit: true, requiresAuth: true }
        },
        {
          path: "management",
          name: "management",
          component: () => import('../views/system/Management.vue'),
          meta: { requiresInit: true, requiresAuth: true, requiresManagement: true },
        },
        {
          path: "knowledge-bases",
          name: "knowledgeBaseList",
          component: () => import("../views/knowledge/KnowledgeBaseList.vue"),
          meta: { requiresInit: true, requiresAuth: true }
        },
        {
          path: "knowledge-bases/:kbId",
          name: "knowledgeBaseDetail",
          component: () => import("../views/knowledge/KnowledgeBase.vue"),
          meta: { requiresInit: true, requiresAuth: true }
        },
        {
          path: "knowledge-search",
          // Eski yolu yönlendirme olarak koru, isteğe bağlı q parametresiyle genel komut panelini (⌘K) aç
          redirect: (to) => {
            const q = to.query.q
            return {
              path: '/platform/knowledge-bases',
              query: typeof q === 'string' ? { cmdk: q } : { cmdk: '' },
            }
          },
        },
        {
          path: "artifacts",
          name: "artifactLibrary",
          component: () => import("../views/artifacts/ArtifactLibrary.vue"),
          meta: { requiresInit: true, requiresAuth: true, requiredCapability: 'settings.sandbox' }
        },
        {
          path: "toolbox/:section?",
          name: "toolbox",
          component: () => import("../views/toolbox/Toolbox.vue"),
          meta: { requiresInit: true, requiresAuth: true }
        },
        {
          path: "agents",
          name: "agentList",
          component: () => import("../views/agent/AgentList.vue"),
          meta: { requiresInit: true, requiresAuth: true, requiredCapability: 'agents' }
        },
        {
          path: "integrations",
          redirect: (to) => {
            const tab = typeof to.query.tab === 'string' ? to.query.tab : undefined
            const incoming = typeof to.query.section === 'string' ? to.query.section : 'integrations'
            const rest = { ...to.query }
            delete rest.tab
            return {
              path: '/platform/settings',
              query: {
                ...rest,
                section: normalizeSettingsSection(incoming, tab),
              },
            }
          },
          meta: { requiresInit: true, requiresAuth: true }
        },
        {
          path: "creatChat",
          name: "globalCreatChat",
          component: () => import("../views/creatChat/creatChat.vue"),
          meta: { requiresInit: true, requiresAuth: true }
        },
        {
          path: "knowledge-bases/:kbId/creatChat",
          name: "kbCreatChat",
          component: () => import("../views/creatChat/creatChat.vue"),
          meta: { requiresInit: true, requiresAuth: true }
        },
        {
          path: "chat/:chatid",
          name: "chat",
          component: () => import("../views/chat/index.vue"),
          meta: { requiresInit: true, requiresAuth: true }
        },
        {
          path: "organizations",
          name: "organizationList",
          component: () => import("../views/organization/OrganizationList.vue"),
          meta: { requiresInit: true, requiresAuth: true, requiredCapability: 'organizations' }
        },
        // Compatibility redirects for /platform/system/* URLs. System
        // administration surfaces live as tabs on the Management page; keep
        // stable URLs for bookmarks and external links.
        {
          path: "system",
          redirect: { path: "/platform/management", query: { tab: "system-global" } },
          meta: { requiresInit: true, requiresAuth: true, requiresSystemAdmin: true },
        },
        {
          path: "system/settings",
          name: "systemSettings",
          redirect: { path: "/platform/management", query: { tab: "system-global" } },
          meta: { requiresInit: true, requiresAuth: true, requiresSystemAdmin: true },
        },
        {
          path: "system/admins",
          name: "systemAdmins",
          redirect: { path: "/platform/management", query: { tab: "system-global" } },
          meta: { requiresInit: true, requiresAuth: true, requiresSystemAdmin: true },
        },
        {
          path: "system/queues",
          name: "systemQueues",
          redirect: { path: "/platform/management", query: { tab: "runtime-queues" } },
          meta: { requiresInit: true, requiresAuth: true, requiresSystemAdmin: true },
        },
      ],
    },
    // Dev-only markdown rendering test page
    ...(import.meta.env.DEV ? [{
      path: '/platform/dev/markdown',
      name: 'markdownTest',
      component: () => import('../views/dev/MarkdownTestPage.vue'),
      meta: { requiresAuth: false, requiresInit: false }
    }] : []),
  ],
});

async function hydrateSessionFromToken(authStore: ReturnType<typeof useAuthStore>) {
  const token = localStorage.getItem('rethra_token')
  if (!token) return false

  if (!authStore.token) {
    authStore.setToken(token)
  }

  const storedRefreshToken = localStorage.getItem('rethra_refresh_token')
  if (storedRefreshToken && !authStore.refreshToken) {
    authStore.setRefreshToken(storedRefreshToken)
  }

  // /auth/me için veritabanına yazma mantığı yalnızca auth store içinde tek bir yerde tutulur (user / tenant / memberships /
  // capabilities); burada yalnızca önce token store içine yerleştirilir. İsteğin kendisi başlangıç ve kenar çubuğuyla ortak tekilleştirme kullanır.
  return authStore.refreshFromAuthMe()
}

let liteDeepLinkRestoreDone = false

// Rota koruması: kimlik doğrulama durumunu ve sistem başlatma durumunu kontrol et
router.beforeEach(async (to, from, next) => {
  const authStore = useAuthStore()

  // OIDC geri dönüş giriş sonucu, App.vue öğesi bağlandıktan sonra URL hash'ini işlemesine bağlıdır.
  // Burada önce “giriş yapılmadı” diye /login adresine engellenirse, geri dönüş sonucunun kaydedilme fırsatı olmaz.
  if (hasPendingOIDCCallback()) {
    next()
    return
  }

  // Preserve bookmarks for tools that have moved out of Settings.
  if (to.path === '/platform/settings' && isToolboxSection(to.query.section)) {
    next({ ...toolboxLocation(to.query.section,
      typeof to.query.sandboxId === 'string' ? to.query.sandboxId : undefined), replace: true })
    return
  }

  // Old Settings links (?section=system-global etc.) now open the Management page.
  if (to.path === '/platform/settings' && typeof to.query.section === 'string'
    && SYSTEM_ADMIN_MANAGEMENT_TABS.has(to.query.section)) {
    next({ path: '/platform/management', query: { tab: to.query.section }, replace: true })
    return
  }

  // Lite: zorla yenilemeden sonra varsayılan ana sayfaya düşülürse, bu oturumda en son ziyaret edilen /platform alt yolunu geri yükle
  if (!liteDeepLinkRestoreDone) {
    liteDeepLinkRestoreDone = true
    if (isLiteEdition(authStore)) {
      const saved = sessionStorage.getItem(LITE_LAST_PATH_KEY)
      if (saved && isSafeLiteRestoreTarget(saved) && isLiteSpaDefaultEntry(to)) {
        if (saved !== to.fullPath) {
          next(saved)
          return
        }
      }
    }
  }

  // Tenantless onboarding still requires a valid user token even though it
  // deliberately skips the normal tenant/system-initialization gates.
  if (to.path === '/onboarding/workspace') {
    if (!authStore.isLoggedIn) {
      const restored = await hydrateSessionFromToken(authStore)
      if (!restored) {
        next('/login')
        return
      }
    }
    if (authStore.hasValidTenant) {
      next('/platform/knowledge-bases')
    } else {
      next()
    }
    return
  }

  // Giriş sayfasına veya başlatma sayfasına erişiliyorsa doğrudan izin ver
  if (to.meta.requiresAuth === false || to.meta.requiresInit === false) {
    // Giriş yapmış kullanıcı giriş sayfasına erişirse, bilgi tabanı listesi sayfasına yönlendir
    if (to.path === '/login' && authStore.isLoggedIn) {
      next(authStore.hasValidTenant ? '/platform/knowledge-bases' : '/onboarding/workspace')
      return
    }
    next()
    return
  }

  // Kullanıcı kimlik doğrulama durumunu kontrol et
  if (to.meta.requiresAuth !== false) {
    if (!authStore.isLoggedIn) {
      const restored = await hydrateSessionFromToken(authStore)
      if (restored) {
        next(
          !authStore.hasValidTenant && to.meta.requiresTenant !== false
            ? '/onboarding/workspace'
            : to.fullPath,
        )
        return
      }

      next('/login')
      return
    }
  }

  if (to.meta.requiresTenant !== false && !authStore.hasValidTenant) {
    next('/onboarding/workspace')
    return
  }

  // Dağıtım yeteneği yalnızca “arka ucun bu işlevi sağlayıp sağlamadığını” açıklar; hizmet sağlığını veya yapılandırılıp yapılandırılmadığını yansıtmaz.
  // Denetim başarısız olduğunda Store fail-open uygular; gerçek yetki ve kullanılabilirlik yine arka uç arayüzü tarafından doğrulanır.
  const deploymentCapabilities = useDeploymentCapabilitiesStore()
  await deploymentCapabilities.ensureLoaded()
  const requiredCapability = to.meta.requiredCapability as DeploymentCapabilityKey | undefined
  if (requiredCapability && !deploymentCapabilities.isSupported(requiredCapability)) {
    MessagePlugin.warning(i18n.global.t('settings.capabilityUnavailable'))
    next('/platform/knowledge-bases')
    return
  }

  // SystemAdmin gate — checked AFTER auth so a non-admin who's logged
  // out gets redirected to /login first (consistent with how the rest
  // of the auth flow works), and only an authenticated non-admin sees
  // the bounce. This is UI-only; the server enforces the real check.
  if (to.meta.requiresSystemAdmin === true) {
    if (!authStore.isSystemAdmin) {
      next('/platform/knowledge-bases')
      return
    }
  }

  if (to.meta.requiresManagement === true && !authStore.isSystemAdmin && !authStore.canAccessAllTenants && !authStore.hasRole('admin')) {
    next('/platform/knowledge-bases')
    return
  }

  next()
})

router.afterEach((to) => {
  if (!isLiteEdition(useAuthStore())) return
  if (to.path === '/login') return
  if (!to.path.startsWith('/platform')) return
  sessionStorage.setItem(LITE_LAST_PATH_KEY, to.fullPath)
})

export default router
