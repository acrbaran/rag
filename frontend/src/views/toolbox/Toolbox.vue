<template>
  <main class="toolbox-page">
    <header class="toolbox-header">
      <h2>
        <ResourceIcon type="toolbox" :size="24" />
        {{ t('toolbox.title') }}
      </h2>
      <p class="toolbox-subtitle">{{ t('toolbox.description') }}</p>
    </header>

    <template v-if="visibleItems.length">
      <div class="toolbox-toolbar">
        <div class="toolbox-tabs" role="tablist" :aria-label="t('toolbox.title')">
          <button v-for="item in visibleItems" :id="`toolbox-tab-${item.key}`" :key="item.key" type="button"
            role="tab" :aria-selected="selectedItem?.key === item.key" :aria-controls="`toolbox-panel-${item.key}`"
            @click="select(item.key)">
            {{ t(item.title) }}
            <span v-if="item.key === 'browserconnection' && browserStatus" class="toolbox-tab__status"
              :class="`is-${browserStatus}`" role="img" :title="t(`localBrowser.${browserStatus}`)"
              :aria-label="t(`localBrowser.${browserStatus}`)" />
            <span v-else-if="item.key !== 'browserconnection' && counts[item.key] !== undefined"
              class="toolbox-tab__count">{{ counts[item.key] }}</span>
          </button>
        </div>
      </div>

      <section v-if="selectedItem" :id="`toolbox-panel-${selectedItem.key}`" class="toolbox-main" role="tabpanel"
        :aria-labelledby="`toolbox-tab-${selectedItem.key}`">
        <div class="toolbox-section-head">
          <div class="toolbox-section-head__text">
            <h3>
              {{ t(selectedItem.title) }}
              <t-tooltip v-if="'help' in selectedItem" :content="t(selectedItem.help)" placement="bottom"
                overlay-class-name="skill-settings__help-tooltip">
                <t-icon name="help-circle" class="toolbox-help" :aria-label="t(selectedItem.help)" />
              </t-tooltip>
            </h3>
            <p>{{ t(selectedItem.description) }}</p>
          </div>
          <t-button v-if="'action' in selectedItem" theme="primary" class="toolbox-action"
            @click="panel?.openAdd?.()">
            <template #icon><t-icon name="add" size="16px" /></template>
            {{ t(selectedItem.action) }}
          </t-button>
        </div>
        <div class="toolbox-panel">
          <SkillSettings v-if="selectedItem.key === 'skills'" ref="panel" :key="sandboxId"
            :initial-sandbox-id="sandboxId" @count="counts.skills = $event" />
          <McpSettings v-else-if="selectedItem.key === 'mcp'" ref="panel" @count="counts.mcp = $event" />
          <BrowserConnectionSettings v-else-if="selectedItem.key === 'browserconnection'" />
        </div>
      </section>
    </template>

    <EmptyState v-else icon="tools" :title="t('toolbox.unavailable')" />
  </main>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useBrowserConnectionStore } from '@/stores/browserConnection'
import { useDeploymentCapabilitiesStore } from '@/stores/deploymentCapabilities'
import { listSkillCatalog } from '@/api/skill'
import { listMCPServices } from '@/api/mcp-service'
import {
  TOOLBOX_ITEMS,
  canAccessToolboxSection,
  toolboxLocation,
  type ToolboxSection,
} from '@/config/toolbox'
import ResourceIcon from '@/components/icons/ResourceIcon.vue'
import EmptyState from '@/components/EmptyState.vue'
import SkillSettings from '@/views/settings/SkillSettings.vue'
import McpSettings from '@/views/settings/McpSettings.vue'
import BrowserConnectionSettings from '@/views/settings/BrowserConnectionSettings.vue'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const authStore = useAuthStore()
const browserConnection = useBrowserConnectionStore()
const capabilities = useDeploymentCapabilitiesStore()
const panel = ref<{ openAdd?: () => void } | null>(null)
const counts = reactive<Partial<Record<ToolboxSection, number>>>({})
const requestedSection = computed(() => typeof route.params.section === 'string' ? route.params.section : '')
const sandboxId = computed(() => typeof route.query.sandboxId === 'string' ? route.query.sandboxId : '')
const visibleItems = computed(() => TOOLBOX_ITEMS.filter((item) => canAccessToolboxSection(item.key, {
  currentTenantRole: authStore.currentTenantRole,
  canAccessAllTenants: authStore.canAccessAllTenants,
  hasRole: (role) => authStore.hasRole(role),
  isSupported: (capability) => capabilities.isSupported(capability),
})))
const selectedItem = computed(() => visibleItems.value.find((item) => item.key === requestedSection.value))

const browserStatus = computed(() => {
  if (!browserConnection.loaded || !browserConnection.enabled) return ''
  if (browserConnection.connected) return 'connected'
  return browserConnection.device ? 'offline' : 'notPaired'
})

const select = (section: ToolboxSection) => {
  if (section === requestedSection.value) return
  void router.replace(toolboxLocation(section))
}

// The bare /toolbox URL, and tools lost to a role or workspace switch, land on
// the first tool the user can still open.
watch([selectedItem, visibleItems], () => {
  const fallback = visibleItems.value[0]
  if (!selectedItem.value && fallback) void router.replace(toolboxLocation(fallback.key))
}, { immediate: true })

// Tab badges for tools that are not open; the open panel keeps its own badge current.
const summaries: Record<ToolboxSection, () => Promise<void>> = {
  skills: async () => { counts.skills = (await listSkillCatalog())?.data?.length ?? 0 },
  mcp: async () => { counts.mcp = (await listMCPServices()).length },
  browserconnection: async () => { if (!browserConnection.loaded) await browserConnection.refresh() },
}
watch(() => visibleItems.value.map((item) => item.key), (keys, previous = []) => {
  for (const key of keys) {
    if (!previous.includes(key)) summaries[key]().catch(() => {})
  }
}, { immediate: true })
</script>

<style scoped lang="less">
@import (reference) '@/components/css/artifact-filter-tabs.less';

.toolbox-page {
  flex: 1;
  min-width: 0;
  height: 100%;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 48px 40px 0;
  color: var(--td-text-color-primary);
  font-family: var(--app-font-heading);
}

.toolbox-header {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-bottom: 24px;
  flex-shrink: 0;

  h2 {
    display: flex;
    align-items: center;
    gap: 8px;
    margin: 0;
    font-family: var(--app-font-heading);
    font-size: var(--app-text-page-title, var(--app-text-4xl));
    font-weight: 500;
    line-height: 1.04;
    letter-spacing: -0.028em;
  }
}

.toolbox-subtitle {
  margin: 4px 0 0;
  color: var(--td-text-color-secondary);
  font-family: var(--app-font-heading);
  font-size: var(--app-text-base);
  font-weight: 400;
  line-height: 18.75px;
}

// Mirrors ResourceListToolbar: the shared pill track above a hairline divider.
.toolbox-toolbar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  padding-bottom: var(--app-space-3);
  border-bottom: 1px solid var(--td-component-stroke);
}

// No overflow clipping here: it would cut off the selected pill's shadow.
.toolbox-tabs {
  .artifact-filter-tabs();

  button {
    white-space: nowrap;
  }
}

.toolbox-tab__count {
  margin-left: 5px;
  font-size: var(--app-text-xs);
  font-variant-numeric: tabular-nums;
}

.toolbox-tab__status {
  display: inline-block;
  width: 6px;
  height: 6px;
  margin-left: 6px;
  border-radius: 50%;
  background: var(--td-text-color-disabled);
  vertical-align: middle;

  &.is-connected {
    background: var(--td-success-color);
  }

  &.is-offline {
    background: var(--td-warning-color);
  }
}

.toolbox-main {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  padding: var(--app-space-5) 0 var(--app-space-8);
}

.toolbox-section-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--app-space-4);
  margin-bottom: var(--app-space-5);

  h3 {
    display: flex;
    align-items: center;
    gap: 6px;
    margin: 0;
    font-family: var(--app-font-heading);
    font-size: var(--app-text-2xl);
    font-weight: 600;
    line-height: 26px;
  }

  p {
    max-width: 720px;
    margin: 4px 0 0;
    color: var(--td-text-color-secondary);
    font-size: var(--app-text-md);
    line-height: 20px;
  }
}

.toolbox-section-head__text {
  min-width: 0;
}

.toolbox-action.t-button {
  flex-shrink: 0;
  gap: 6px;
}

.toolbox-help {
  flex-shrink: 0;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-xl);
  cursor: help;

  &:hover {
    color: var(--td-text-color-secondary);
  }
}

.toolbox-panel {
  // Panels are shared with the settings dialog; here the section head
  // already carries their title and description.
  > :deep(* > .section-header) {
    display: none;
  }
}

@media (max-width: 720px) {
  .toolbox-page {
    padding: var(--app-space-4) var(--app-space-4) 0;
  }

  .toolbox-section-head {
    flex-direction: column;
  }
}
</style>
