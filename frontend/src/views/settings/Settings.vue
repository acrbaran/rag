<template>
  <SettingsModalShell :visible="visible" :title="$t('general.settings')" @close="modalShell.requestClose">
    <template #sidebar-header-extra>
      <div class="settings-search">
        <t-icon name="search" class="settings-search__icon" aria-hidden="true" />
        <input v-model="navQuery" class="settings-search__input" type="text"
          :placeholder="$t('settings.dialogSearchPlaceholder')" :aria-label="$t('settings.dialogSearchPlaceholder')" />
        <button v-if="navQuery" type="button" class="settings-search__clear" :aria-label="$t('settings.dialogSearchClear')"
          @click="navQuery = ''">
          <t-icon name="close" aria-hidden="true" />
        </button>
      </div>
    </template>
    <template #nav>
      <p v-if="navQuery && visibleNavGroups.length === 0" class="settings-search__empty" role="status">
        {{ $t('settings.dialogSearchNoResults') }}
      </p>
      <template v-for="group in visibleNavGroups" :key="group.key">
        <div class="nav-group-title">{{ group.label }}</div>
        <template v-for="item in group.items" :key="item.key">
          <div :class="['nav-item', {
            'active': currentSection === item.key,
            'has-submenu': item.children && item.children.length > 0,
            'expanded': expandedMenus.includes(item.key)
          }]" @click="handleNavClick(item)">
            <!-- Web araması özel SVG simgesi kullanır -->
            <svg v-if="item.key === 'websearch'" width="17" height="17" viewBox="0 0 18 18" fill="none"
              xmlns="http://www.w3.org/2000/svg" class="nav-icon">
              <circle cx="9" cy="9" r="7" stroke="currentColor" stroke-width="1.2" fill="none" />
              <path d="M 9 2 A 3.5 7 0 0 0 9 16" stroke="currentColor" stroke-width="1.2" fill="none" />
              <path d="M 9 2 A 3.5 7 0 0 1 9 16" stroke="currentColor" stroke-width="1.2" fill="none" />
              <line x1="2.94" y1="5.5" x2="15.06" y2="5.5" stroke="currentColor" stroke-width="1.2"
                stroke-linecap="round" />
              <line x1="2.94" y1="12.5" x2="15.06" y2="12.5" stroke="currentColor" stroke-width="1.2"
                stroke-linecap="round" />
            </svg>
            <!-- Sandbox: yalıtılmış çalışma penceresi -->
            <svg v-else-if="item.key === 'sandbox'" width="17" height="17" viewBox="0 0 18 18" fill="none"
              xmlns="http://www.w3.org/2000/svg" class="nav-icon">
              <rect x="2.5" y="3" width="13" height="12" rx="2" stroke="currentColor" stroke-width="1.2"
                fill="none" />
              <path d="M2.5 6.5h13" stroke="currentColor" stroke-width="1.2" />
              <path d="M5.5 10h4M5.5 12.5h2.5" stroke="currentColor" stroke-width="1.2"
                stroke-linecap="round" />
            </svg>
            <t-icon v-else :name="item.icon" class="nav-icon" />
            <span class="nav-label">{{ item.label }}</span>
            <t-icon v-if="item.children && item.children.length > 0"
              :name="expandedMenus.includes(item.key) ? 'chevron-down' : 'chevron-right'"
              class="expand-icon" />
          </div>

          <!-- Alt menü -->
          <Transition name="submenu">
            <div v-if="item.children && expandedMenus.includes(item.key)" class="submenu">
              <div v-for="(child, childIndex) in item.children" :key="childIndex"
                :class="['submenu-item', { 'active': currentSubSection === child.key }]"
                @click.stop="handleSubMenuClick(item.key, child.key)">
                <span class="submenu-label">{{ child.label }}</span>
              </div>
            </div>
          </Transition>
        </template>
      </template>
    </template>
    <div class="content-wrapper" :class="{
      'content-wrapper--wide': currentSection === 'members',
      'content-wrapper--full': isIntegrationSection(currentSection),
    }">
      <p v-if="currentScope" class="settings-scope">
        <span class="settings-scope__name">{{ $t(`settings.dialogScope.${currentScope}`) }}</span>{{ " · " }}{{ $t(`settings.dialogScope.${currentScope}Description`) }}
      </p>
      <!-- Rolün mevcut section'a erişmesine izin verilmez (deep-link ile girildiğinde / alanlar arası geçişten sonra rol düşürüldüğünde) —— belirli section render işleminden önceliklidir.
           Normal gezinme navItems filter üzerinden buraya gelmez, ancak watch(navItems) fallback'i rol düşürüldüğünde
           anında tetiklenir; bu bölüm eski URL'lerle uyumluluk için yedek sağlar. -->
      <div v-if="!canSeeSection(currentSection)" class="section role-denied">
        <div class="role-denied-icon">
          <t-icon name="lock-on" size="48px" />
        </div>
        <div class="role-denied-title">{{ $t('settings.roleDenied.title') }}</div>
        <div class="role-denied-desc">{{ $t('settings.roleDenied.desc') }}</div>
      </div>
      <template v-else>
        <!-- Genel ayarlar -->
        <div v-if="currentSection === 'general'" class="section">
          <GeneralSettings />
        </div>

        <!-- Duyurular (kişisel bildirim geçmişi) -->
        <div v-if="currentSection === 'announcements'" class="section">
          <AnnouncementNotifications in-settings />
        </div>

        <!-- Model yapılandırması -->
        <div v-if="currentSection === 'models'" class="section">
          <ModelSettings />
        </div>

        <!-- Web araması yapılandırması -->
        <div v-if="currentSection === 'websearch'" class="section">
          <WebSearchSettings />
        </div>

        <!-- Mesaj yönetimi -->
        <div v-if="currentSection === 'chathistory'" class="section">
          <ChatHistorySettings />
        </div>

        <!-- Uzun süreli bellek（alan düzeyi anahtar） -->
        <div v-if="currentSection === 'memory'" class="section">
          <MemoryWorkspaceSettings />
        </div>

        <!-- Belleğim（kişisel bellek yönetimi） -->
        <div v-if="currentSection === 'mymemory'" class="section">
          <MemorySettings />
        </div>

        <!-- Sandbox anahtarı (üyenin kendi becerileri / sandbox anahtarı) -->
        <div v-if="currentSection === 'envvars'" class="section">
          <EnvVarSettings />
        </div>

        <!-- Vektör veritabanı motoru -->
        <div v-if="currentSection === 'vectorstore'" class="section">
          <VectorStoreSettings />
        </div>

        <!-- Ayrıştırma motoru -->
        <div v-if="currentSection === 'parser'" class="section">
          <ParserEngineSettings />
        </div>

        <!-- Depolama motoru -->
        <div v-if="currentSection === 'storage'" class="section">
          <StorageBackendSettings />
        </div>

        <!-- Sandbox -->
        <div v-if="currentSection === 'sandbox'" class="section">
          <SandboxSettings />
        </div>

        <!-- Sistem bilgileri -->
        <div v-if="currentSection === 'system'" class="section">
          <SystemInfo />
        </div>

        <!-- Kullanıcı bilgileri (hesap temel bilgileri: ID / kullanıcı adı / e-posta / kayıt zamanı).
           Kullanıcının temel bilgileri owner yetkisine bağlı olmamalıdır. -->
        <div v-if="currentSection === 'userprofile'" class="section">
          <UserProfile />
        </div>

        <!-- Alan bilgileri -->
        <div v-if="currentSection === 'tenant'" class="section">
          <TenantInfo />
        </div>

        <!-- Üye yönetimi (#1303 PR 3) -->
        <div v-if="currentSection === 'members'" class="section">
          <TenantMembers />
        </div>

        <!-- Yayın entegrasyonu -->
        <div v-if="isIntegrationSection(currentSection)" class="section">
          <IntegrationSettingsSection :tab="integrationTabFromSection(currentSection)" />
        </div>
      </template>
    </div>
  </SettingsModalShell>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { LocationQueryRaw } from 'vue-router'
import { useUIStore } from '@/stores/ui'
import { useAuthStore } from '@/stores/auth'
import { useDeploymentCapabilitiesStore } from '@/stores/deploymentCapabilities'
import { useI18n } from 'vue-i18n'
import { MessagePlugin } from 'tdesign-vue-next'
import { useModalShell } from '@/composables/useModalShell'
import SettingsModalShell from '@/components/SettingsModalShell.vue'
import SystemInfo from './SystemInfo.vue'
import TenantInfo from './TenantInfo.vue'
import UserProfile from './UserProfile.vue'
import GeneralSettings from './GeneralSettings.vue'
import AnnouncementNotifications from '@/components/announcements/AnnouncementNotifications.vue'
import ModelSettings from './ModelSettings.vue'
import WebSearchSettings from './WebSearchSettings.vue'
import ChatHistorySettings from './ChatHistorySettings.vue'
import MemorySettings from './MemorySettings.vue'
import EnvVarSettings from './EnvVarSettings.vue'
import MemoryWorkspaceSettings from './MemoryWorkspaceSettings.vue'
import VectorStoreSettings from './VectorStoreSettings.vue'
import ParserEngineSettings from './ParserEngineSettings.vue'
import StorageBackendSettings from './StorageBackendSettings.vue'
import SandboxSettings from './SandboxSettings.vue'
import TenantMembers from './TenantMembers.vue'
import IntegrationSettingsSection from '@/views/integrations/IntegrationSettingsSection.vue'
import {
  INTEGRATION_PREVIEW_ITEMS,
  INTEGRATION_TAB_CAPABILITY,
  INTEGRATION_TAB_MIN_ROLE,
} from '@/config/integrations'
import { SETTINGS_SECTION_MIN_ROLE } from '@/config/settingsAccess'
import { SETTINGS_SECTION_CAPABILITY, skillSettingsSupported } from '@/config/deploymentCapabilities'
import { isToolboxSection, toolboxLocation } from '@/config/toolbox'
import { hostSkillsOnly } from '@/utils/skillTarget'
import {
  buildSettingsRouteQuery,
  integrationSectionKey,
  integrationTabFromSection,
  isIntegrationSection,
  normalizeSettingsSection as normalizeSettingsSectionFromQuery,
  settingsQueryUnchanged,
} from '@/config/settingsRoute'

const route = useRoute()
const router = useRouter()
const uiStore = useUIStore()
const authStore = useAuthStore()
const deploymentCapabilities = useDeploymentCapabilitiesStore()
const { t, te } = useI18n()
const hostSkills = computed(() => hostSkillsOnly(
  deploymentCapabilities.isSupported('settings.sandbox.remote'),
  deploymentCapabilities.isSupported('settings.sandbox.host'),
))
function envNavLabel(): string {
  const hostKey = 'envVarSettings.host.title'
  if (hostSkills.value && te(hostKey)) return t(hostKey)
  return t('envVarSettings.title')
}

const currentSection = ref<string>('general')
const currentSubSection = ref<string>('')
const expandedMenus = ref<string[]>([])

type NavItem = {
  key: string
  icon: string
  label: string
  children?: Array<{ key: string; label: string }>
}

type NavGroup = {
  key: string
  label: string
  items: NavItem[]
}

// Ayarlar ikincil gezinmesindeki en düşük görünür rol settingsAccess.ts'ten gelir ve
// internal/router/router.go içindeki koruma matrisiyle hizalanır.
// “Sayfada en az 1 anlamlı yazma işlemi için gereken en düşük rolü” temel alarak, temel altyapı
// yapılandırmasını (models yazma, websearch yazma, parser/storage/vector/mcp
// CRUD, sandbox bağlantısı, skills kurulumu, chat-history yapılandırması) topluca admin kapsamına alın; salt okunur türleri (general / system info /
// tenant-info / members listesi) viewer'a görünür bırakın; en hassas reset api
// key yalnızca owner'a özeldir. Bu tabloyu değiştirmeden önce router.go içindeki ilgili rota grubunu doğrulayın.
//
// Özel not:
// - chathistory sayfasındaki tek “Mesaj indekslemeyi etkinleştir” anahtarı PUT /tenants/kv/chat-history-config
//   arka uçta g.Admin() kullanır. viewer/contributor'a giriş noktasını gösterip, anahtarı açmalarına izin verip, kaydederken
//   403 almaları çok kötü bir deneyimdir; bu nedenle giriş noktası doğrudan admin kapsamındadır.
// - models listesi viewer tarafından okunabilir, sayfa içindeki “+ Model ekle / Düzenle / Sil” düğmeleri
//   ModelSettings.vue içinde ayrıca hasRole('admin') ile denetlenir; bu nedenle giriş noktasını
//   viewer'a bırakmak mantıklıdır (contributor da model listesini görüntüleyebilir).
const normalizeSettingsSection = (section: string) => {
  return normalizeSettingsSectionFromQuery(section, route.query.tab as string | undefined)
}

const syncSettingsRoute = (sectionKey: string) => {
  if (route.path !== '/platform/settings') return
  const query = buildSettingsRouteQuery(sectionKey, route.query)
  if (settingsQueryUnchanged(route.query, query)) return
  void router.replace({
    path: '/platform/settings',
    query: query as LocationQueryRaw,
  })
}

const isSectionSupported = (key: string): boolean => {
  if (isIntegrationSection(key)) {
    return deploymentCapabilities.isSupported(
      INTEGRATION_TAB_CAPABILITY[integrationTabFromSection(key)],
    )
  }
  if (key === 'skills' || key === 'envvars') {
    return skillSettingsSupported(deploymentCapabilities.capabilities)
  }
  // Duyurular çok kullanıcılı platformun parçasıdır; Lite'ta zil gibi bu bölüm de gizlenir.
  if (key === 'announcements') return !authStore.isLiteMode
  return deploymentCapabilities.isSupported(SETTINGS_SECTION_CAPABILITY[key])
}

const canSeeSection = (key: string): boolean => {
  if (isIntegrationSection(key)) {
    const min = INTEGRATION_TAB_MIN_ROLE[integrationTabFromSection(key)]
    if (!min) return true
    if (authStore.canAccessAllTenants) return true
    return authStore.hasRole(min)
  }
  const min = SETTINGS_SECTION_MIN_ROLE[key] ?? 'viewer'
  // canAccessAllTenants（superuser）, rota katmanında olduğu gibi mutlaka bypass edilmelidir; aksi hâlde cross-tenant olur.
  // Yöneticiler, işlem yapma yetkileri olan girişleri göremiyor (TenantMembers.vue içindeki canManage'e bakın).
  if (authStore.canAccessAllTenants) return true
  return authStore.hasRole(min)
}

const navItems = computed(() => {
  // ad-hoc isAdmin/isOwner kontrollerinin birçok yere dağılmasını önlemek için her zaman SETTINGS_SECTION_MIN_ROLE tablosunu kullanın.
  // Sunucu tarafı her rotada yine g.Viewer/Admin/Owner'ı esas alır; burada yalnızca UI'nin
  // girişi gösterip göstermeyeceği belirlenir; giriş kurallarını değiştirirken settingsAccess.ts ve ilgili arka uç rotalarını da güncelleyin.
  const integrationItems: NavItem[] = INTEGRATION_PREVIEW_ITEMS.map((item) => ({
    key: integrationSectionKey(item.key),
    icon: item.icon.name,
    label: t(`integrations.tabs.${item.key}`),
  }))
  const all: NavItem[] = [
    { key: 'general', icon: 'setting', label: t('general.title') },
    { key: 'announcements', icon: 'notification', label: t('announcements.settingsTitle') },
    { key: 'models', icon: 'control-platform', label: t('settings.modelManagement') },
    { key: 'websearch', icon: 'search', label: t('settings.webSearchConfig') },
    { key: 'chathistory', icon: 'chat', label: t('chatHistorySettings.title') },
    { key: 'memory', icon: 'bulletpoint', label: t('memoryWorkspaceSettings.title') },
    { key: 'vectorstore', icon: 'data-base', label: t('settings.vectorStoreEngine') },
    { key: 'parser', icon: 'file-search', label: t('settings.parserEngine') },
    { key: 'storage', icon: 'cloud', label: t('settings.storageEngine') },
    { key: 'sandbox', icon: 'code', label: t('settings.sandbox.title') },
    { key: 'system', icon: 'info-circle', label: t('settings.versionInfo') },
    { key: 'userprofile', icon: 'user', label: t('userProfile.title') },
    { key: 'mymemory', icon: 'bookmark', label: t('memorySettings.title') },
    { key: 'envvars', icon: 'key', label: envNavLabel() },
    { key: 'tenant', icon: 'user-circle', label: t('settings.tenantInfo') },
    { key: 'members', icon: 'usergroup', label: t('tenantMember.title') },
    ...integrationItems,
  ]
  // currentTenantRole'un boş olması 「membership henüz yüklenmedi」 anlamına gelir — tüm
  // viewer girişlerini render edip rol gelir gelmez tekrar kaybolmalarındansa, önce render etmemek daha güvenlidir; önceki members
  // giriş stratejisiyle tutarlıdır.
  if (!authStore.currentTenantRole && !authStore.canAccessAllTenants) {
    return [] as NavItem[]
  }
  return all.filter((it) => canSeeSection(it.key) && isSectionSupported(it.key))
})

const navGroups = computed<NavGroup[]>(() => {
  const itemMap = new Map(navItems.value.map((item) => [item.key, item]))
  const pickItems = (keys: string[]) => keys.map((key) => itemMap.get(key)).filter(Boolean) as NavItem[]
  // Gruplama: Hesap → Alan → Modeller → Yayınlama Entegrasyonları → Veri ve Uzantılar → Sistem Yönetimi → Platform
  // Önemli düzenleme: kişisel tercihleri(general) ve kullanıcı bilgilerini 「Hesap」 altında toplayın;
  // alan içi özellik anahtarını(chathistory) 「Platform」dan 「Alan」a taşıyın;
  // iki adet 2~3 öğelik dar grubu önlemek için arama motorunu ve harici entegrasyonları 「Veri ve Uzantılar」 altında birleştirin.
  return [
    {
      key: 'account',
      label: t('settings.navGroups.account'),
      items: pickItems(['general', 'announcements', 'userprofile', 'mymemory', 'envvars']),
    },
    {
      key: 'workspace',
      label: t('settings.navGroups.workspace'),
      items: pickItems(['tenant', 'members', 'chathistory', 'memory']),
    },
    {
      key: 'models_runtime',
      label: t('settings.navGroups.modelsRuntime'),
      items: pickItems(['models']),
    },
    {
      key: 'integrations',
      label: t('integrations.title'),
      items: pickItems(INTEGRATION_PREVIEW_ITEMS.map((item) => integrationSectionKey(item.key))),
    },
    {
      key: 'data_extensions',
      label: t('settings.navGroups.dataExtensions'),
      items: pickItems([
        'vectorstore',
        'parser',
        'storage',
        'sandbox',
        'websearch',
      ]),
    },
    {
      key: 'platform',
      label: t('settings.navGroups.platform'),
      items: pickItems(['system']),
    },
  ].filter((group) => group.items.length > 0)
})

const navQuery = ref('')
const GROUP_SCOPE: Record<string, 'personal' | 'workspace' | 'platform'> = {
  account: 'personal',
  platform: 'platform',
}
const currentScope = computed(() => {
  const group = navGroups.value.find((g) => g.items.some((item) => item.key === currentSection.value))
  if (!group) return null
  return GROUP_SCOPE[group.key] || 'workspace'
})
const visibleNavGroups = computed<NavGroup[]>(() => {
  const query = navQuery.value.trim().toLocaleLowerCase()
  if (!query) return navGroups.value
  const matches = (value?: string) => (value || '').toLocaleLowerCase().includes(query)
  return navGroups.value
    .map((group) => {
      if (matches(group.label)) return group
      const items = group.items.filter((item: NavItem) =>
        matches(item.label) || (item.children || []).some((child: { label?: string }) => matches(child.label)))
      return { ...group, items }
    })
    .filter((group) => group.items.length > 0)
})

// Gezinme öğesi tıklama işleme
const handleNavClick = (item: any) => {
  if (item.children && item.children.length > 0) {
    // Alt menü varsa genişletme durumunu değiştir
    const index = expandedMenus.value.indexOf(item.key)
    if (index > -1) {
      expandedMenus.value.splice(index, 1)
    } else {
      expandedMenus.value.push(item.key)
    }
    currentSubSection.value = item.children[0].key
  } else {
    currentSubSection.value = ''
  }

  // İlgili sayfaya geç ve URL'yi ?section=<navKey> olarak eşzamanla.
  // Aksi hâlde başka bir section'dan buraya tıklandığında query değişmez ve rota dinleyicisi içeriği geri çeker.
  currentSection.value = item.key
  syncSettingsRoute(item.key)
}

// Alt menü tıklama işleme
const handleSubMenuClick = (parentKey: string, childKey: string) => {
  currentSection.value = parentKey
  currentSubSection.value = childKey

  // İlgili model türü alanına kaydır
  setTimeout(() => {
    const element = document.querySelector(`[data-model-type="${childKey}"]`)
    if (element) {
      element.scrollIntoView({ behavior: 'smooth', block: 'start' })
    }
  }, 100)
}

// Modal görünümünü kontrol et
const visible = computed(() => {
  return route.path === '/platform/settings' || uiStore.showSettingsModal
})

// Modalı kapat
const handleClose = () => {
  // Blur before unmount so TDesign textarea autosize won't run on a detached node.
  if (document.activeElement instanceof HTMLElement) {
    document.activeElement.blur()
  }
  uiStore.closeSettings()
  // Geçerli rota ayarlar sayfasıysa önceki sayfaya dön
  if (route.path === '/platform/settings') {
    router.back()
  }
}

// Existing in-product shortcuts can still call openSettings; moved tools
// always navigate to the toolbox, with the requested sandbox selection intact.
const redirectToToolbox = (section: string, subSection?: string | null) => {
  if (!isToolboxSection(section)) return false
  uiStore.closeSettings()
  void router.push(toolboxLocation(section, subSection || undefined))
  return true
}

// Başlangıç gezinme ayarını dinle
watch(() => uiStore.settingsInitialSection, (section) => {
  if (section && visible.value) {
    const normalizedSection = normalizeSettingsSection(section)
    if (redirectToToolbox(normalizedSection, uiStore.settingsInitialSubSection)) return
    if (deploymentCapabilities.loaded && !isSectionSupported(normalizedSection)) {
      MessagePlugin.warning(t('settings.capabilityUnavailable'))
      currentSection.value = navItems.value[0]?.key || 'general'
      currentSubSection.value = ''
      return
    }
    currentSection.value = normalizedSection
    syncSettingsRoute(normalizedSection)
    const navItem = (navItems.value as any[]).find((item) => item.key === normalizedSection)
    if (navItem && navItem.children && navItem.children.length > 0) {
      if (!expandedMenus.value.includes(section)) {
        expandedMenus.value.push(section)
      }
      currentSubSection.value = uiStore.settingsInitialSubSection || navItem.children[0].key
      if (uiStore.settingsInitialSubSection) {
        setTimeout(() => {
          const element = document.querySelector(`[data-model-type="${uiStore.settingsInitialSubSection}"]`)
          if (element) {
            element.scrollIntoView({ behavior: 'smooth', block: 'start' })
          }
        }, 300)
      }
    } else {
      currentSubSection.value = ''
    }
  }
}, { immediate: true })

watch(
  () => [visible.value, route.path, route.query.section, deploymentCapabilities.loaded] as const,
  ([isVisible, path, section, capabilitiesLoaded]) => {
    if (!isVisible || path !== '/platform/settings') return
    if (typeof section !== 'string') {
      syncSettingsRoute(currentSection.value || 'general')
      return
    }
    const normalizedSection = normalizeSettingsSectionFromQuery(
      section,
      typeof route.query.tab === 'string' ? route.query.tab : undefined,
    )
    if (capabilitiesLoaded && !isSectionSupported(normalizedSection)) {
      MessagePlugin.warning(t('settings.capabilityUnavailable'))
      const fallback = navItems.value[0]?.key || 'general'
      currentSection.value = fallback
      currentSubSection.value = ''
      syncSettingsRoute(fallback)
      return
    }
    currentSection.value = normalizedSection
    currentSubSection.value = ''
    syncSettingsRoute(normalizedSection)
  },
  { immediate: true },
)

// Alan değiştirildikten sonra roller değişebilir; önceden görünen admin-only panel kaybolabilir.
// currentSection artık gösterilmeyen bir key'e denk gelirse, ilk görünür öğeye geri dön.
watch(navItems, (items) => {
  if (!items.some((item) => item.key === currentSection.value)) {
    const fallback = items[0]?.key || 'general'
    currentSection.value = fallback
    currentSubSection.value = ''
    syncSettingsRoute(fallback)
  }
})

// Esc / maske tıklamasıyla kapat (diğer ayar türü modallarla aynı kabuk katmanı etkileşimini paylaşır)
const modalShell = useModalShell({
  visible: () => visible.value,
  close: handleClose,
})

// Hızlı gezinme olaylarını işle
const handleSettingsNav = (e: CustomEvent) => {
  const { section, subsection } = e.detail
  if (section) {
    const normalizedSection = normalizeSettingsSection(section)
    if (redirectToToolbox(normalizedSection, subsection)) return
    if (deploymentCapabilities.loaded && !isSectionSupported(normalizedSection)) {
      MessagePlugin.warning(t('settings.capabilityUnavailable'))
      currentSection.value = navItems.value[0]?.key || 'general'
      currentSubSection.value = ''
      return
    }
    currentSection.value = normalizedSection
    syncSettingsRoute(normalizedSection)
    // Alt menü varsa otomatik olarak genişlet
    const navItem = (navItems.value as any[]).find((item: any) => item.key === normalizedSection)
    if (navItem && navItem.children && navItem.children.length > 0) {
      if (!expandedMenus.value.includes(section)) {
        expandedMenus.value.push(section)
      }
      // subsection varsa ilgili alt menü öğesini seç
      currentSubSection.value = subsection || navItem.children[0].key
    }
  }
}

onMounted(() => {
  window.addEventListener('settings-nav', handleSettingsNav as EventListener)
})

watch(currentSection, () => {
  if (document.activeElement instanceof HTMLElement) {
    document.activeElement.blur()
  }
})

onUnmounted(() => {
  window.removeEventListener('settings-nav', handleSettingsNav as EventListener)
})
</script>

<style lang="less" scoped>
.settings-search {
  position: relative;
  font-family: var(--app-font-heading);
}

.settings-search__icon {
  position: absolute;
  top: 50%;
  left: 10px;
  width: 16px;
  height: 16px;
  font-size: 16px;
  transform: translateY(-50%);
  color: var(--td-text-color-secondary);
  pointer-events: none;
}

.settings-search__input {
  box-sizing: border-box;
  width: 100%;
  height: 32px;
  padding: 0 32px;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-pill);
  background: var(--td-bg-color-page);
  color: var(--td-text-color-primary);
  font: inherit;
  font-size: var(--app-text-base);
  line-height: 18.75px;
  outline: none;
  transition: border-color var(--app-motion-fast) ease;

  &::placeholder {
    color: var(--td-text-color-secondary);
  }

  &:focus-visible {
    border-color: var(--app-ring-select);
  }
}

.settings-search__clear {
  position: absolute;
  top: 50%;
  right: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  padding: 0;
  border: 0;
  border-radius: var(--app-radius-pill);
  background: transparent;
  color: var(--td-text-color-secondary);
  transform: translateY(-50%);
  cursor: pointer;

  :deep(.t-icon) {
    font-size: 14px;
  }

  &:hover {
    color: var(--td-text-color-primary);
  }
}

.settings-scope {
  max-width: 65ch;
  margin: 0 0 20px;
  padding-right: 24px;
  font-family: var(--app-font-family);
  font-size: var(--app-text-xs);
  line-height: 1.625;
  color: var(--td-text-color-secondary);
}

.settings-scope__name {
  font-weight: 500;
  color: var(--td-text-color-primary);
}

.settings-search__empty {
  margin: 0;
  padding: 8px 12px;
  font-size: var(--app-text-base);
  color: var(--td-text-color-secondary);
}

/* Kaplama katmanı */
/* Açılır pencere kapsayıcısı */
/* Kapat düğmesi */
/* Sol gezinti çubuğu: ilk sürüme göre biraz daha kompakt; yazı boyutu ve boşluklar dengeli */
.expand-icon {
  margin-left: 4px;
  font-size: var(--app-text-base);
  transition: transform var(--app-motion-base) ease;
}

/* Alt menü */
.submenu {
  margin-left: 28px;
  margin-bottom: 3px;
  overflow: hidden;
}

.submenu-item {
  padding: 5px 12px;
  margin-bottom: 2px;
  border-radius: var(--app-radius-xs);
  cursor: pointer;
  color: var(--td-text-color-primary);
  font-size: var(--app-text-md);
  transition: all var(--app-motion-base) ease;
  user-select: none;

  &:hover {
    background-color: var(--td-bg-color-container-hover);
    color: var(--td-text-color-primary);
  }

  &.active {
    background-color: var(--td-bg-color-secondarycontainer);
    color: var(--td-brand-color);
    font-weight: 500;
  }
}

.submenu-label {
  display: block;
}

/* Alt menü animasyonu */
.submenu-enter-active,
.submenu-leave-active {
  transition: all var(--app-motion-base) ease;
}

.submenu-enter-from {
  opacity: 0;
  max-height: 0;
}

.submenu-enter-to {
  opacity: 1;
  max-height: 300px;
}

.submenu-leave-from {
  opacity: 1;
  max-height: 300px;
}

.submenu-leave-to {
  opacity: 0;
  max-height: 0;
}

/* Sağ içerik alanı */
.content-wrapper {
  // Bumped from 600 to 760 when the modal grew from 900→1080 (see
  // .settings-modal). Without this, single-column panes (General,
  // Tenant, API key, …) leave a wide right-hand gutter inside the
  // wider modal. 760 keeps comfortable reading-width on long
  // descriptions without the form fields stretching to the full
  // panel width — which would look stranger than a small gutter.
  width: 100%;
  padding: 24px;
  box-sizing: border-box;

  /* Üye / denetim tablosunda çok sayıda sütun var; 600px işlem sütununu kenara sıkıştırır; sağ içerik sütununu tamamen kullanmak daha güvenlidir. */
  &--wide {
    max-width: none;
    width: 100%;
    padding: 24px;
    box-sizing: border-box;
  }

  &--full {
    max-width: none;
    width: 100%;
    padding: 24px;
    box-sizing: border-box;
  }
}

.section {
  animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* Açılır pencere animasyonu */
/* Kaydırma çubuğu stili */
.role-denied {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 64px 24px;
  gap: 12px;
  min-height: 240px;

  .role-denied-icon {
    color: var(--td-text-color-placeholder);
  }

  .role-denied-title {
    font-size: var(--app-text-xl);
    font-weight: 500;
    color: var(--td-text-color-primary);
  }

  .role-denied-desc {
    font-size: var(--app-text-md);
    color: var(--td-text-color-secondary);
    max-width: 360px;
    line-height: 1.6;
  }
}
</style>
