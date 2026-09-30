<template>
  <main class="management">
    <header>
      <h1>Yönetim</h1>
      <p>{{ description }}</p>
    </header>
    <nav class="management-tabs" aria-label="Yönetim bölümleri">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        type="button"
        :class="{ active: tab.key === activeTab }"
        :aria-current="tab.key === activeTab ? 'page' : undefined"
        @click="selectTab(tab.key)"
      >
        <t-icon :name="tab.icon" aria-hidden="true" />{{ tab.label }}
      </button>
    </nav>

    <section class="management-panel">
      <FeedbackManagement v-if="activeTab === 'feedback'" />
      <SystemSettings v-else-if="activeTab === 'system-global'" />
      <ModelCatalog v-else-if="activeTab === 'model-catalog'" />
      <RuntimeQueues v-else-if="activeTab === 'runtime-queues'" />
      <PlatformAPIKeys v-else-if="activeTab === 'platform-api-keys'" />
      <SystemAuditLog v-else-if="activeTab === 'system-audit-log'" />
      <AdminAnnouncements v-else-if="activeTab === 'system-announcements'" />
      <SystemUsers v-else-if="activeTab === 'users'" />
      <SystemWorkspaces v-else-if="activeTab === 'workspaces'" />
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { SYSTEM_ADMIN_MANAGEMENT_TABS } from '@/config/settingsAccess'
import FeedbackManagement from './FeedbackManagement.vue'
import SystemSettings from './SystemSettings.vue'
import ModelCatalog from './ModelCatalog.vue'
import RuntimeQueues from './RuntimeQueues.vue'
import PlatformAPIKeys from './PlatformAPIKeys.vue'
import SystemAuditLog from './SystemAuditLog.vue'
import AdminAnnouncements from './AdminAnnouncements.vue'
import SystemUsers from './SystemUsers.vue'
import SystemWorkspaces from './SystemWorkspaces.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const { t } = useI18n()

const tabs = computed(() => [
  ...(auth.isSystemAdmin ? [
    { key: 'users', label: 'Kullanıcılar', icon: 'usergroup' },
    { key: 'workspaces', label: 'Çalışma alanları', icon: 'layers' },
    { key: 'system-global', label: t('settings.system'), icon: 'server' },
    { key: 'model-catalog', label: t('modelCatalog.title'), icon: 'control-platform' },
    { key: 'runtime-queues', label: t('settings.taskQueue'), icon: 'queue' },
    { key: 'platform-api-keys', label: t('platformApiKeys.title'), icon: 'secured' },
    { key: 'system-audit-log', label: t('system.globalSettings.audit.tabLabel'), icon: 'history' },
    { key: 'system-announcements', label: t('announcements.admin.tab'), icon: 'notification' },
  ] : []),
  { key: 'feedback', label: 'Geri Bildirim', icon: 'chat' },
])

// System admins land on Users by default; everyone else only has feedback.
// The backend still enforces RequireSystemAdmin on every system endpoint.
const activeTab = computed(() => {
  const requested = typeof route.query.tab === 'string' ? route.query.tab : ''
  if (requested === 'feedback') return 'feedback'
  if (!auth.isSystemAdmin) return 'feedback'
  return SYSTEM_ADMIN_MANAGEMENT_TABS.has(requested) ? requested : 'users'
})

const description = computed(() => auth.isSystemAdmin
  ? 'Kullanıcıları, çalışma alanlarını, sohbet geri bildirimlerini ve platform genelindeki sistem yönetimi ayarlarını yönetin.'
  : 'Sohbetlerle ilgili değerlendirme, öneri ve şikâyetleri inceleyin.')

function selectTab(key: string) {
  if (key === activeTab.value) return
  void router.replace({ path: '/platform/management', query: { tab: key } })
}
</script>

<style scoped>
.management {
  /* Keep absolutely positioned descendants (sr-only live regions) inside
     this scroll container instead of letting them grow the document. */
  position: relative;
  width: 100%;
  min-height: 100%;
  overflow: auto;
  box-sizing: border-box;
  margin: 0 auto;
  padding: 32px 40px 48px;
  color: var(--td-text-color-primary);
  font-family: var(--app-font-heading);
}
/* rag-platform SuperadminUsers başlığı: text-ui-30 sm:text-ui-34 / 500 / leading 1.04 / -0.028em */
h1 { margin: 0 0 4px; font-size: var(--app-text-page-title); font-weight: 500; letter-spacing: -0.028em; line-height: 1.04; }
header p { max-width: 42rem; margin: 0; color: var(--td-text-color-secondary); font-size: var(--app-text-base); line-height: 1.43; }
.management-tabs {
  display: flex;
  align-items: center;
  gap: 3px;
  width: max-content;
  max-width: 100%;
  overflow-x: auto;
  box-sizing: border-box;
  margin: 28px 0 28px;
  padding: 4px;
  border-radius: 99px;
  background: var(--td-bg-color-container-hover);
  white-space: nowrap;
  scrollbar-width: none;
}
.management-tabs::-webkit-scrollbar { display: none; }
/* rag-platform yönetim sekmeleri (SuperadminAuditLog / EmbedLogs): h2 text-lg, açıklama text-sm muted.
   Ayarlar penceresindeki text-xl sekme başlıkları settings-section.less'te kalır. */
.management-panel :deep(.section-header h2) { font-size: var(--app-text-xl); line-height: 1.5556; letter-spacing: 0; }
.management-panel :deep(.section-header .section-description) { max-width: 65ch; font-size: var(--app-text-base); line-height: 1.43; }
/* rag-platform yönetim sayfası font-heading kapsayıcısında; tablolar Hellix'i miras alır.
   TDesign .t-table { font: var(--td-font-body-medium) } ailesi Inter'e sıfırladığı için geri devral. */
.management-panel :deep(.t-table) { font-family: inherit; }
.management-tabs > button {
  display: inline-flex;
  min-height: 32px;
  flex: none;
  align-items: center;
  gap: 6px;
  box-sizing: border-box;
  padding: 0 10px;
  border: 0;
  border-radius: 99px;
  background: transparent;
  color: var(--td-text-color-secondary);
  font: inherit;
  font-size: var(--app-text-base);
  font-weight: 500;
  cursor: pointer;
  transition: background-color 180ms ease, color 180ms ease;
}
.management-tabs > button:hover,
.management-tabs > .active { color: var(--td-text-color-primary); background: var(--td-bg-color-container); }
.management-tabs .t-icon { flex: none; font-size: 16px; }
.management-tabs > button:focus-visible { outline: 2px solid var(--td-brand-color); outline-offset: 2px; }
@media (max-width: 600px) { .management { padding: 20px 16px; } }
@media (prefers-reduced-motion: reduce) { .management-tabs > button { transition: none; } }
</style>
