<template>
  <div class="memory-settings">
    <div class="section-header">
      <div class="section-header-titlewrap">
        <h2>{{ t('memorySettings.title') }}</h2>
        <t-popup
          placement="bottom-start"
          trigger="hover"
          overlay-class-name="memory-usage-popup-overlay"
        >
          <button
            type="button"
            class="usage-trigger-btn"
            :aria-label="t('memorySettings.usage.iconHint')"
            :title="t('memorySettings.usage.iconHint')"
          >
            <t-icon name="info-circle" size="16px" />
          </button>
          <template #content>
            <div class="usage-popup">
              <div class="usage-popup-title">{{ t('memorySettings.usage.title') }}</div>
              <p class="usage-popup-intro">{{ t('memorySettings.usage.intro') }}</p>
              <div class="usage-popup-rows">
                <div v-for="key in usageRowKeys" :key="key" class="usage-popup-row">
                  <span class="usage-popup-label">{{ t(`memorySettings.usage.rows.${key}.label`) }}</span>
                  <span class="usage-popup-text">{{ t(`memorySettings.usage.rows.${key}.text`) }}</span>
                </div>
              </div>
            </div>
          </template>
        </t-popup>
      </div>
      <p class="section-description">{{ t('memorySettings.description') }}</p>
    </div>

    <section class="overview">
      <div class="overview-core">
        <div class="overview-control">
          <span v-if="overviewState" class="status-pill" :class="`is-${overviewState}`">
            <span class="status-dot" />
            {{ t(`memorySettings.overview.${overviewState}`) }}
          </span>
          <div class="overview-switch-row">
            <div class="overview-copy">
              <label class="overview-label">{{ t('memorySettings.enableLabel') }}</label>
              <p class="overview-desc">{{ t('memorySettings.enableDescription') }}</p>
            </div>
            <t-switch
              v-model="userEnabled"
              class="overview-switch"
              :disabled="!settings || !settings.workspace_enabled"
              @change="handleEnabledChange"
            />
          </div>
          <!-- An agent can opt out on its own, so this switch being on is not a
               promise that every conversation uses memory. Say so here rather
               than letting someone conclude the page is broken. -->
          <p v-if="userEnabled && settings?.workspace_enabled" class="overview-hint">
            <t-icon name="info-circle" size="14px" />
            <span>{{ t('memorySettings.agentDisabledHint') }}</span>
          </p>
          <!-- Workspace switch is off: say so plainly instead of showing a personal
               toggle that would appear to work and change nothing. -->
          <div v-if="settings && !settings.workspace_enabled" class="notice">
            <t-icon name="error-circle" size="16px" />
            <span>{{ t('memorySettings.workspaceDisabled') }}</span>
          </div>
        </div>
        <div class="overview-metric">
          <span class="metric-value">{{ counts.active }}</span>
          <span class="metric-label">
            <t-icon name="check-circle" size="14px" />
            {{ t('memorySettings.overview.inUse') }}
          </span>
        </div>
      </div>
      <div class="overview-stats">
        <button
          v-for="stat in statTiles"
          :key="stat.value"
          type="button"
          class="stat-tile"
          :class="{ 'is-current': tab === stat.value, 'is-attention': stat.attention }"
          @click="handleTabChange(stat.value)"
        >
          <span class="stat-icon"><t-icon :name="tabIcon(stat.value)" size="16px" /></span>
          <span class="stat-text">
            <span class="stat-count">{{ stat.count }}</span>
            <span class="stat-label">{{ tabName(stat.value) }}</span>
          </span>
          <t-icon name="chevron-right" class="stat-arrow" size="16px" />
        </button>
      </div>
    </section>

    <div class="list-section">
      <div class="list-toolbar">
        <div class="list-title">
          <h3>{{ t('memorySettings.listTitle') }}</h3>
          <span class="list-count">{{ t('memorySettings.listCount', { count: totalAll }) }}</span>
        </div>
        <div class="list-actions">
          <t-popup
            v-model="addVisible"
            trigger="click"
            placement="bottom-end"
            destroy-on-close
            overlay-class-name="memory-add-popup-overlay"
          >
            <t-button size="small" theme="primary" class="action-btn is-primary" :disabled="!canWrite">
              <template #icon><t-icon name="add" /></template>
              {{ t('memorySettings.add') }}
            </t-button>
            <template #content>
              <div class="add-popup" @click.stop>
                <div class="add-popup-title">{{ t('memorySettings.addTitle') }}</div>
                <label class="add-field">
                  <span class="add-label">{{ t('memorySettings.addKindLabel') }}</span>
                  <t-select
                    v-model="draftKind"
                    size="small"
                    :popup-props="{ overlayClassName: 'memory-add-kind-popup' }"
                  >
                    <t-option v-for="kind in kinds" :key="kind" :value="kind" :label="kindLabel(kind)" />
                  </t-select>
                  <span class="add-kind-hint">{{ kindHint(draftKind) }}</span>
                </label>
                <label class="add-field">
                  <span class="add-label">{{ t('memorySettings.addContentLabel') }}</span>
                  <t-textarea
                    v-model="draftContent"
                    :placeholder="t('memorySettings.addPlaceholder')"
                    :maxlength="300"
                    :autosize="{ minRows: 3, maxRows: 6 }"
                  />
                </label>
                <div class="add-popup-footer">
                  <t-button size="small" variant="outline" @click="addVisible = false">
                    {{ t('common.cancel') }}
                  </t-button>
                  <t-button size="small" theme="primary" :disabled="!draftContent.trim()" @click="handleCreate">
                    {{ t('memorySettings.add') }}
                  </t-button>
                </div>
              </div>
            </template>
          </t-popup>
          <t-button size="small" variant="outline" class="action-btn" @click="handleExport">
            <template #icon><t-icon name="download" /></template>
            {{ t('memorySettings.export') }}
          </t-button>
          <t-popconfirm
            :content="t('memorySettings.consolidateConfirm')"
            :confirm-btn="{ content: t('memorySettings.consolidate') }"
            :cancel-btn="t('common.cancel')"
            placement="bottom"
            @confirm="handleConsolidate"
          >
            <t-button
              size="small"
              variant="outline"
              class="action-btn"
              :loading="consolidating"
              :disabled="!canWrite || totalAll === 0"
            >
              <template #icon><t-icon name="swap" /></template>
              {{ t('memorySettings.consolidate') }}
            </t-button>
          </t-popconfirm>
          <span class="action-divider" aria-hidden="true" />
          <t-popconfirm
            theme="danger"
            :content="t('memorySettings.clearConfirm')"
            :confirm-btn="{ content: t('memorySettings.clear'), theme: 'danger' }"
            :cancel-btn="t('common.cancel')"
            placement="left"
            @confirm="handleClear"
          >
            <t-button
              size="small"
              theme="danger"
              variant="text"
              class="action-btn is-danger"
              :disabled="totalAll === 0 && trackingCount === 0 && documentCount === 0"
            >
              <template #icon><t-icon name="delete" /></template>
              {{ t('memorySettings.clear') }}
            </t-button>
          </t-popconfirm>
        </div>
      </div>

      <div class="segmented" role="tablist">
        <button
          v-for="value in tabs"
          :key="value"
          type="button"
          role="tab"
          class="segment"
          :class="{ 'is-active': tab === value, 'is-attention': value === 'pending' && counts.pending > 0 }"
          :aria-selected="tab === value"
          @click="handleTabChange(value)"
        >
          <t-icon :name="tabIcon(value)" size="14px" />
          <span class="segment-label">{{ tabName(value) }}</span>
          <span class="segment-count">{{ tabCount(value) }}</span>
        </button>
      </div>

      <t-loading :loading="loading" class="list-loading">
        <p v-if="statusHint" class="status-hint">
          <t-icon name="info-circle" size="14px" />
          <span>{{ statusHint }}</span>
        </p>
        <div v-if="listIsEmpty" class="empty">
          <span class="empty-icon"><t-icon :name="tabIcon(tab)" size="22px" /></span>
          <p class="empty-title">{{ emptyTitle }}</p>
          <p class="empty-desc">{{ emptyDescription }}</p>
        </div>
        <ul v-if="isDocuments && documents.length > 0" class="memory-list">
          <li
            v-for="(doc, index) in documents"
            :key="doc.id"
            class="memory-item"
            :style="{ '--i': index }"
          >
            <span class="item-icon"><t-icon name="file" size="16px" /></span>
            <div class="memory-main">
              <p class="memory-content is-title">{{ doc.title || t('memorySettings.untitledDocument') }}</p>
              <div class="memory-meta">
                <span class="meta-chip">{{ t('memorySettings.documentsHits', { hits: doc.hits }) }}</span>
                <span class="meta-text">{{ formatTime(doc.last_used_at) }}</span>
              </div>
            </div>
            <div class="memory-actions">
              <t-button
                size="small"
                theme="primary"
                variant="text"
                :disabled="!doc.knowledge_base_id"
                :title="doc.knowledge_base_id ? t('memorySettings.openDocument') : t('memorySettings.openDocumentUnavailable')"
                @click="handleOpenDocument(doc)"
              >
                <template #icon><t-icon name="jump" /></template>
                {{ t('memorySettings.openDocument') }}
              </t-button>
              <t-popconfirm
                theme="danger"
                :content="t('memorySettings.stopTrackingDocumentConfirm')"
                :confirm-btn="{ content: t('memorySettings.stopTrackingDocument'), theme: 'danger' }"
                :cancel-btn="t('common.cancel')"
                placement="left"
                @confirm="handleStopTrackingDocument(doc)"
              >
                <t-button size="small" theme="default" variant="text">
                  {{ t('memorySettings.stopTrackingDocument') }}
                </t-button>
              </t-popconfirm>
            </div>
          </li>
        </ul>
        <ul v-else-if="isTracking && topics.length > 0" class="memory-list">
          <li
            v-for="(topic, index) in topics"
            :key="topic.id"
            class="memory-item"
            :style="{ '--i': index }"
          >
            <span class="item-icon"><t-icon name="chart-bubble" size="16px" /></span>
            <div class="memory-main">
              <p class="memory-content is-title">{{ topic.topic }}</p>
              <div class="topic-progress" :class="{ 'is-ready': topicProgress(topic) >= 100 }">
                <span class="progress-track">
                  <span class="progress-fill" :style="{ transform: `scaleX(${topicProgress(topic) / 100})` }" />
                </span>
                <span class="progress-text">{{ topicProgressText(topic) }}</span>
              </div>
              <div class="memory-meta">
                <span class="meta-chip">{{ t('memorySettings.kinds.interest') }}</span>
                <span
                  v-if="topic.aliases && topic.aliases.length > 0"
                  class="meta-text memory-topic"
                  :title="topic.aliases.join(', ')"
                >
                  {{ t('memorySettings.trackingAliases', { aliases: topic.aliases.join(', ') }) }}
                </span>
                <span class="meta-text">{{ formatTime(topic.last_seen_at) }}</span>
              </div>
            </div>
            <div class="memory-actions">
              <t-button
                size="small"
                theme="primary"
                variant="text"
                :disabled="!canWrite"
                @click="handlePromoteTopic(topic)"
              >
                <template #icon><t-icon name="star" /></template>
                {{ t('memorySettings.promoteTopic') }}
              </t-button>
              <t-popconfirm
                theme="danger"
                :content="t('memorySettings.dismissTopicConfirm')"
                :confirm-btn="{ content: t('memorySettings.dismissTopic'), theme: 'danger' }"
                :cancel-btn="t('common.cancel')"
                placement="left"
                @confirm="handleDismissTopic(topic)"
              >
                <t-button size="small" theme="default" variant="text">
                  {{ t('memorySettings.dismissTopic') }}
                </t-button>
              </t-popconfirm>
            </div>
          </li>
        </ul>
        <ul v-else-if="isItems && items.length > 0" class="memory-list">
          <li
            v-for="(item, index) in items"
            :key="item.id"
            class="memory-item"
            :class="[`is-${item.status}`, { 'is-editing': editingId === item.id }]"
            :style="{ '--i': index }"
          >
            <span class="item-icon" :title="kindHint(item.kind)">
              <t-icon :name="kindIcon(item.kind)" size="16px" />
            </span>
            <div class="memory-main">
              <div v-if="editingId === item.id" class="memory-edit">
                <t-textarea
                  v-model="editingContent"
                  :autosize="{ minRows: 2, maxRows: 6 }"
                  @keydown.enter.ctrl="handleSaveEdit(item)"
                />
                <div class="memory-edit-actions">
                  <t-button size="small" variant="outline" @click="editingId = ''">
                    {{ t('common.cancel') }}
                  </t-button>
                  <t-button size="small" theme="primary" @click="handleSaveEdit(item)">
                    {{ t('common.save') }}
                  </t-button>
                </div>
              </div>
              <p v-else class="memory-content" :class="{ inactive: isRetired(item) }">
                {{ item.content }}
              </p>
              <div class="memory-meta">
                <span class="meta-chip" :title="kindHint(item.kind)">{{ kindLabel(item.kind) }}</span>
                <span
                  v-if="item.topic && item.topic !== item.content"
                  class="meta-text memory-topic"
                  :title="item.topic"
                >
                  {{ item.topic }}
                </span>
                <span class="meta-text">{{ originLabel(item.origin) }}</span>
                <span class="meta-text">{{ formatTime(item.valid_from) }}</span>
              </div>
            </div>
            <div class="memory-actions" :class="{ 'is-persistent': item.status === 'pending' }">
              <template v-if="item.status === 'pending'">
                <t-button
                  size="small"
                  theme="primary"
                  class="guess-btn"
                  :disabled="!canWrite"
                  @click="handleConfirm(item)"
                >
                  <template #icon><t-icon name="check" /></template>
                  {{ t('memorySettings.confirmGuess') }}
                </t-button>
                <t-button size="small" variant="outline" class="guess-btn" @click="handleReject(item)">
                  <template #icon><t-icon name="close" /></template>
                  {{ t('memorySettings.rejectGuess') }}
                </t-button>
              </template>
              <t-button
                v-if="item.status === 'active'"
                size="small"
                theme="default"
                variant="text"
                shape="square"
                :disabled="!canWrite"
                :title="t('common.edit')"
                @click="startEdit(item)"
              >
                <template #icon><t-icon name="edit" /></template>
              </t-button>
              <!-- Rejecting a guess already drops it and remembers the refusal, so a
                   delete button on a pending row is a weaker duplicate of "No". -->
              <t-popconfirm
                v-if="item.status !== 'pending'"
                theme="danger"
                :content="t('memorySettings.deleteConfirm')"
                :confirm-btn="{ content: t('common.delete'), theme: 'danger' }"
                :cancel-btn="t('common.cancel')"
                placement="left"
                @confirm="handleDelete(item)"
              >
                <t-button
                  size="small"
                  theme="danger"
                  variant="text"
                  shape="square"
                  :title="t('common.delete')"
                >
                  <template #icon><t-icon name="delete" /></template>
                </t-button>
              </t-popconfirm>
            </div>
          </li>
        </ul>
      </t-loading>

      <t-pagination
        v-if="listTotal > pageSize"
        class="memory-pagination"
        :total="listTotal"
        :page-size="pageSize"
        :current="page"
        :show-jumper="false"
        :show-page-size="false"
        @current-change="handlePageChange"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import {
  clearMemoryItems,
  consolidateMemory,
  createMemoryItem,
  confirmMemoryItem,
  deleteMemoryDocument,
  deleteMemoryItem,
  deleteMemoryTopic,
  exportMemoryItems,
  getMemorySettings,
  listMemoryDocuments,
  listMemoryItems,
  listMemoryTopics,
  promoteMemoryTopic,
  rejectMemoryItem,
  updateMemoryEnabled,
  updateMemoryItem,
  type MemoryDoc,
  type MemoryItem,
  type MemoryKind,
  type MemorySettings,
  type MemoryStatus,
  type MemoryTopic,
} from '@/api/memory'

const { t } = useI18n()
const router = useRouter()

type MemoryListTab = MemoryStatus | 'tracking' | 'documents'

const settings = ref<MemorySettings | null>(null)
const userEnabled = ref(false)
const items = ref<MemoryItem[]>([])
const topics = ref<MemoryTopic[]>([])
const documents = ref<MemoryDoc[]>([])
const total = ref(0)
const trackingCount = ref(0)
const documentCount = ref(0)
const tab = ref<MemoryListTab>('active')
const loading = ref(false)
const consolidating = ref(false)
const page = ref(1)
const pageSize = 20

const draftKind = ref<MemoryKind>('fact')
const draftContent = ref('')
const addVisible = ref(false)
const editingId = ref('')
const editingContent = ref('')
const editingImportance = ref(3)

const kinds: MemoryKind[] = ['profile', 'preference', 'fact', 'task', 'interest']
const usageRowKeys = ['alwaysOn', 'situational', 'interest', 'tracking', 'documents', 'pending', 'inactive'] as const

const statuses: MemoryStatus[] = ['active', 'pending', 'superseded', 'archived']
const tabs: MemoryListTab[] = ['active', 'pending', 'tracking', 'documents', 'superseded', 'archived']
const statusLabelKeys: Record<MemoryStatus, string> = {
  active: 'memorySettings.statusActive',
  pending: 'memorySettings.statusPending',
  superseded: 'memorySettings.statusSuperseded',
  archived: 'memorySettings.statusArchived',
}
const counts = ref<Record<MemoryStatus, number>>({
  active: 0,
  pending: 0,
  superseded: 0,
  archived: 0,
})

const isTracking = computed(() => tab.value === 'tracking')
const isDocuments = computed(() => tab.value === 'documents')
const isItems = computed(() => !isTracking.value && !isDocuments.value)
const listTotal = computed(() => {
  if (isTracking.value) return trackingCount.value
  if (isDocuments.value) return documentCount.value
  return total.value
})
const listIsEmpty = computed(() => {
  if (isTracking.value) return topics.value.length === 0
  if (isDocuments.value) return documents.value.length === 0
  return items.value.length === 0
})

const tabName = (value: MemoryListTab) => {
  if (value === 'tracking') return t('memorySettings.statusTracking')
  if (value === 'documents') return t('memorySettings.statusDocuments')
  return t(statusLabelKeys[value])
}

const tabCount = (value: MemoryListTab) => {
  if (value === 'tracking') return trackingCount.value
  if (value === 'documents') return documentCount.value
  return counts.value[value]
}

// The overview tiles are shortcuts to the tabs a person most often needs to act
// on; pending is flagged because nothing there is used until it is answered.
const statTiles = computed(() => [
  { value: 'pending' as MemoryListTab, count: counts.value.pending, attention: counts.value.pending > 0 },
  { value: 'tracking' as MemoryListTab, count: trackingCount.value, attention: false },
  { value: 'documents' as MemoryListTab, count: documentCount.value, attention: false },
])

const overviewState = computed(() => {
  if (!settings.value) return ''
  if (!settings.value.workspace_enabled) return 'workspaceOff'
  return settings.value.effective ? 'on' : 'off'
})

const kindIcons: Record<MemoryKind, string> = {
  profile: 'user',
  preference: 'heart',
  fact: 'lightbulb',
  task: 'task',
  interest: 'star',
}

const kindIcon = (kind: MemoryKind) => kindIcons[kind] || 'bookmark'

const tabIcons: Record<MemoryListTab, string> = {
  active: 'check-circle',
  pending: 'help-circle',
  tracking: 'chart-bubble',
  documents: 'file',
  superseded: 'history',
  archived: 'folder',
}

const tabIcon = (value: MemoryListTab) => tabIcons[value]

const totalAll = computed(() => statuses.reduce((sum, value) => sum + counts.value[value], 0))

const emptyTitle = computed(() => {
  if (tab.value === 'pending') return t('memorySettings.pendingEmptyTitle')
  if (tab.value === 'tracking') return t('memorySettings.trackingEmptyTitle')
  if (tab.value === 'documents') return t('memorySettings.documentsEmptyTitle')
  if (tab.value === 'superseded') return t('memorySettings.supersededEmptyTitle')
  if (tab.value === 'archived') return t('memorySettings.archivedEmptyTitle')
  return t('memorySettings.emptyTitle')
})

const emptyDescription = computed(() => {
  if (tab.value === 'pending') return t('memorySettings.pendingEmptyDescription')
  if (tab.value === 'tracking') return t('memorySettings.trackingEmptyDescription')
  if (tab.value === 'documents') return t('memorySettings.documentsEmptyDescription')
  if (tab.value === 'superseded') return t('memorySettings.supersededEmptyDescription')
  if (tab.value === 'archived') return t('memorySettings.archivedEmptyDescription')
  return t('memorySettings.emptyDescription')
})

const statusHint = computed(() => {
  if (isTracking.value) {
    return topics.value.length === 0 ? '' : t('memorySettings.trackingHint')
  }
  if (isDocuments.value) {
    return documents.value.length === 0 ? '' : t('memorySettings.documentsHint')
  }
  if (items.value.length === 0) return ''
  if (tab.value === 'pending') return t('memorySettings.pendingHint')
  if (tab.value === 'superseded') return t('memorySettings.supersededHint')
  if (tab.value === 'archived') return t('memorySettings.archivedHint')
  return ''
})

// Writing requires both switches; the list itself stays readable either way so
// a user who just turned memory off can still review and delete what is stored.
const canWrite = computed(() => settings.value?.effective === true)

// A pending guess is not in use yet, but it is not retired either: the greyed-out
// strike-through reads as "thrown away" and only fits superseded and archived rows.
const isRetired = (item: MemoryItem) =>
  item.status === 'superseded' || item.status === 'archived'

const kindLabel = (kind: MemoryKind) => t(`memorySettings.kinds.${kind}`)

const kindHint = (kind: MemoryKind) => t(`memorySettings.kindHints.${kind}`)

const originLabel = (origin: MemoryItem['origin']) => t(`memorySettings.origins.${origin}`)

const formatTime = (value: string) => {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const now = new Date()
  const time = date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  if (date.toDateString() === now.toDateString()) {
    return time
  }
  if (date.getFullYear() === now.getFullYear()) {
    return `${date.getMonth() + 1}/${date.getDate()} ${time}`
  }
  return `${date.getFullYear()}/${date.getMonth() + 1}/${date.getDate()}`
}

const topicProgress = (topic: MemoryTopic) => {
  const threshold = Math.max(topic.threshold, 1)
  return Math.min(100, Math.round((topic.hits / threshold) * 100))
}

const topicProgressText = (topic: MemoryTopic) => {
  if (topic.hits >= topic.threshold) {
    return t('memorySettings.trackingReady')
  }
  return t('memorySettings.trackingProgress', { hits: topic.hits, threshold: topic.threshold })
}

const loadSettings = async () => {
  try {
    const response = await getMemorySettings()
    settings.value = response.data
    userEnabled.value = response.data.user_enabled
  } catch (error: any) {
    console.error('Failed to load memory settings:', error)
  }
}

const loadItems = async () => {
  loading.value = true
  try {
    const response = await listMemoryItems({
      status: tab.value as MemoryStatus,
      limit: pageSize,
      offset: (page.value - 1) * pageSize,
    })
    items.value = response.data || []
    total.value = response.total || 0
  } catch (error: any) {
    console.error('Failed to load memories:', error)
    items.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

const loadTopics = async () => {
  loading.value = true
  try {
    const response = await listMemoryTopics({
      limit: pageSize,
      offset: (page.value - 1) * pageSize,
    })
    topics.value = response.data || []
    trackingCount.value = response.total || 0
  } catch (error: any) {
    console.error('Failed to load topics:', error)
    topics.value = []
    trackingCount.value = 0
  } finally {
    loading.value = false
  }
}

const loadDocuments = async () => {
  loading.value = true
  try {
    const response = await listMemoryDocuments({
      limit: pageSize,
      offset: (page.value - 1) * pageSize,
    })
    documents.value = response.data || []
    documentCount.value = response.total || 0
  } catch (error: any) {
    console.error('Failed to load documents:', error)
    documents.value = []
    documentCount.value = 0
  } finally {
    loading.value = false
  }
}

const loadList = async () => {
  if (isTracking.value) {
    await loadTopics()
    return
  }
  if (isDocuments.value) {
    await loadDocuments()
    return
  }
  await loadItems()
}

// Counts are loaded per status so every tab carries its own size, which is how
// a user finds out an inference is waiting for them without switching tabs.
const loadCounts = async () => {
  const [totals, topicResponse, documentResponse] = await Promise.all([
    Promise.all(
      statuses.map(async (value) => {
        try {
          const response = await listMemoryItems({ status: value, limit: 1 })
          return response.total || 0
        } catch (error: any) {
          return 0
        }
      }),
    ),
    listMemoryTopics({ limit: 1 }).catch(() => ({ total: 0 })),
    listMemoryDocuments({ limit: 1 }).catch(() => ({ total: 0 })),
  ])
  statuses.forEach((value, index) => {
    counts.value[value] = totals[index]
  })
  trackingCount.value = topicResponse.total || 0
  documentCount.value = documentResponse.total || 0
}

const reload = async () => {
  page.value = 1
  await Promise.all([loadList(), loadCounts()])
}

const handlePromoteTopic = async (topic: MemoryTopic) => {
  try {
    await promoteMemoryTopic(topic.id)
    MessagePlugin.success(t('memorySettings.promoteSuccess'))
    tab.value = 'active'
    await reload()
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('memorySettings.promoteFailed'))
  }
}

const handleDismissTopic = async (topic: MemoryTopic) => {
  try {
    await deleteMemoryTopic(topic.id)
    MessagePlugin.success(t('memorySettings.dismissSuccess'))
    await reload()
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('memorySettings.dismissFailed'))
  }
}

const handleOpenDocument = (doc: MemoryDoc) => {
  if (!doc.knowledge_base_id) {
    MessagePlugin.warning(t('memorySettings.openDocumentUnavailable'))
    return
  }
  router.push({
    name: 'knowledgeBaseDetail',
    params: { kbId: doc.knowledge_base_id },
    query: { knowledge_id: doc.knowledge_id },
  })
}

const handleStopTrackingDocument = async (doc: MemoryDoc) => {
  try {
    await deleteMemoryDocument(doc.id)
    MessagePlugin.success(t('memorySettings.stopTrackingDocumentSuccess'))
    await reload()
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('memorySettings.stopTrackingDocumentFailed'))
  }
}

const handleConfirm = async (item: MemoryItem) => {
  try {
    await confirmMemoryItem(item.id)
    MessagePlugin.success(t('memorySettings.confirmSuccess'))
    await reload()
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('memorySettings.confirmFailed'))
  }
}

const handleReject = async (item: MemoryItem) => {
  try {
    await rejectMemoryItem(item.id)
    MessagePlugin.success(t('memorySettings.rejectSuccess'))
    await reload()
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('memorySettings.rejectFailed'))
  }
}

const handleTabChange = async (value: string | number) => {
  tab.value = value as MemoryListTab
  await reload()
}

const handlePageChange = async (current: number) => {
  page.value = current
  await loadList()
}

const handleEnabledChange = async (value: boolean) => {
  try {
    const response = await updateMemoryEnabled(value)
    settings.value = response.data
    userEnabled.value = response.data.user_enabled
    MessagePlugin.success(
      value ? t('memorySettings.toasts.enabled') : t('memorySettings.toasts.disabled'),
    )
  } catch (error: any) {
    userEnabled.value = !value
    MessagePlugin.error(t('memorySettings.toasts.saveFailed', { message: error?.message || '' }))
  }
}

watch(addVisible, (visible) => {
  if (visible) draftContent.value = ''
})

const handleCreate = async () => {
  const content = draftContent.value.trim()
  if (!content) return
  try {
    await createMemoryItem({ kind: draftKind.value, content })
    draftContent.value = ''
    addVisible.value = false
    tab.value = 'active'
    await reload()
    await loadSettings()
    MessagePlugin.success(t('memorySettings.toasts.added'))
  } catch (error: any) {
    MessagePlugin.error(t('memorySettings.toasts.saveFailed', { message: error?.message || '' }))
  }
}

const startEdit = (item: MemoryItem) => {
  editingId.value = item.id
  editingContent.value = item.content
  editingImportance.value = item.importance
}

const handleSaveEdit = async (item: MemoryItem) => {
  const content = editingContent.value.trim()
  if (!content) return
  try {
    await updateMemoryItem(item.id, { content, importance: editingImportance.value })
    editingId.value = ''
    await Promise.all([loadItems(), loadCounts()])
    MessagePlugin.success(t('memorySettings.toasts.updated'))
  } catch (error: any) {
    MessagePlugin.error(t('memorySettings.toasts.saveFailed', { message: error?.message || '' }))
  }
}

const handleDelete = async (item: MemoryItem) => {
  try {
    await deleteMemoryItem(item.id)
    await Promise.all([loadItems(), loadCounts()])
    await loadSettings()
    MessagePlugin.success(t('memorySettings.toasts.deleted'))
  } catch (error: any) {
    MessagePlugin.error(t('memorySettings.toasts.saveFailed', { message: error?.message || '' }))
  }
}

const handleClear = async () => {
  try {
    const response = await clearMemoryItems()
    await reload()
    await loadSettings()
    MessagePlugin.success(t('memorySettings.toasts.cleared', { count: response.removed || 0 }))
  } catch (error: any) {
    MessagePlugin.error(t('memorySettings.toasts.saveFailed', { message: error?.message || '' }))
  }
}

// A review that merges nothing is the normal outcome, so saying only "nothing
// to do" leaves the person unable to tell a tidy store from a broken button.
const consolidateSkipMessage: Record<string, string> = {
  too_few_items: 'memorySettings.consolidateTooFewItems',
  no_candidates: 'memorySettings.consolidateNoCandidates',
  model_declined: 'memorySettings.consolidateModelDeclined',
  too_soon: 'memorySettings.consolidateTooSoon',
}

const handleConsolidate = async () => {
  if (consolidating.value) return
  consolidating.value = true
  try {
    const response = await consolidateMemory()
    const result = response.data
    if (result?.merged || result?.demoted || result?.expired) {
      MessagePlugin.success(
        t('memorySettings.consolidateSuccess', {
          merged: result.merged || 0,
          demoted: result.demoted || 0,
          expired: result.expired || 0,
        }),
      )
    } else if (result?.skipped === 'model_unavailable') {
      // Nothing merged because nothing could be judged, which is a problem to
      // fix rather than a tidy store.
      MessagePlugin.warning(t('memorySettings.consolidateModelUnavailable'))
    } else {
      MessagePlugin.info(t(consolidateSkipMessage[result?.skipped ?? ''] ?? 'memorySettings.consolidateNothing'))
    }
    await reload()
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('memorySettings.consolidateFailed'))
  } finally {
    consolidating.value = false
  }
}

const handleExport = async () => {
  try {
    const response = await exportMemoryItems()
    const blob = new Blob([JSON.stringify(response.data || [], null, 2)], {
      type: 'application/json',
    })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = 'rethra-memories.json'
    link.click()
    URL.revokeObjectURL(url)
  } catch (error: any) {
    MessagePlugin.error(t('memorySettings.toasts.saveFailed', { message: error?.message || '' }))
  }
}

onMounted(async () => {
  await loadSettings()
  await reload()
})
</script>

<style lang="less" scoped>
@import (reference) '@/components/css/settings-section.less';

@ease: cubic-bezier(0.32, 0.72, 0, 1);

.memory-settings {
  width: 100%;
  container-type: inline-size;
  --mem-shell: var(--app-surface-muted);
  --mem-card: var(--td-bg-color-container);
  --mem-ring: color-mix(in srgb, var(--td-component-stroke) 70%, transparent);
  --mem-inner-highlight: inset 0 1px 0 rgba(255, 255, 255, 0.9);
  --mem-lift: 0 1px 2px rgba(0, 0, 0, 0.04), 0 8px 24px -12px rgba(0, 0, 0, 0.12);
}

:root[theme-mode='dark'] .memory-settings {
  --mem-shell: #1a1a1a;
  --mem-card: #262626;
  --mem-ring: rgba(255, 255, 255, 0.07);
  --mem-inner-highlight: inset 0 1px 0 rgba(255, 255, 255, 0.05);
  --mem-lift: 0 1px 2px rgba(0, 0, 0, 0.3), 0 10px 28px -12px rgba(0, 0, 0, 0.6);
}

.section-header {
  .settings-section-header();
}

.section-header-titlewrap {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.usage-trigger-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 24px;
  height: 24px;
  margin: 0;
  padding: 0;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: pointer;
  line-height: 0;
  transition: background-color var(--app-motion-base) @ease, color var(--app-motion-base) @ease;

  :deep(.t-icon) {
    display: block;
  }

  &:hover {
    background-color: var(--td-bg-color-secondarycontainer);
    color: var(--td-brand-color);
  }

  &:focus-visible {
    outline: 2px solid var(--td-brand-color-focus);
    outline-offset: 1px;
  }
}

/* Overview: an outer tray holding a raised inner card, so the switch and the
   headline number read as one object with the shortcuts tucked beneath. */
.overview {
  padding: 6px;
  border-radius: var(--app-radius-2xl);
  background: var(--mem-shell);
  box-shadow: 0 0 0 1px var(--mem-ring);
  animation: mem-rise 520ms @ease both;
}

.overview-core {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 20px;
  padding: 20px 22px;
  border-radius: calc(var(--app-radius-2xl) - 6px);
  background: var(--mem-card);
  box-shadow: 0 0 0 1px var(--mem-ring), var(--mem-inner-highlight), 0 1px 2px rgba(0, 0, 0, 0.03);
}

.overview-control {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
}

.status-pill {
  display: inline-flex;
  align-items: center;
  align-self: flex-start;
  gap: 6px;
  padding: 3px 10px 3px 8px;
  border-radius: 999px;
  font-size: var(--app-text-xs);
  font-weight: 500;
  letter-spacing: 0.02em;
  text-transform: uppercase;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);

  .status-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: currentColor;
  }

  &.is-on {
    background: color-mix(in srgb, var(--td-brand-color) 12%, transparent);
    color: var(--td-brand-color-6, var(--td-brand-color));

    .status-dot {
      box-shadow: 0 0 0 3px color-mix(in srgb, var(--td-brand-color) 22%, transparent);
    }
  }

  &.is-workspaceOff {
    background: color-mix(in srgb, var(--td-warning-color) 12%, transparent);
    color: var(--td-warning-color-6, var(--td-warning-color));
  }
}

:root[theme-mode='dark'] .status-pill {
  &.is-on {
    color: var(--td-brand-color);
  }

  &.is-workspaceOff {
    color: var(--td-warning-color-4, var(--td-warning-color));
  }
}

.overview-switch-row {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.overview-copy {
  flex: 1;
  min-width: 0;
}

.overview-label {
  display: block;
  margin: 0 0 4px;
  font-size: var(--app-text-lg);
  font-weight: 500;
  line-height: 1.35;
  color: var(--td-text-color-primary);
}

.overview-desc {
  margin: 0;
  max-width: 520px;
  font-size: var(--app-text-md);
  line-height: 1.55;
  color: var(--td-text-color-secondary);
}

.overview-switch {
  flex-shrink: 0;
  margin-top: 2px;
}

.overview-hint {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  margin: 0;
  max-width: 560px;
  font-size: var(--app-text-sm);
  line-height: 1.55;
  color: var(--td-text-color-placeholder);

  .t-icon {
    flex-shrink: 0;
    margin-top: 2px;
  }
}

.notice {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 10px 12px;
  border-radius: var(--app-radius-md);
  background: color-mix(in srgb, var(--td-warning-color) 9%, transparent);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--td-warning-color) 18%, transparent);
  color: var(--td-text-color-primary);
  font-size: var(--app-text-md);
  line-height: 1.5;

  .t-icon {
    flex-shrink: 0;
    margin-top: 1px;
    color: var(--td-warning-color);
  }
}

.overview-metric {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  justify-content: center;
  min-width: 120px;
  padding-left: 22px;
  border-left: 1px solid var(--mem-ring);
}

.metric-value {
  font-family: var(--app-font-heading);
  font-size: 44px;
  font-weight: 500;
  line-height: 1;
  letter-spacing: -0.03em;
  font-variant-numeric: tabular-nums;
  color: var(--td-text-color-primary);
}

.metric-label {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  margin-top: 8px;
  font-size: var(--app-text-sm);
  color: var(--td-text-color-placeholder);
  white-space: nowrap;

  .t-icon {
    color: var(--td-brand-color);
  }
}

.overview-stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 6px;
  margin-top: 6px;
}

.stat-tile {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  padding: 10px 12px;
  border: none;
  border-radius: calc(var(--app-radius-2xl) - 8px);
  background: transparent;
  color: var(--td-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: background-color var(--app-motion-base) @ease, box-shadow var(--app-motion-base) @ease;

  &:hover {
    background: var(--mem-card);
    box-shadow: 0 0 0 1px var(--mem-ring);

    .stat-arrow {
      opacity: 1;
      transform: translateX(0);
    }
  }

  &:focus-visible {
    outline: 2px solid var(--td-brand-color-focus);
    outline-offset: 1px;
  }

  &.is-current {
    background: var(--mem-card);
    box-shadow: 0 0 0 1px var(--app-ring-soft);
  }

  &.is-attention .stat-icon {
    background: color-mix(in srgb, var(--td-warning-color) 14%, transparent);
    color: var(--td-warning-color);
  }

  &.is-attention .stat-count {
    color: var(--td-warning-color);
  }
}

.stat-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 30px;
  height: 30px;
  border-radius: 9px;
  background: var(--mem-card);
  box-shadow: 0 0 0 1px var(--mem-ring);
  color: var(--td-text-color-secondary);
}

.stat-text {
  display: flex;
  flex-direction: column;
  min-width: 0;
  flex: 1;
}

.stat-count {
  font-family: var(--app-font-heading);
  font-size: var(--app-text-lg);
  font-weight: 500;
  line-height: 1.2;
  font-variant-numeric: tabular-nums;
}

.stat-label {
  overflow: hidden;
  font-size: var(--app-text-sm);
  color: var(--td-text-color-placeholder);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.stat-arrow {
  flex-shrink: 0;
  color: var(--td-text-color-placeholder);
  opacity: 0;
  transform: translateX(-4px);
  transition: opacity var(--app-motion-base) @ease, transform var(--app-motion-base) @ease;
}

/* List */
.list-section {
  margin-top: 32px;
  animation: mem-rise 520ms @ease 80ms both;
}

.list-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 14px;
}

.list-title {
  display: flex;
  align-items: baseline;
  gap: 10px;

  h3 {
    margin: 0;
    font-family: var(--app-font-heading);
    font-size: var(--app-text-xl);
    font-weight: 500;
    letter-spacing: -0.01em;
    color: var(--td-text-color-primary);
  }
}

.list-count {
  font-size: var(--app-text-sm);
  font-variant-numeric: tabular-nums;
  color: var(--td-text-color-placeholder);
}

.list-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.action-btn {
  border-radius: 999px !important;
  padding-inline: 12px !important;
  transition: transform var(--app-motion-fast) @ease, background-color var(--app-motion-base) @ease;

  &:not(.t-is-disabled):active {
    transform: scale(0.97);
  }

  &.t-button--variant-outline:not(.t-is-disabled) {
    border-color: var(--td-component-stroke);
    background: var(--td-bg-color-container);
  }
}

.action-divider {
  width: 1px;
  height: 18px;
  margin: 0 4px;
  background: var(--td-component-stroke);
}

.segmented {
  display: flex;
  flex-wrap: wrap;
  gap: 2px;
  padding: 4px;
  margin-bottom: 14px;
  border-radius: 19px;
  background: var(--mem-shell);
  box-shadow: inset 0 0 0 1px var(--mem-ring);
}

.segment {
  display: inline-flex;
  align-items: center;
  flex-shrink: 0;
  gap: 6px;
  height: 30px;
  padding: 0 10px 0 12px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--td-text-color-secondary);
  font: inherit;
  font-size: var(--app-text-md);
  white-space: nowrap;
  cursor: pointer;
  transition: background-color var(--app-motion-base) @ease, color var(--app-motion-base) @ease,
    box-shadow var(--app-motion-base) @ease;

  &:hover {
    color: var(--td-text-color-primary);
  }

  &:focus-visible {
    outline: 2px solid var(--td-brand-color-focus);
    outline-offset: 1px;
  }

  &.is-active {
    background: var(--mem-card);
    color: var(--td-text-color-primary);
    box-shadow: 0 0 0 1px var(--mem-ring), 0 1px 3px rgba(0, 0, 0, 0.08);
  }

  &.is-attention .segment-count {
    background: color-mix(in srgb, var(--td-warning-color) 16%, transparent);
    color: var(--td-warning-color);
  }
}

.segment-count {
  min-width: 20px;
  padding: 1px 6px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--td-text-color-primary) 6%, transparent);
  font-size: var(--app-text-xs);
  font-variant-numeric: tabular-nums;
  line-height: 16px;
  text-align: center;
  color: var(--td-text-color-placeholder);

  .segment.is-active & {
    background: color-mix(in srgb, var(--td-brand-color) 12%, transparent);
    color: var(--td-brand-color);
  }
}

.list-loading {
  display: block;
}

.status-hint {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  margin: 0 0 12px;
  padding: 0 4px;
  font-size: var(--app-text-sm);
  line-height: 1.55;
  color: var(--td-text-color-secondary);

  .t-icon {
    flex-shrink: 0;
    margin-top: 2px;
    color: var(--td-text-color-placeholder);
  }
}

.memory-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.memory-item {
  position: relative;
  display: flex;
  align-items: flex-start;
  gap: 14px;
  padding: 14px 14px 14px 14px;
  border-radius: var(--app-radius-lg);
  background: var(--mem-card);
  box-shadow: 0 0 0 1px var(--mem-ring);
  animation: mem-rise 460ms @ease both;
  animation-delay: calc(min(var(--i, 0), 10) * 35ms);
  transition: box-shadow var(--app-motion-slow) @ease, transform var(--app-motion-slow) @ease;

  &:hover,
  &:focus-within {
    box-shadow: 0 0 0 1px var(--app-ring-soft), var(--mem-lift);
    transform: translateY(-1px);

    .memory-actions {
      opacity: 1;
    }
  }

  &.is-pending {
    background: color-mix(in srgb, var(--td-warning-color) 5%, var(--mem-card));
    box-shadow: 0 0 0 1px color-mix(in srgb, var(--td-warning-color) 22%, transparent);

    .item-icon {
      background: color-mix(in srgb, var(--td-warning-color) 12%, transparent);
      color: var(--td-warning-color);
    }
  }

  &.is-superseded,
  &.is-archived {
    background: transparent;
  }

  &.is-editing {
    box-shadow: 0 0 0 1px var(--app-ring-select), var(--mem-lift);
    transform: none;
  }
}

.item-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  border-radius: 10px;
  background: var(--mem-shell);
  box-shadow: inset 0 0 0 1px var(--mem-ring);
  color: var(--td-text-color-secondary);
}

.memory-main {
  flex: 1;
  min-width: 0;
  padding-top: 1px;
}

.memory-content {
  margin: 0 0 6px;
  font-size: var(--app-text-base);
  line-height: 1.6;
  color: var(--td-text-color-primary);
  word-break: break-word;

  &.is-title {
    font-weight: 500;
  }

  &.inactive {
    color: var(--td-text-color-placeholder);
    text-decoration: line-through;
    text-decoration-color: color-mix(in srgb, var(--td-text-color-placeholder) 60%, transparent);
  }
}

.memory-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px 0;
  font-size: var(--app-text-sm);
  line-height: 18px;
  color: var(--td-text-color-placeholder);
}

.meta-chip {
  display: inline-flex;
  align-items: center;
  margin-right: 8px;
  padding: 1px 8px;
  border-radius: 999px;
  background: var(--mem-shell);
  box-shadow: inset 0 0 0 1px var(--mem-ring);
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);
  font-weight: 500;
}

.meta-text + .meta-text::before {
  content: '';
  display: inline-block;
  width: 3px;
  height: 3px;
  margin: 0 8px;
  border-radius: 50%;
  background: currentColor;
  opacity: 0.6;
  vertical-align: middle;
}

.memory-topic {
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.topic-progress {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 2px 0 8px;
  max-width: 440px;

  .progress-track {
    position: relative;
    flex: 1;
    min-width: 80px;
    height: 6px;
    overflow: hidden;
    border-radius: 999px;
    background: var(--mem-shell);
    box-shadow: inset 0 0 0 1px var(--mem-ring);
  }

  .progress-fill {
    position: absolute;
    inset: 0;
    border-radius: inherit;
    background: color-mix(in srgb, var(--td-brand-color) 70%, var(--td-text-color-placeholder));
    transform-origin: left center;
    transition: transform var(--app-motion-slow) @ease;
  }

  &.is-ready .progress-fill {
    background: var(--td-brand-color);
  }

  .progress-text {
    flex-shrink: 1;
    min-width: 0;
    font-size: var(--app-text-sm);
    line-height: 18px;
    color: var(--td-text-color-placeholder);
  }
}

.memory-edit {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 8px;
}

.memory-edit-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.memory-actions {
  display: flex;
  align-items: center;
  flex-shrink: 0;
  gap: 4px;
  opacity: 0.35;
  transition: opacity var(--app-motion-base) @ease;

  &.is-persistent {
    opacity: 1;
  }
}

.guess-btn {
  border-radius: 999px !important;
  padding-inline: 10px !important;
}

.memory-pagination {
  margin-top: 16px;
}

.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 44px 24px;
  border-radius: var(--app-radius-xl);
  background: var(--mem-shell);
  box-shadow: inset 0 0 0 1px var(--mem-ring);
  text-align: center;
  animation: mem-rise 460ms @ease both;
}

.empty-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  margin-bottom: 14px;
  border-radius: 14px;
  background: var(--mem-card);
  box-shadow: 0 0 0 1px var(--mem-ring), var(--mem-inner-highlight), 0 6px 16px -8px rgba(0, 0, 0, 0.18);
  color: var(--td-text-color-secondary);
}

.empty-title {
  margin: 0 0 4px;
  font-size: var(--app-text-lg);
  font-weight: 500;
  color: var(--td-text-color-primary);
}

.empty-desc {
  max-width: 420px;
  margin: 0;
  font-size: var(--app-text-md);
  line-height: 1.55;
  color: var(--td-text-color-placeholder);
}

@keyframes mem-rise {
  from {
    opacity: 0;
    transform: translateY(8px);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (hover: none) {
  .memory-actions {
    opacity: 1;
  }
}

@container (max-width: 640px) {
  .overview-core {
    grid-template-columns: minmax(0, 1fr);
    padding: 18px;
  }

  .overview-metric {
    flex-direction: row;
    align-items: baseline;
    justify-content: flex-start;
    gap: 10px;
    padding: 14px 0 0;
    border-left: none;
    border-top: 1px solid var(--mem-ring);
  }

  .metric-value {
    font-size: 32px;
  }

  .overview-stats {
    grid-template-columns: minmax(0, 1fr);
  }

  .stat-arrow {
    opacity: 1;
    transform: none;
  }

  .memory-item {
    flex-wrap: wrap;
  }

  .memory-actions {
    width: 100%;
    justify-content: flex-end;
    opacity: 1;
  }
}

@media (prefers-reduced-motion: reduce) {
  .overview,
  .list-section,
  .memory-item,
  .empty {
    animation: none;
  }

  .memory-item,
  .stat-arrow,
  .progress-fill {
    transition: none;
  }

  .memory-item:hover {
    transform: none;
  }
}
</style>


<!-- t-popup renders into body, so the add form and usage popover have to be styled globally. -->
<style lang="less">
.memory-usage-popup-overlay {
  z-index: 3050 !important;

  .t-popup__content {
    padding: 0 !important;
    width: 380px;
    max-width: calc(100vw - 24px);
    border-radius: var(--app-radius-xl) !important;
    background: var(--td-bg-color-container) !important;
    border: 0.5px solid var(--td-component-stroke) !important;
    box-shadow: 0 0 0 0.5px rgba(0, 0, 0, 0.03), var(--app-popover-shadow) !important;
  }

  .usage-popup {
    padding: 14px 16px 12px;
    word-break: normal;
  }

  .usage-popup-title {
    font-size: var(--app-text-md);
    font-weight: 500;
    color: var(--td-text-color-primary);
  }

  .usage-popup-intro {
    margin: 4px 0 12px;
    font-size: var(--app-text-sm);
    line-height: 1.5;
    color: var(--td-text-color-placeholder);
  }

  .usage-popup-rows {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .usage-popup-row {
    display: flex;
    align-items: flex-start;
    gap: 12px;
    line-height: 1.5;
  }

  .usage-popup-label {
    flex: 0 0 88px;
    font-size: var(--app-text-sm);
    font-weight: 500;
    color: var(--td-text-color-primary);
  }

  .usage-popup-text {
    flex: 1;
    min-width: 0;
    font-size: var(--app-text-sm);
    color: var(--td-text-color-secondary);
  }
}

:root[theme-mode='dark'] .memory-usage-popup-overlay .t-popup__content {
  background: rgba(36, 36, 36, 0.92) !important;
  border-color: rgba(255, 255, 255, 0.08) !important;
  box-shadow:
    0 0 0 0.5px rgba(255, 255, 255, 0.05),
    0 2px 4px rgba(0, 0, 0, 0.12),
    0 8px 32px rgba(0, 0, 0, 0.28) !important;
}

.memory-add-popup-overlay {
  z-index: 3050;

  .t-popup__content {
    padding: 14px 16px !important;
    width: 320px;
    max-width: calc(100vw - 24px);
    border-radius: var(--app-radius-xl) !important;
    background: var(--td-bg-color-container) !important;
    border: 0.5px solid var(--td-component-stroke) !important;
    box-shadow: 0 0 0 0.5px rgba(0, 0, 0, 0.03), var(--app-popover-shadow) !important;
  }

  .add-popup {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .add-popup-title {
    font-size: var(--app-text-base);
    font-weight: 500;
    color: var(--td-text-color-primary);
  }

  .add-field {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .add-label {
    font-size: var(--app-text-sm);
    color: var(--td-text-color-secondary);
  }

  .add-kind-hint {
    font-size: var(--app-text-sm);
    line-height: 18px;
    color: var(--td-text-color-placeholder);
  }

  .add-popup-footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }
}

/* The kind dropdown mounts to body too, above the popup that opened it. */
.memory-add-kind-popup {
  z-index: 6200;
}

:root[theme-mode='dark'] .memory-add-popup-overlay .t-popup__content {
  background: rgba(36, 36, 36, 0.92) !important;
  border-color: rgba(255, 255, 255, 0.08) !important;
  box-shadow:
    0 0 0 0.5px rgba(255, 255, 255, 0.05),
    0 2px 4px rgba(0, 0, 0, 0.12),
    0 8px 32px rgba(0, 0, 0, 0.28) !important;
}
</style>
