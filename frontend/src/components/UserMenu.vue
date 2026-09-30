<template>
  <div class="user-menu" :class="{ 'user-menu--collapsed': uiStore.sidebarCollapsed }" ref="menuRef">
    <!-- Kullanıcı düğmesi -->
    <div class="user-button" data-guide="user-menu" role="button" tabindex="0" :aria-expanded="menuVisible"
      @click="toggleMenu" @keydown.enter.prevent="toggleMenu" @keydown.space.prevent="toggleMenu">
      <div class="user-avatar">
        <img v-if="userAvatar" :src="userAvatar" :alt="$t('common.avatar')" />
        <span v-else class="avatar-placeholder">{{ userInitial }}</span>
      </div>
      <template v-if="!uiStore.sidebarCollapsed">
        <div class="user-info">
          <!-- Çoklu alan / superuser: ilk satır alan adı, ikinci satır username · rol. Tek alan: takma ad + e-posta. -->
          <template v-if="showTenantIdentityLine">
            <div class="user-tenant-name" :title="activeTenantName">{{ activeTenantName }}</div>
            <div class="user-tenant-meta">
              <span v-if="userName && userName !== activeTenantName" class="user-tenant-meta-name">{{ userName }}</span>
              <span v-if="(userName && userName !== activeTenantName) && currentRoleLabel"
                class="user-tenant-meta-sep">·</span>
              <t-icon v-if="currentRoleIcon" :name="currentRoleIcon" size="12px" class="user-tenant-meta-icon" />
              <span v-if="currentRoleLabel" class="user-tenant-meta-role">{{ currentRoleLabel }}</span>
            </div>
          </template>
          <template v-else>
            <div class="user-name">{{ userName }}</div>
            <div class="user-email">{{ userEmail }}</div>
          </template>
        </div>
      </template>
    </div>
    <button v-if="!uiStore.sidebarCollapsed" type="button" class="user-settings-shortcut"
      :aria-label="$t('general.settings')" @click.stop="handleSettings">
      <t-icon name="setting-1" />
    </button>

    <!-- Açılır menü -->
    <Transition name="dropdown">
      <div v-if="menuVisible" class="user-dropdown" role="menu" @click.stop @keydown.esc.prevent="menuVisible = false">
        <!-- Keep platform actions while matching the reference menu's single-column rhythm. -->
        <button type="button" class="menu-item" role="menuitem" aria-keyshortcuts="Meta+Comma Control+Comma" @click="handleSettings">
          <t-icon name="setting-1" class="menu-icon" />
          <span>{{ $t('general.settings') }}</span>
          <span class="menu-shortcut">⌘,</span>
        </button>
        <button type="button" class="menu-item" role="menuitem" @click="handleQuickNav('general')">
          <t-icon name="user" class="menu-icon" />
          <span>{{ $t('general.personalSettings') }}</span>
        </button>
        <button v-if="!authStore.isLiteMode" ref="tenantMenuItemRef" type="button" class="menu-item"
          role="menuitem" @click="handleQuickNav('tenant')"
          @mouseenter="showTenantSwitcher && showTenantSubmenu()"
          @mouseleave="showTenantSwitcher && scheduleHideTenantSubmenu()">
          <t-icon name="user-circle" class="menu-icon" />
          <span>{{ $t('settings.workspaceSettings') }}</span>
        </button>
        <!-- “Yönetim” türündeki hızlı girişler yalnızca gerçekten yazma yetkisi olan kişilere gösterilir. Salt okunur üye listesi ve model listesi
             yine de 「Tüm ayarlar」 üzerinden erişilebilir; böylece viewer yanıltıcı yönetim girişlerini görmez. -->
        <button v-if="canManageMembers" type="button" class="menu-item" role="menuitem" @click="handleQuickNav('members')">
          <t-icon name="usergroup" class="menu-icon" />
          <span>{{ $t('tenantMember.title') }}</span>
        </button>
        <button v-if="canManageModels" type="button" class="menu-item" role="menuitem" @click="handleQuickNav('models')">
          <t-icon name="control-platform" class="menu-icon" />
          <span>{{ $t('settings.modelManagement') }}</span>
        </button>
        <button v-if="authStore.isSystemAdmin || authStore.canAccessAllTenants || authStore.hasRole('admin')" type="button" class="menu-item" role="menuitem" @click="handleManagement">
          <t-icon name="setting-1" class="menu-icon" />
          <span>Yönetim</span>
        </button>
        <template v-if="!authStore.isLiteMode">
          <button type="button" class="menu-item danger" role="menuitem" @click="handleLogout">
            <t-icon name="logout" class="menu-icon" />
            <span>{{ $t('auth.logout') }}</span>
          </button>
        </template>
      </div>
    </Transition>

    <!-- Tenant switcher floating panel — shares the same teleport rationale
         as the IM submenu. Data comes from authStore.memberships, kept fresh via
         GET /auth/me when the submenu opens (throttled) and after invite/create. -->
    <Teleport to="body">
      <div v-if="tenantSubmenuOpen" class="tenant-submenu-floating" :style="tenantSubmenuStyle"
        @mouseenter="showTenantSubmenu" @mouseleave="scheduleHideTenantSubmenu">
        <div class="tenant-submenu-header">
          {{ $t('tenant.switcher.menuLabel') }}
        </div>
        <div class="tenant-submenu-list">
          <div v-for="m in switchableMemberships" :key="m.tenant_id" class="tenant-submenu-item"
            :class="{ 'is-current': isCurrentTenant(m.tenant_id) }" @click="switchToTenant(m)">
            <div class="tenant-submenu-item-avatar" :class="{ 'is-current': isCurrentTenant(m.tenant_id) }">
              {{ tenantInitial(m) }}
              <!-- Home göstergesi: home tenant satırındaki avatarın sağ alt köşesine küçük bir home ekleyin
                   icon. Meta satırında ayrı bir 「Benim」pill oluşturmaya kıyasla bu daha az yer kaplar,
                   ve tüm satırlardaki rozet sütunlarını hizalı tutar. -->
              <span v-if="isHomeTenant(m.tenant_id)" class="tenant-submenu-item-home-dot"
                :title="$t('tenant.switcher.homeTooltip')">
                <t-icon name="home" size="9px" />
              </span>
            </div>
            <!-- İki satırlı düzen: ilk satır tenant adı (kalan genişliğin tamamını alır, rozet tarafından kesilmesini önler
                 — önceden home + mevcut iki rozeti aynı satırdayken, uzun tenant adı doğrudan
                 üç noktayla kısaltılıyordu); ikinci satır role (rol simgesiyle) + “Mevcut” rozeti.
                 home rozeti tenant adı baş harfi avatar'ının köşesine taşındı, artık meta
                 satırında ek yer kaplamıyor; rozet sütun genişliklerinin eşitsiz olmasını önler. -->
            <div class="tenant-submenu-item-info">
              <span class="tenant-submenu-item-name">{{ tenantDisplayName(m) }}</span>
              <div class="tenant-submenu-item-meta">
                <span class="tenant-submenu-item-role">
                  <t-icon v-if="roleIcon(m.role)" :name="roleIcon(m.role)" size="12px"
                    class="tenant-submenu-item-role-icon" />
                  {{ formatRole(m.role) }}
                </span>
                <span v-if="isCurrentTenant(m.tenant_id)" class="tenant-submenu-item-badge">{{
                  $t('tenant.switcher.currentBadge') }}</span>
              </div>
            </div>
          </div>
          <div v-if="switchableMemberships.length === 0" class="tenant-submenu-empty">
            {{ $t('tenant.switcher.empty') }}
          </div>
        </div>
        <!-- Kendi kendine oluşturma girişi, /auth/me tarafından döndürülen arka uç yetenekleriyle tutarlı tutulur. -->
        <div v-if="authStore.canCreateTenant" class="tenant-submenu-create" @click="openCreateTenantDialog">
          <t-icon name="add" class="tenant-submenu-create-icon" />
          <span class="tenant-submenu-create-label">{{ $t('tenant.create.action') }}</span>
        </div>
      </div>
    </Teleport>

    <!-- Çalışma alanı oluşturma penceresi -->
    <CreateTenantDialog v-model:visible="createTenantDialogVisible" @created="onTenantCreated" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useUIStore } from '@/stores/ui'
import { useAuthStore } from '@/stores/auth'
import { MessagePlugin } from 'tdesign-vue-next'
import { logout as logoutApi } from '@/api/auth'
import { useI18n } from 'vue-i18n'
import CreateTenantDialog from '@/components/CreateTenantDialog.vue'
import {
  navigateAfterTenantSwitch,
  persistLastActiveTenantPreference,
  stashTenantSwitchToast,
} from '@/utils/tenantSwitch'
import type { TenantInfo } from '@/api/tenant'
import { useRoleLabel, useHomeTenant } from '@/composables/useRoleLabel'
import { getRootZoom, rectToCssPx, cssViewportSize } from '@/utils/zoom'
import { SETTINGS_MANAGEMENT_SHORTCUT_MIN_ROLE } from '@/config/settingsAccess'
const { t } = useI18n()

const router = useRouter()
const uiStore = useUIStore()
const authStore = useAuthStore()
const { formatRole, roleIcon } = useRoleLabel()
const { homeTenantId, isHomeTenantActive, isHomeTenant } = useHomeTenant()

// Üstteki kullanıcı kartında gösterilen alan adı / mevcut rol: tenant değiştiriciyle gerçek zamanlı değişir.
// activeTenantName öncelikle değiştiricide seçilen adı kullanır (home tenant adına fallback dahil),
// tek alanlı kullanıcılar da kendi home tenant adlarını düzgün biçimde gösterebilir.
const activeTenantName = computed(() => {
  return (
    authStore.selectedTenantName ||
    authStore.tenant?.name ||
    ''
  )
})
const currentRoleLabel = computed(() => formatRole(authStore.currentTenantRole))
const currentRoleIcon = computed(() => roleIcon(authStore.currentTenantRole))

// Tek alanlı kullanıcılar (memberships <= 1 ve superuser değil) = her zaman home + owner, üçüncü
// satır user-email bilgisinin tekrarıdır ve görsel alan kaplamasına gerek yoktur; yalnızca çok alanlı / superuser için
// oluşturulur. Lite modunda RBAC kavramı yoktur, bu nedenle her zaman gizlenir.
const showTenantIdentityLine = computed(() => {
  if (authStore.isLiteMode) return false
  if (authStore.canAccessAllTenants) return true
  return (authStore.memberships ?? []).length > 1
})

// Hızlı girişler, sayfanın en düşük görünür rolü yerine “yönetim yeteneklerini” kullanır: üye listesi ve model listesi
// viewer tarafından görüntülenebilir, ancak avatar menüsündeki “Yönetim” girişi yalnızca gerçekten yönetim işlemleri yapabilen rollere hizmet eder.
const canManageMembers = computed(() =>
  authStore.canAccessAllTenants || authStore.hasRole(SETTINGS_MANAGEMENT_SHORTCUT_MIN_ROLE.members),
)
const canManageModels = computed(() =>
  authStore.canAccessAllTenants ||
  authStore.isSystemAdmin ||
  authStore.hasRole(SETTINGS_MANAGEMENT_SHORTCUT_MIN_ROLE.models),
)
const menuRef = ref<HTMLElement>()
const tenantMenuItemRef = ref<HTMLElement>()
const menuVisible = ref(false)
const tenantSubmenuOpen = ref(false)
const tenantSubmenuStyle = ref<Record<string, string>>({})
let tenantSubmenuHideTimer: ReturnType<typeof setTimeout> | null = null

// Kullanıcı bilgisi doğrudan auth store'dan okunur: başlangıçta main.ts zaten /auth/me ile kalibre edilmiştir,
// burada tekrar ayrı bir istek yapılmaz; ilk ekranda iki kez /auth/me çağrılmasını önler.
const userInfo = computed(() => ({
  username: authStore.user?.username || t('common.defaultUser'),
  email: authStore.user?.email || 'user@example.com',
  avatar: authStore.user?.avatar || '',
}))

const userName = computed(() => userInfo.value.username)
const userEmail = computed(() => userInfo.value.email)
const userAvatar = computed(() => userInfo.value.avatar)

// Kullanıcı adı baş harfi (avatar olmadığında gösterilmek üzere)
const userInitial = computed(() => {
  return userName.value.charAt(0).toUpperCase()
})

// Değiştirme menüsü gösterimi
const toggleMenu = () => {
  menuVisible.value = !menuVisible.value
}

// Ayarlardaki belirli bir bölüme hızlı gezinme
const handleQuickNav = (section: string) => {
  menuVisible.value = false
  uiStore.openSettings()
  router.push({ path: '/platform/settings', query: { section } })
}

// Ayarları aç
const handleSettings = () => {
  menuVisible.value = false
  uiStore.openSettings()
  router.push('/platform/settings')
}

const handleSettingsShortcut = (event: KeyboardEvent) => {
  if ((event.metaKey || event.ctrlKey) && event.key === ',') {
    event.preventDefault()
    handleSettings()
  }
}

const handleManagement = () => {
  menuVisible.value = false
  router.push('/platform/management')
}

// Hover-driven submenu controls. A small hide delay tolerates the pointer
// slipping off briefly onto the gap between menu item and submenu pane.
const closeAll = () => {
  tenantSubmenuOpen.value = false
  menuVisible.value = false
}

// ---------- Create new tenant ----------
// Normal kullanıcı, alan alt menüsünün altında "+ Yeni çalışma alanı oluştur" seçeneğine tıklar → CreateTenantDialog açılır →
// arka uç owner olan bir tenant_members satırı yazar → doğrudan yeni alana geçilir. switchToTenant ile aynı
// setSelectedTenant + navigateAfterTenantSwitch akışı yeniden kullanılır; token'ın
// hâlâ eski alanı göstermesinden kaynaklanan SSE / store tutarsızlığını önler.
const createTenantDialogVisible = ref(false)

const openCreateTenantDialog = () => {
  closeAll()
  if (!authStore.canCreateTenant) {
    MessagePlugin.info(t('tenant.create.disabled'))
    return
  }
  createTenantDialogVisible.value = true
}

const onTenantCreated = async (newTenant: TenantInfo) => {
  await authStore.refreshFromAuthMe()
  authStore.setSelectedTenant(newTenant.id, newTenant.name)
  const persist = persistLastActiveTenantPreference(newTenant.id)
  Promise.race([persist, new Promise((r) => setTimeout(r, 300))])
    .finally(() => navigateAfterTenantSwitch())
}

// ---------- Tenant switcher submenu ----------
//
// Same hover-driven submenu pattern; data comes from
// authStore.memberships (refreshed from /auth/me when the submenu opens and
// after membership-changing actions). PR 4 of #1303 relaxed the X-Tenant-ID
// gate in middleware/auth.go to accept active membership rows, so flipping
// authStore.selectedTenantId here is enough — the next page reload re-issues
// every request with the new header and the server resolves the role server-side.
type Membership = {
  tenant_id: number
  tenant_name?: string
  role: string
}

// switchableMemberships is the curated list shown in the dropdown. We keep
// the active tenant in there (with a "Current" badge) so the user has a
// single place to glance at "where am I right now"; clicking the current
// row is a no-op (handled in switchToTenant).
const switchableMemberships = computed<Membership[]>(() => {
  return authStore.memberships ?? []
})

// Rendered whenever the user has at least one membership — even single-
// tenant users need this submenu to discover the "create new workspace"
// entry at the bottom. Multi-tenant users additionally use it to switch
// between memberships. Cross-tenant superusers keep using the sidebar
// TenantSelector for the "any tenant in the system" case, so we don't
// double-show that here.
const showTenantSwitcher = computed(() => {
  return switchableMemberships.value.length >= 1
})

const isCurrentTenant = (id: number) => {
  const active = authStore.effectiveTenantId
  return active != null && Number(active) === Number(id)
}

const tenantDisplayName = (m: Membership) =>
  m.tenant_name && m.tenant_name.trim() !== '' ? m.tenant_name : `#${m.tenant_id}`

const tenantInitial = (m: Membership) => {
  const name = tenantDisplayName(m).trim()
  return (name.charAt(0) || '?').toUpperCase()
}

const switchToTenant = (m: Membership) => {
  if (isCurrentTenant(m.tenant_id)) {
    closeAll()
    return
  }
  // Etkin alanı her zaman selectedTenantId içine yaz; böylece request.ts her zaman X-Tenant-ID ekler.
  // Eski uygulamada “home'a geri dönünce override'ı temizleme”, isteklerin JWT ile kodlanan alana geri düşmesine neden olur,
  // oysa last_active != home olan oturumlarda JWT tam olarak peer alanıdır (bkz.
  // userService.resolveLoginTenantID); sonuç olarak home'a dönmek yerinde saymak olur.
  // Sunucu tarafındaki kalıcı tercih hâlâ home/peer ayrımına göre tutulur: home durumunda last_active temizlenir,
  // böylece sonraki temiz yeniden girişte doğru şekilde home'a dönülebilir.
  const home = homeTenantId.value
  const switchingToHome = home !== null && home === m.tenant_id
  authStore.setSelectedTenant(m.tenant_id, tenantDisplayName(m))
  closeAll()
  // Toast, reload sonrasında App.vue tarafından gösterilir (burada doğrudan gösterilirse hard reload tarafından yok edilir).
  stashTenantSwitchToast({
    name: tenantDisplayName(m),
    role: formatRole(m.role) || undefined,
    roleEnum: m.role || undefined,
  })
  // Persist "last active tenant" preference (switching to home clears
  // it). Hard reload so every cached store / open SSE stream / in-flight
  // request gets re-keyed under the new tenant; navigateAfterTenantSwitch
  // redirects to the platform home so tenant-scoped resource paths don't
  // white-screen. Race the persist against the existing 400ms grace
  // window so most writes complete before the page tears down.
  const persist = persistLastActiveTenantPreference(switchingToHome ? null : m.tenant_id)
  Promise.race([persist, new Promise((r) => setTimeout(r, 400))])
    .finally(() => navigateAfterTenantSwitch())
}

let lastTenantSubmenuMembershipRefresh = 0
const TENANT_SUBMENU_MEMBERSHIP_REFRESH_MS = 2000

const showTenantSubmenu = () => {
  if (tenantSubmenuHideTimer) {
    clearTimeout(tenantSubmenuHideTimer)
    tenantSubmenuHideTimer = null
  }
  positionTenantSubmenu()
  tenantSubmenuOpen.value = true
  clampFloatingToViewport('.tenant-submenu-floating', tenantSubmenuStyle)
  const now = Date.now()
  if (now - lastTenantSubmenuMembershipRefresh >= TENANT_SUBMENU_MEMBERSHIP_REFRESH_MS) {
    lastTenantSubmenuMembershipRefresh = now
    void authStore.refreshFromAuthMe()
  }
}

const scheduleHideTenantSubmenu = () => {
  if (tenantSubmenuHideTimer) clearTimeout(tenantSubmenuHideTimer)
  tenantSubmenuHideTimer = setTimeout(() => {
    tenantSubmenuOpen.value = false
    tenantSubmenuHideTimer = null
  }, 180)
}

const positionTenantSubmenu = () => {
  const el = tenantMenuItemRef.value
  if (!el) return
  // Submenu is rendered with `position: fixed` under the root zoom — see
  // `.tenant-submenu-floating` styles. Anchor coords come from a visual-pixel
  // rect; normalize to CSS pixels before writing them back to CSS.
  const zoom = getRootZoom()
  const rect = rectToCssPx(el.getBoundingClientRect(), zoom)
  const { width: vw } = cssViewportSize(zoom)
  const PANEL_WIDTH = 264
  const GAP = 8
  const MARGIN = 8

  let left = rect.right + GAP
  if (left + PANEL_WIDTH + MARGIN > vw) {
    left = Math.max(MARGIN, rect.left - PANEL_WIDTH - GAP)
  }

  const top = Math.max(MARGIN, rect.top)

  tenantSubmenuStyle.value = {
    left: `${left}px`,
    top: `${top}px`,
  }
}

// Anchor the floating submenu just to the right of the hovered menu item,
// clamped to the viewport so it stays visible near the screen edge.
const clampFloatingToViewport = (selector: string, target: { value: Record<string, string> }) => {
  requestAnimationFrame(() => {
    const panel = document.querySelector(selector) as HTMLElement | null
    if (!panel) return
    const MARGIN = 8
    // `offsetHeight` and `target.value.top` are CSS pixels; `innerHeight` is
    // visual pixels under root zoom. Normalize the latter to keep the
    // comparison in one coordinate system.
    const { height: vh } = cssViewportSize()
    const h = panel.offsetHeight
    const currentTop = parseFloat(target.value.top || '0') || 0
    const maxTop = vh - h - MARGIN
    if (currentTop > maxTop) {
      target.value = { ...target.value, top: `${Math.max(MARGIN, maxTop)}px` }
    }
  })
}

// Çıkış yap
const handleLogout = async () => {
  menuVisible.value = false

  try {
    // Çıkış yapmak için arka uç API'sini çağır
    await logoutApi()
  } catch (error) {
    // API çağrısı başarısız olsa bile yerel temizlemeye devam et
    console.error('Logout API call failed:', error)
  }

  // Tüm durumu ve yerel depolamayı temizle
  authStore.logout()

  MessagePlugin.success(t('auth.logout'))

  // Giriş sayfasına yönlendir
  router.push('/login')
}

// Kullanıcı bilgilerini eşleştir: bu oturum zaten /auth/me ile eşleştirildi (main.ts başlangıcı, giriş akışı),
// bu nedenle store doğrudan yeniden kullanılır; yalnızca otomatik başlatma gibi /auth/me yolundan geçmeyen yollarda gerçekten istek gönderilir.
// Kalıcı kayıt mantığı (user / tenant / memberships / capabilities) auth store içinde birleştirilmiştir,
// burada artık alan kopyalama işlemi elle yeniden yazılmaz.
const loadUserInfo = async () => {
  try {
    await authStore.ensureAuthMe()
  } catch (error) {
    console.error('Failed to load user info:', error)
  }
}

// Menüyü kapatmak için dışarı tıkla
const handleClickOutside = (e: MouseEvent) => {
  const target = e.target as Node
  if (menuRef.value && menuRef.value.contains(target)) return
  // Tenant submenu is teleported to body, so it's not inside menuRef.
  const tenantFloating = document.querySelector('.tenant-submenu-floating')
  if (tenantFloating && tenantFloating.contains(target)) return
  menuVisible.value = false
  tenantSubmenuOpen.value = false
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
  window.addEventListener('keydown', handleSettingsShortcut)
  loadUserInfo()
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
  window.removeEventListener('keydown', handleSettingsShortcut)
})
</script>

<style lang="less" scoped>
.user-menu {
  position: relative;
  width: 100%;

  &--collapsed {
    .user-button {
      justify-content: center;
      padding: 6px 3px;
      gap: 0;
    }

    .user-dropdown {
      left: calc(100% + 8px);
      bottom: 0;
      right: auto;
      width: 256px;
    }
  }
}

.user-button {
  display: flex;
  align-items: center;
  gap: 9px;
  height: 44px;
  box-sizing: border-box;
  padding: 3px 45px 3px 8px;
  border-radius: var(--app-radius-lg);
  font-family: var(--app-font-heading);
  cursor: pointer;
  transition: all var(--app-motion-base);
  background: transparent;

  &:hover {
    background: #f0f0f0;
  }

  &[aria-expanded='true'] {
    background: #f0f0f0;
  }

  &:focus-visible {
    outline: 2px solid var(--td-brand-color);
    outline-offset: 2px;
  }
}

.user-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  overflow: hidden;
  flex-shrink: 0;
  background: linear-gradient(135deg, var(--td-brand-color) 0%, var(--td-brand-color-active) 100%);
  display: flex;
  align-items: center;
  justify-content: center;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .avatar-placeholder {
    color: var(--td-text-color-anti);
    font-size: var(--app-text-sm);
    font-weight: 600;
    line-height: 1;
  }
}

.user-info {
  flex: 1;
  min-width: 0;
  text-align: left;
  display: flex;
  flex-direction: column;
  gap: 1px;
  line-height: 1.25;
  justify-content: center;

  .user-name {
    font-family: var(--app-font-heading);
    font-size: var(--app-text-caption);
    font-weight: 600;
    letter-spacing: 0;
    color: #383835;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .user-email {
    font-size: var(--app-text-micro);
    color: oklch(0.5486 0 0);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .user-tenant-name {
    font-family: var(--app-font-heading);
    font-size: var(--app-text-caption);
    font-weight: 500;
    letter-spacing: 0.025em;
    color: #383835;
    line-height: 1.25;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .user-tenant-meta {
    display: flex;
    align-items: center;
    gap: 4px;
    margin-top: 0;
    min-width: 0;
    font-size: var(--app-text-micro);
    line-height: 1.25;
    color: oklch(0.5486 0 0);

    .user-tenant-meta-name {
      flex: 0 1 auto;
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .user-tenant-meta-sep {
      flex-shrink: 0;
      color: var(--td-text-color-placeholder);
    }

    .user-tenant-meta-icon {
      flex-shrink: 0;
      color: inherit;
    }

    .user-tenant-meta-role {
      flex-shrink: 0;
    }
  }
}

.user-settings-shortcut {
  position: absolute;
  top: 8px;
  right: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;

  &:hover,
  &:focus-visible {
    background: var(--app-nav-hover);
    color: var(--td-text-color-primary);
  }

  .t-icon {
    font-size: 16px;
  }
}

.user-dropdown {
  position: absolute;
  bottom: 100%;
  left: 5.5px;
  width: 256px;
  max-height: calc(100vh - 24px);
  margin-bottom: 8px;
  padding: 4px;
  box-sizing: border-box;
  overflow-y: auto;
  background: var(--td-bg-color-container);
  border: 0;
  border-radius: var(--app-radius-lg);
  box-shadow: 0 2px 8px -2px rgba(0, 0, 0, 0.16);
  font-family: var(--app-font-heading);
  text-align: left;
  z-index: 1000;
}

.menu-item {
  display: flex;
  align-items: center;
  width: 100%;
  height: 36px;
  gap: 10px;
  padding: 0 12px;
  border: 0;
  border-radius: var(--app-menu-item-radius);
  background: transparent;
  color: var(--td-text-color-primary);
  font-family: var(--app-font-heading);
  font-size: var(--app-text-base);
  font-weight: 400;
  line-height: 20px;
  text-align: left;
  cursor: pointer;

  &:hover,
  &:focus-visible {
    background: var(--app-nav-hover);
    color: var(--td-text-color-primary);
    outline: none;
  }

  &.danger {
    color: var(--td-error-color-6);

    &:hover,
    &:focus-visible {
      background: var(--td-error-color-1);
      color: var(--td-error-color-6);
    }
  }

  .menu-icon {
    width: 16px;
    height: 16px;
    font-size: 16px;
    flex-shrink: 0;
    color: inherit;
  }
}

.menu-shortcut {
  margin-left: auto;
  color: var(--td-text-color-secondary);
  font-family: var(--app-font-family);
  font-size: var(--app-text-xs);
  font-weight: 400;
}

// Açılır liste animasyonu
.dropdown-enter-active,
.dropdown-leave-active {
  transition: opacity 100ms ease-out, transform 100ms ease-out;
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(8px);
}

.dropdown-enter-to,
.dropdown-leave-from {
  opacity: 1;
  transform: translateY(0);
}

</style>

<style lang="less">
// Tenant switcher submenu — teleported to <body>.
// All styling for the panel itself lives here (not in a child component) so
// the markup in UserMenu.vue stays self-contained.
.tenant-submenu-floating {
  position: fixed;
  z-index: 1100;
  width: 264px;
  max-height: 340px;
  display: flex;
  flex-direction: column;
  background: var(--td-bg-color-container);
  border: 0;
  border-radius: var(--app-radius-lg);
  box-shadow: 0 2px 8px -2px rgba(0, 0, 0, 0.16);
  // Pointer bridge so the user can slide off the menu item onto the panel
  // without hitting the gap and triggering mouseleave-hide.
  padding-left: 2px;
  overflow: hidden;

  .tenant-submenu-header {
    padding: 8px 12px 6px;
    font-size: var(--app-text-sm);
    font-weight: 500;
    color: var(--td-text-color-secondary);
    border-bottom: 0.5px solid var(--td-component-stroke);
  }

  .tenant-submenu-list {
    overflow-y: auto;
    padding: 4px;
  }

  .tenant-submenu-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 12px;
    border-radius: var(--app-menu-item-radius);
    cursor: pointer;
    transition: background var(--app-motion-fast);

    &:hover {
      background: var(--td-bg-color-secondarycontainer);
    }

    &.is-current {
      background: var(--td-bg-color-secondarycontainer);
      cursor: default;

      .tenant-submenu-item-name {
        color: var(--td-text-color-primary);
        font-weight: 500;
      }
    }
  }

  .tenant-submenu-item-avatar {
    width: 28px;
    height: 28px;
    border-radius: var(--app-radius-sm);
    background: var(--td-bg-color-secondarycontainer);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: var(--app-text-md);
    font-weight: 600;
    color: var(--td-text-color-secondary);
    flex-shrink: 0;

    &.is-current {
      background: linear-gradient(135deg, var(--td-brand-color) 0%, var(--td-brand-color-active) 100%);
      color: var(--td-text-color-anti);
    }
  }

  .tenant-submenu-item-info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .tenant-submenu-item-name {
    font-size: var(--app-text-md);
    color: var(--td-text-color-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  // İkinci satır: role + rozet, inline biçimde birlikte sıralanır. Rozet ikincil konuma küçültülür,
  // böylece ilk satırdaki tenant adı tam genişliği alır (önceden uzun adlar rozet tarafından sıkıştırılıp üç nokta oluyordu).
  .tenant-submenu-item-meta {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-wrap: wrap;
    min-width: 0;
  }

  .tenant-submenu-item-role {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-size: var(--app-text-xs);
    color: var(--td-text-color-placeholder);

    .tenant-submenu-item-role-icon {
      flex-shrink: 0;
      // Renk role metin rengini devralır; görsel odağı çalmaz
      color: inherit;
    }
  }

  .tenant-submenu-item-badge {
    flex-shrink: 0;
    font-size: var(--app-text-2xs);
    font-weight: 600;
    line-height: 1.2;
    padding: 2px 6px;
    border-radius: var(--app-radius-xs);
    background: var(--td-bg-color-component);
    color: var(--td-text-color-secondary);
  }

  // Home göstergesi avatar'ın sağ altına bindirilmiş küçük bir dot olarak değiştirildi; meta satırında ek yer kaplamaz, böylece
  // her satırdaki rozet sütun genişliği hizalanır; kullanıcı home olmayan bir tenant'a geçtiğinde bu küçük icon yine tek bakışta
  // “ana alanımın hangi satırda olduğunu” gösterir.
  .tenant-submenu-item-avatar {
    position: relative;
  }

  .tenant-submenu-item-home-dot {
    position: absolute;
    right: -3px;
    bottom: -3px;
    width: 14px;
    height: 14px;
    border-radius: 50%;
    background: var(--td-bg-color-container);
    color: var(--td-text-color-secondary);
    border: 1.5px solid var(--td-bg-color-container);
    display: flex;
    align-items: center;
    justify-content: center;
    pointer-events: none;
    box-shadow: 0 0 0 0.5px var(--td-success-color-light);
  }

  .tenant-submenu-empty {
    padding: 12px 10px;
    text-align: center;
    font-size: var(--app-text-sm);
    color: var(--td-text-color-placeholder);
  }

  .tenant-submenu-create {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 10px;
    margin: 3px 4px 5px;
    border-top: .5px solid var(--td-component-stroke);
    border-radius: var(--app-radius-sm);
    cursor: pointer;
    color: var(--td-brand-color);
    font-size: var(--app-text-base);
    font-weight: 500;
    transition: background var(--app-motion-fast);

    &:hover {
      background: color-mix(in srgb, var(--td-brand-color) 8%, transparent);
    }

    .tenant-submenu-create-icon {
      font-size: var(--app-text-xl);
      flex-shrink: 0;
    }

    .tenant-submenu-create-label {
      flex: 1;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
      font-size: var(--app-text-sm);
    }
  }
}
</style>
