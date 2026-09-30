<template>
  <div class="feedback-management">
    <header class="section-header">
      <h2>Sohbet geri bildirimleri</h2>
      <p class="section-description">Kaydedilmiş sohbetlerle ilgili değerlendirme, öneri ve şikâyetler.</p>
    </header>

    <div class="feedback-toolbar">
      <t-input v-model="filters.q" clearable class="feedback-toolbar__search" placeholder="Yorum, sohbet veya kullanıcı ara">
        <template #prefix-icon><t-icon name="search" /></template>
      </t-input>
      <t-date-range-picker v-model="filters.range" clearable allow-input class="feedback-toolbar__range"
        :placeholder="['Başlangıç', 'Bitiş']" :disable-date="disableFutureDate" />
      <t-select v-if="global" v-model="filters.workspace_id" clearable filterable class="feedback-toolbar__select"
        placeholder="Tüm çalışma alanları" :options="workspaceOptions" />
      <t-select v-model="filters.helpful" clearable class="feedback-toolbar__select feedback-toolbar__select--rating"
        placeholder="Tüm değerlendirmeler" :options="helpfulOptions" />
      <t-select v-model="filters.category" clearable class="feedback-toolbar__select feedback-toolbar__select--short"
        placeholder="Tüm türler" :options="categoryOptions" />
      <div class="feedback-toolbar__meta">
        <button v-if="hasFilters" type="button" class="feedback-toolbar__clear" @click="clearFilters">Filtreleri temizle</button>
        <span>{{ total }} sonuç</span>
        <button type="button" class="feedback-refresh" :disabled="loading" title="Yenile" aria-label="Yenile" @click="load">
          <t-icon :name="loading ? 'loading' : 'refresh'" :class="{ 'feedback-refresh--spin': loading }" />
        </button>
      </div>
    </div>

    <div v-if="error" class="feedback-state">
      <t-alert theme="error" :message="error">
        <template #operation>
          <t-button size="small" @click="load">Tekrar dene</t-button>
        </template>
      </t-alert>
    </div>

    <div v-else class="data-table-shell feedback-table-shell"
      :class="{ 'feedback-table-shell--single-page': total <= PAGE_SIZE }">
      <t-table row-key="id" :data="items" :columns="columns" :pagination="pagination" :loading="loading"
        disable-data-page size="medium" hover
        @row-click="({ row }: { row: Feedback }) => openDetail(row)"
        @page-change="(info: { current: number }) => goToPage(info.current)">
        <template #updated_at="{ row }">
          <span class="feedback-cell-text">{{ formatDate(row.updated_at) }}</span>
        </template>
        <template #user="{ row }">
          <div class="feedback-user">
            <span class="feedback-user__name">{{ row.user_name || row.user_id }}</span>
            <span v-if="row.user_email" class="feedback-user__email">{{ row.user_email }}</span>
          </div>
        </template>
        <template #workspace="{ row }">
          <span class="feedback-cell-text">{{ row.workspace_name || '—' }}</span>
        </template>
        <template #session="{ row }">
          <span class="feedback-cell-text feedback-ellipsis" :title="row.session_title || ''">{{ row.session_title || 'Sohbet' }}</span>
        </template>
        <template #helpful="{ row }">
          <t-tag v-if="row.helpful !== null" size="small" variant="light" :theme="row.helpful ? 'success' : 'danger'">
            {{ ratingLabel(row.helpful) }}
          </t-tag>
          <span v-else class="feedback-muted">{{ ratingLabel(row.helpful) }}</span>
        </template>
        <template #category="{ row }">
          <t-tag v-if="row.category" size="small" variant="light" :theme="row.category === 'suggestion' ? 'primary' : 'warning'">
            {{ categoryLabel(row.category) }}
          </t-tag>
          <span v-else class="feedback-muted">—</span>
        </template>
        <template #comment="{ row }">
          <span class="feedback-cell-text feedback-ellipsis" :title="row.comment || ''">{{ row.comment || '—' }}</span>
        </template>
        <template #actions="{ row }">
          <t-button variant="text" theme="primary" size="small" @click.stop="openDetail(row)">Görüntüle</t-button>
        </template>
        <template #empty>
          <t-empty description="Geri bildirim bulunamadı." />
        </template>
      </t-table>
    </div>

    <SettingDrawer :visible="detailVisible" :title="selected?.session_title || 'Sohbet'"
      :description="selected ? `${selected.user_name || selected.user_id} · ${formatDate(selected.updated_at)}` : ''"
      icon="chat" width="640px" storage-key="setting-drawer:width:feedback-detail" hide-footer
      @update:visible="detailVisible = $event">
      <template v-if="selected">
        <section class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">Geri bildirim</h4>
          <dl class="feedback-fields">
            <div><dt>Kullanıcı</dt><dd>{{ selected.user_name || selected.user_id }}<span v-if="selected.user_email" class="feedback-muted"> · {{ selected.user_email }}</span></dd></div>
            <div v-if="global"><dt>Çalışma alanı</dt><dd>{{ selected.workspace_name || '—' }}</dd></div>
            <div><dt>Gönderim</dt><dd>{{ formatDate(selected.updated_at) }}</dd></div>
            <div>
              <dt>Değerlendirme</dt>
              <dd class="feedback-fields__tags">
                <t-tag v-if="selected.helpful !== null" size="small" variant="light" :theme="selected.helpful ? 'success' : 'danger'">
                  {{ ratingLabel(selected.helpful) }}
                </t-tag>
                <span v-else class="feedback-muted">{{ ratingLabel(selected.helpful) }}</span>
                <t-tag v-if="selected.category" size="small" variant="light" :theme="selected.category === 'suggestion' ? 'primary' : 'warning'">
                  {{ categoryLabel(selected.category) }}
                </t-tag>
              </dd>
            </div>
          </dl>
          <p class="feedback-comment" :class="{ 'feedback-muted': !selected.comment }">{{ selected.comment || 'Yorum yok' }}</p>
        </section>
        <section class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">İlgili sohbet</h4>
          <div v-if="detailLoading" class="feedback-state feedback-state--loading"><t-loading size="small" /></div>
          <t-alert v-else-if="detailError" theme="error" :message="detailError" />
          <p v-else-if="!messages.length" class="feedback-muted">Sohbet içeriği bulunamadı.</p>
          <div v-else class="feedback-transcript">
            <article v-for="message in messages" :key="message.id" :class="`feedback-transcript__item--${message.role}`">
              <span class="feedback-transcript__role">{{ roleLabel(message.role) }}</span>
              <p>{{ message.content }}</p>
            </article>
          </div>
        </section>
      </template>
    </SettingDrawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import type { TableProps } from 'tdesign-vue-next'
import SettingDrawer from '@/components/settings/SettingDrawer.vue'
import { useAuthStore } from '@/stores/auth'
import { getFeedbackDetail, listFeedback, type Feedback } from '@/api/feedback'
import { searchTenants } from '@/api/tenant'

type Message = { id: string; role: string; content: string; created_at: string }
type Filters = { q: string; range: string[]; workspace_id?: string; helpful?: string; category?: string }

const PAGE_SIZE = 20
const auth = useAuthStore()
const global = computed(() => auth.isSystemAdmin)

const emptyFilters = (): Filters => ({ q: '', range: [], workspace_id: undefined, helpful: undefined, category: undefined })
const filters = reactive<Filters>(emptyFilters())
const hasFilters = computed(() => !!(filters.q.trim() || filters.range.length || filters.workspace_id || filters.helpful || filters.category))
const workspaces = ref<{ id: number; name: string }[]>([])
const workspaceOptions = computed(() => workspaces.value.map(w => ({ label: w.name || String(w.id), value: String(w.id) })))
const helpfulOptions = [{ label: 'Faydalı', value: 'true' }, { label: 'Faydalı değil', value: 'false' }]
const categoryOptions = [{ label: 'Öneri', value: 'suggestion' }, { label: 'Şikâyet', value: 'complaint' }]

const items = ref<Feedback[]>([])
const page = ref(1)
const total = ref(0)
const loading = ref(false)
const error = ref('')
let requestNumber = 0

const pagination = computed(() => ({ current: page.value, pageSize: PAGE_SIZE, total: total.value, showPageSize: false }))
const columns = computed<TableProps['columns']>(() => [
  { colKey: 'updated_at', title: 'Gönderim', width: 160 },
  { colKey: 'user', title: 'Kullanıcı', width: 200 },
  ...(global.value ? [{ colKey: 'workspace', title: 'Çalışma alanı', width: 150 }] : []),
  { colKey: 'session', title: 'Sohbet', width: 180 },
  { colKey: 'helpful', title: 'Değerlendirme', width: 130 },
  { colKey: 'category', title: 'Tür', width: 96 },
  { colKey: 'comment', title: 'Yorum' },
  { colKey: 'actions', title: '', width: 120, align: 'right' },
])

function formatDate(value: string) { return new Intl.DateTimeFormat('tr-TR', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)) }
function ratingLabel(value: boolean | null) { return value === true ? 'Faydalı' : value === false ? 'Faydalı değil' : 'Değerlendirilmedi' }
function categoryLabel(value: string) { return value === 'suggestion' ? 'Öneri' : 'Şikâyet' }
function roleLabel(role: string) { return role === 'user' ? 'Kullanıcı' : role === 'assistant' ? 'Asistan' : role }
function disableFutureDate(date: Date) { return date.getTime() > Date.now() }
function dayBoundary(value: string, nextDay: boolean) {
  const [year, month, day] = value.split('-').map(Number)
  return new Date(year, month - 1, day + (nextDay ? 1 : 0)).toISOString()
}

function clearFilters() { Object.assign(filters, emptyFilters()) }
function goToPage(next: number) { if (next === page.value) return; page.value = next; void load() }

// Typing in the search box waits briefly; every other filter reloads at once.
let searchTimer: ReturnType<typeof setTimeout> | undefined
watch(() => filters.q, () => {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => { page.value = 1; void load() }, 300)
})
watch(() => [filters.range, filters.workspace_id, filters.helpful, filters.category], () => { page.value = 1; void load() })
onBeforeUnmount(() => clearTimeout(searchTimer))

async function loadWorkspaces() {
  if (!global.value) return
  try { const res = await searchTenants({ page: 1, page_size: 100 }); workspaces.value = res.data?.items ?? [] } catch { workspaces.value = [] }
}

async function load() {
  const current = ++requestNumber
  loading.value = true
  error.value = ''
  const query = new URLSearchParams({ page: String(page.value), page_size: String(PAGE_SIZE) })
  const q = filters.q.trim()
  if (q) query.set('q', q)
  const [from, to] = filters.range
  if (from) query.set('from', dayBoundary(from, false))
  if (to) query.set('to', dayBoundary(to, true))
  for (const key of ['workspace_id', 'helpful', 'category'] as const) if (filters[key]) query.set(key, filters[key]!)
  try {
    const res = await listFeedback(global.value, query)
    if (current !== requestNumber) return
    items.value = res.data.items
    total.value = res.data.total
  } catch { if (current === requestNumber) error.value = 'Geri bildirimler yüklenemedi.' }
  finally { if (current === requestNumber) loading.value = false }
}

// ---------- Ayrıntı çekmecesi ----------

const detailVisible = ref(false)
const selected = ref<Feedback>()
const messages = ref<Message[]>([])
const detailLoading = ref(false)
const detailError = ref('')
let detailRequest = 0

async function openDetail(row: Feedback) {
  const current = ++detailRequest
  selected.value = row
  messages.value = []
  detailError.value = ''
  detailLoading.value = true
  detailVisible.value = true
  try {
    const res = await getFeedbackDetail(global.value, row.id)
    if (current !== detailRequest) return
    selected.value = res.data.feedback
    messages.value = res.data.messages
  } catch { if (current === detailRequest) detailError.value = 'Sohbet içeriği yüklenemedi.' }
  finally { if (current === detailRequest) detailLoading.value = false }
}

onMounted(() => { void load(); void loadWorkspaces() })
</script>

<style lang="less" scoped>
@import (reference) '@/components/css/settings-section.less';

.feedback-management {
  width: 100%;
}

.section-header {
  .settings-section-header();
}

.feedback-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.feedback-toolbar__search {
  width: 264px;
}

.feedback-toolbar__range {
  width: 284px;
}

.feedback-toolbar__select {
  width: 200px;
}

.feedback-toolbar__select--short {
  width: 180px;
}

.feedback-toolbar__select--rating {
  width: 210px;
}

.feedback-toolbar__meta {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  margin-left: auto;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  white-space: nowrap;
}

.feedback-toolbar__clear {
  padding: 0;
  border: none;
  background: transparent;
  color: var(--td-brand-color);
  font: inherit;
  cursor: pointer;

  &:hover {
    color: var(--td-brand-color-hover);
  }
}

.feedback-refresh {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  padding: 0;
  border: none;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: pointer;
  transition: color var(--app-motion-base) ease, background var(--app-motion-base) ease;

  &:hover:not(:disabled) {
    color: var(--td-brand-color);
    background: var(--td-bg-color-secondarycontainer);
  }

  &:disabled {
    cursor: default;
    opacity: 0.7;
  }
}

.feedback-refresh--spin {
  animation: wk-spin 0.8s linear infinite;
}

.feedback-state {
  min-height: 160px;
}

.feedback-state--loading {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 80px;
}

.data-table-shell {
  overflow-x: auto;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-lg);
  background-color: var(--td-bg-color-container);

  &:deep(thead th) {
    background-color: var(--td-bg-color-secondarycontainer);
    font-size: var(--app-text-md);
    font-weight: 600;
  }

  &:deep(.t-table td),
  &:deep(.t-table th) {
    padding-top: 12px;
    padding-bottom: 12px;
    vertical-align: middle;
  }

  &:deep(.t-table__pagination) {
    padding: 12px 16px;
  }
}

.feedback-table-shell {
  &:deep(.t-table tbody tr) {
    cursor: pointer;
  }

  &--single-page :deep(.t-table__pagination) {
    display: none;
  }

  &:deep(.t-table td:last-child .t-button) {
    white-space: nowrap;
  }
}

.feedback-user {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.feedback-user__name {
  color: var(--td-text-color-primary);
  font-size: var(--app-text-md);
  font-weight: 500;
  overflow-wrap: anywhere;
}

.feedback-user__email {
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  line-height: 1.4;
  overflow-wrap: anywhere;
}

.feedback-cell-text {
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-md);
}

.feedback-ellipsis {
  display: -webkit-box;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow-wrap: anywhere;
}

.feedback-muted {
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  font-weight: 400;
}

// ---------- Çekmece içeriği ----------

.feedback-fields {
  display: grid;
  gap: 10px;
  margin: 0;

  > div {
    display: grid;
    grid-template-columns: 120px 1fr;
    gap: 12px;
    align-items: center;
  }

  dt {
    color: var(--td-text-color-secondary);
    font-size: var(--app-text-sm);
  }

  dd {
    margin: 0;
    color: var(--td-text-color-primary);
    font-size: var(--app-text-md);
    overflow-wrap: anywhere;
  }
}

.feedback-fields__tags {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}

.feedback-comment {
  margin: 0;
  padding: 12px 14px;
  border-radius: var(--app-radius-md);
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-primary);
  font-size: var(--app-text-md);
  line-height: 1.6;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.feedback-transcript {
  display: flex;
  flex-direction: column;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-md);

  article {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 10px 14px;

    & + article {
      border-top: 1px solid var(--td-component-stroke);
    }

    p {
      margin: 0;
      color: var(--td-text-color-primary);
      font-size: var(--app-text-md);
      line-height: 1.6;
      white-space: pre-wrap;
      overflow-wrap: anywhere;
    }
  }
}

.feedback-transcript__role {
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-sm);
  font-weight: 500;
}

.feedback-transcript__item--user {
  background: var(--td-bg-color-secondarycontainer);
}

@media (max-width: 720px) {
  .feedback-toolbar__search,
  .feedback-toolbar__range,
  .feedback-toolbar__select,
  .feedback-toolbar__select--short,
  .feedback-toolbar__select--rating {
    width: 100%;
  }

  .feedback-toolbar__meta {
    margin-left: 0;
  }
}
</style>
