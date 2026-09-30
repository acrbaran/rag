<template>
  <div class="system-audit-log">
    <header class="section-header audit-page-header">
      <div class="audit-page-header__title">
        <div class="audit-page-header__row">
          <h2>{{ t('system.globalSettings.audit.tabLabel') }}</h2>
          <button
            type="button"
            class="rq-refresh"
            :disabled="auditLoading"
            :title="t('system.globalSettings.audit.refresh')"
            :aria-label="t('system.globalSettings.audit.refresh')"
            @click="reloadAuditLog"
          >
            <t-icon
              :name="auditLoading ? 'loading' : 'refresh'"
              :class="{ 'rq-refresh-spin': auditLoading }"
            />
          </button>
        </div>
        <p class="section-description">{{ t('system.globalSettings.audit.description') }}</p>
      </div>
    </header>

    <div class="audit-page-body">
      <div v-if="auditError" class="audit-page-branch audit-page-branch--error">
        <t-alert theme="error" :message="auditError">
          <template #operation>
            <t-button size="small" @click="reloadAuditLog">
              {{ t('system.globalSettings.audit.retry') }}
            </t-button>
          </template>
        </t-alert>
      </div>

      <div
        v-else-if="!auditLoading && auditEntries.length === 0"
        class="audit-page-branch audit-page-branch--empty"
      >
        <t-empty :description="t('system.globalSettings.audit.empty')" />
      </div>

      <div v-else class="audit-scroll-area narrow-scrollbar audit-page-branch" ref="auditScrollRoot">
        <div class="data-table-shell audit-table-shell">
          <t-table
            row-key="id"
            :data="auditEntries"
            :columns="auditColumns"
            size="medium"
            hover
            @row-click="openAuditDetail"
          >
            <template #created_at="{ row }">
              <div class="audit-time">
                <span class="audit-time-date">{{ formatAuditDatePart(row.created_at) }}</span>
                <span class="audit-time-clock">{{ formatAuditTimePart(row.created_at) }}</span>
              </div>
            </template>
            <template #actor="{ row }">
              <div class="audit-actor">
                <span class="audit-actor-name">
                  {{ row.actor_user_id ? auditActorLabel(row.actor_user_id) :
                    t('system.globalSettings.audit.systemActor') }}
                </span>
                <span v-if="row.actor_role" class="audit-actor-role">
                  {{ auditActorRoleLabel(row.actor_role) }}
                </span>
              </div>
            </template>
            <template #action="{ row }">
              <t-tag :theme="auditActionTheme(row.action)" size="small" variant="light-outline"
                class="audit-action-tag">
                {{ formatAuditAction(row.action) }}
              </t-tag>
            </template>
            <template #target="{ row }">
              <div class="audit-target">
                <template v-if="auditTargetViews.get(row.id)">
                  <span v-if="auditTargetViews.get(row.id)!.label" class="audit-target-label">
                    {{ auditTargetViews.get(row.id)!.label }}
                  </span>
                  <span v-if="auditTargetViews.get(row.id)!.code" class="audit-target-key">
                    {{ auditTargetViews.get(row.id)!.code }}
                  </span>
                  <span v-if="auditTargetViews.get(row.id)!.diff" class="audit-target-diff">
                    {{ auditTargetViews.get(row.id)!.diff }}
                  </span>
                </template>
                <span v-else class="audit-target-empty">—</span>
              </div>
            </template>
            <template #outcome="{ row }">
              <t-tag :theme="auditOutcomeTheme(row.outcome)" size="small" variant="light">
                {{ t('system.globalSettings.audit.outcome.' + row.outcome) }}
              </t-tag>
            </template>
          </t-table>
        </div>

        <div ref="auditLoadSentinelEl" class="audit-load-sentinel" aria-hidden="true" />

        <div v-if="auditLoading && auditEntries.length > 0" class="audit-loading-more">
          <t-loading size="small" />
          <span>{{ t('system.globalSettings.audit.loading') }}</span>
        </div>

        <p v-if="!auditHasMore && auditEntries.length > 0 && !auditLoading" class="audit-end-hint">
          {{ t('system.globalSettings.audit.end') }}
        </p>
      </div>
    </div>

    <SettingDrawer
      v-model:visible="auditDetailVisible"
      class="audit-detail-drawer"
      :title="auditDetailTitle"
      :description="auditDetailDescription"
      icon="file-paste"
      width="640px"
      :min-width="480"
      :max-width="960"
      storage-key="setting-drawer:width:system-audit-detail"
      hide-footer
    >
      <template v-if="selectedAuditEntry">
        <section class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">
            {{ t('system.globalSettings.audit.drawer.sectionSummary') }}
          </h4>
          <dl class="audit-detail-fields">
            <div
              v-for="field in auditSummaryFields(selectedAuditEntry)"
              :key="field.key"
              class="audit-detail-field"
            >
              <dt>{{ field.label }}</dt>
              <dd :title="field.value">
                {{ field.value }}
                <span v-if="field.code" class="audit-detail-code mono">{{ field.code }}</span>
              </dd>
            </div>
          </dl>
        </section>

        <section v-if="auditDetailFields(selectedAuditEntry).length > 0" class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">
            {{ t('system.globalSettings.audit.drawer.sectionDetails') }}
          </h4>
          <dl class="audit-detail-fields">
            <div
              v-for="field in auditDetailFields(selectedAuditEntry)"
              :key="field.key"
              class="audit-detail-field"
            >
              <dt :title="field.key">{{ field.label }}</dt>
              <dd :title="field.value">
                {{ field.value }}
                <span v-if="field.code" class="audit-detail-code mono">{{ field.code }}</span>
              </dd>
            </div>
          </dl>
        </section>

        <section v-if="auditIdentifierFields(selectedAuditEntry).length > 0" class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">
            {{ t('system.globalSettings.audit.drawer.sectionIdentifiers') }}
          </h4>
          <dl class="audit-detail-fields">
            <div
              v-for="field in auditIdentifierFields(selectedAuditEntry)"
              :key="field.key"
              class="audit-detail-field"
            >
              <dt>{{ field.label }}</dt>
              <dd class="mono" :title="field.value">{{ field.value }}</dd>
            </div>
          </dl>
        </section>

        <section v-if="auditRequestFields(selectedAuditEntry).length > 0" class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">
            {{ t('system.globalSettings.audit.drawer.sectionRequest') }}
          </h4>
          <dl class="audit-detail-fields">
            <div
              v-for="field in auditRequestFields(selectedAuditEntry)"
              :key="field.key"
              class="audit-detail-field"
            >
              <dt>{{ field.label }}</dt>
              <dd class="mono" :title="field.value">{{ field.value }}</dd>
            </div>
          </dl>
        </section>

        <section class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">
            {{ t('system.globalSettings.audit.expanded.details') }}
          </h4>
          <pre class="audit-detail-json mono">{{ auditDetailsJSON(selectedAuditEntry) }}</pre>
        </section>
      </template>
    </SettingDrawer>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  listSystemAuditLog,
  type AuditAction,
  type AuditLog,
  type AuditOutcome,
} from '@/api/system'
import SettingDrawer from '@/components/settings/SettingDrawer.vue'
import { AUDIT_ACTION_I18N_ROOTS } from '@/i18n/auditActionRegistry'
import { auditActionLabel } from '@/i18n/auditActionLabel'
import { useAuthStore } from '@/stores/auth'
import { PLATFORM_API_KEY_CAPABILITY_GROUPS } from '@/config/apiKeyCapabilities'
import { RUNTIME_TASK_TYPE_KEYS } from './runtimeTaskTypes'

interface AuditDetailField {
  key: string
  label: string
  value: string
  // Raw backend identifier shown under the readable value (setting key,
  // enum value, queue name …) so operators can still search for it.
  code?: string
}

// Readable label + raw identifier for the target column. `code` is empty
// when it would just repeat `label`.
interface AuditTargetView {
  label: string
  code: string
  diff: string
}

const authStore = useAuthStore()
const { t, tm, te, locale } = useI18n()

const auditEntries = ref<AuditLog[]>([])
const auditLoading = ref(false)
const auditError = ref('')
const auditCursor = ref<number>(0)
const auditHasMore = ref(true)
const AUDIT_PAGE_SIZE = 50

const auditScrollRoot = ref<HTMLElement | null>(null)
const auditLoadSentinelEl = ref<HTMLElement | null>(null)
let auditScrollObserver: IntersectionObserver | null = null

const auditDetailVisible = ref(false)
const selectedAuditEntry = ref<AuditLog | null>(null)

const auditColumns = computed(() => [
  { colKey: 'created_at', title: t('system.globalSettings.audit.columns.time'), width: 120 },
  { colKey: 'actor', title: t('system.globalSettings.audit.columns.actor'), width: 180 },
  // Fits the longest localized label on one line ("Platform API anahtarı
  // iptal edildi"); .audit-action-tag wraps anything longer instead of clipping.
  { colKey: 'action', title: t('system.globalSettings.audit.columns.action'), width: 250 },
  {
    colKey: 'target',
    title: t('system.globalSettings.audit.columns.target'),
    minWidth: 240,
  },
  { colKey: 'outcome', title: t('system.globalSettings.audit.columns.outcome'), width: 80, align: 'center' as const },
])

function formatAuditDatePart(s: string | undefined): string {
  if (!s) return '-'
  try {
    return new Intl.DateTimeFormat(locale.value || 'tr-TR', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
    }).format(new Date(s))
  } catch {
    return s
  }
}

function formatAuditTimePart(s: string | undefined): string {
  if (!s) return ''
  try {
    return new Intl.DateTimeFormat(locale.value || 'tr-TR', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    }).format(new Date(s))
  } catch {
    return ''
  }
}

function auditActionTheme(
  action: AuditAction,
): 'success' | 'warning' | 'danger' | 'primary' | 'default' {
  switch (action) {
    case 'system.admin_promoted':
      return 'success'
    case 'system.admin_revoked':
    case 'system.setting_changed':
    case 'system.queue_task_retried':
    case 'system.queue_task_run_now':
    case 'system.workspace_updated':
      return 'warning'
    case 'system.user_password_reset':
    case 'system.queue_task_deleted':
    case 'system.queue_task_cancelled':
    case 'system.queue_archived_purged':
    case 'system.workspace_deleted':
      return 'danger'
    case 'rbac.access_denied':
      return 'danger'
    default:
      return 'default'
  }
}

function auditOutcomeTheme(o: AuditOutcome): 'success' | 'danger' | 'default' {
  if (o === 'denied') return 'danger'
  if (o === 'success') return 'success'
  return 'default'
}

function formatAuditAction(action: AuditAction): string {
  return auditActionLabel({ tm }, AUDIT_ACTION_I18N_ROOTS.systemGlobal, action)
}

function auditActorLabel(userId: string): string {
  const me = authStore.user
  if (me && me.id === userId) {
    return me.username?.trim() || me.email?.trim() || userId.slice(0, 8)
  }
  return userId.slice(0, 8)
}

function auditActorRoleLabel(role: string): string {
  const key = `system.globalSettings.audit.actorRole.${role}`
  if (te(key)) return t(key)
  return role
}

// Look up an optional translation; undefined when the key is missing so
// callers can fall back to the raw value.
function optionalT(path: string): string | undefined {
  return te(path) ? (t(path) as string) : undefined
}

// Friendly label for a system setting key, shared with the settings page
// (system.globalSettings.keyLabels.<key>).
function settingKeyLabel(key: string): string {
  if (key === 'model_catalog') {
    return t('system.globalSettings.audit.target.modelCatalog')
  }
  return optionalT(`system.globalSettings.keyLabels.${key}`) ?? key
}

function queueLabel(queue: string): string {
  return optionalT(`system.globalSettings.runtime.queueNames.${queue}`) ?? queue
}

function taskTypeLabel(type: string): string {
  const key = RUNTIME_TASK_TYPE_KEYS[type]
  return (key && optionalT(`system.globalSettings.runtime.tasks.taskTypes.${key}`)) || type
}

function taskStateLabel(state: string): string {
  return optionalT(`system.globalSettings.runtime.tasks.states.${state}`) ?? state
}

function roleLabel(role: string): string {
  return optionalT(`tenantMember.role.${role}`)
    ?? optionalT(`system.globalSettings.audit.actorRole.${role}`)
    ?? role
}

const capabilityLabelKeys = new Map(
  PLATFORM_API_KEY_CAPABILITY_GROUPS.flatMap(group =>
    group.capabilities.map(cap => [cap.value as string, cap.labelKey] as const),
  ),
)

function capabilityLabel(cap: string): string {
  const key = capabilityLabelKeys.get(cap)
  return (key && optionalT(key)) || cap
}

function runtimeActionLabel(action: string): string {
  const leaf = { cancel: 'cancel', run_now: 'runNow', delete: 'deleteRecord' }[action]
  return (leaf && optionalT(`system.globalSettings.runtime.tasks.${leaf}`)) || action
}

function auditDetailsObject(row: AuditLog): Record<string, unknown> | null {
  if (row.details && typeof row.details === 'object') {
    return row.details as Record<string, unknown>
  }
  if (typeof row.details === 'string' && row.details) {
    try {
      const parsed = JSON.parse(row.details)
      return parsed && typeof parsed === 'object' ? parsed as Record<string, unknown> : null
    } catch {
      return null
    }
  }
  return null
}

function detailString(details: Record<string, unknown> | null, key: string): string {
  const v = details?.[key]
  return typeof v === 'string' ? v : ''
}

// The backend stringifies some flags ("true") in queue-task payloads.
function isTruthyFlag(v: unknown): boolean {
  return v === true || v === 'true'
}

const VALUE_MAX_LEN = 80
function truncate(s: string): string {
  return s.length > VALUE_MAX_LEN ? s.slice(0, VALUE_MAX_LEN - 1) + '…' : s
}

function rawValueString(v: unknown): string {
  if (v === null || v === undefined) return ''
  if (typeof v === 'string') return v
  if (typeof v === 'number' || typeof v === 'boolean') return String(v)
  try {
    return JSON.stringify(v)
  } catch {
    return String(v)
  }
}

// Readable rendering of a system setting value: enum options use the
// settings page labels, booleans become On/Off, lists are joined.
function settingValueLabel(settingKey: string, v: unknown): string {
  if (v === null || v === undefined) {
    return t('system.globalSettings.audit.target.valueNull')
  }
  if (typeof v === 'boolean') {
    return t(v ? 'system.globalSettings.audit.value.on' : 'system.globalSettings.audit.value.off')
  }
  if (typeof v === 'string') {
    if (!v) return t('system.globalSettings.audit.value.empty')
    return optionalT(`system.globalSettings.enumLabels.${settingKey}.${v}`) ?? v
  }
  if (Array.isArray(v)) {
    return v.length ? v.map(String).join(', ') : t('system.globalSettings.audit.value.empty')
  }
  return rawValueString(v)
}

function formatSettingDiff(settingKey: string, details: Record<string, unknown>): string {
  const oldStr = truncate(settingValueLabel(settingKey, details.old_value))
  const newStr = truncate(settingValueLabel(settingKey, details.new_value))
  if (oldStr === newStr) return ''
  return `${oldStr} → ${newStr}`
}

function auditTargetUser(row: AuditLog, details: Record<string, unknown> | null): string {
  const name = detailString(details, 'target_username')
  const mail = detailString(details, 'target_email')
  if (name && mail) return `${name} (${mail})`
  return name || mail || (row.target_user_id ? row.target_user_id.slice(0, 8) : '')
}

function systemRoleLabel(role: string): string {
  const key = `system.globalSettings.audit.systemRole.${role}`
  return te(key) ? t(key) : role
}

// Short notes under the user target for user lifecycle events.
function auditUserNotes(row: AuditLog, details: Record<string, unknown>): string[] {
  const notes: string[] = []
  switch (row.action) {
    case 'system.admin_promoted':
      if (details.idempotent === true) notes.push(t('system.globalSettings.audit.target.promoteIdempotent'))
      break
    case 'system.admin_revoked':
      if (details.changed === false) notes.push(t('system.globalSettings.audit.target.revokeNoop'))
      break
    case 'system.user_created':
      if (details.idempotent === true) notes.push(t('system.globalSettings.audit.target.userExisted'))
      if (details.password_generated === true) notes.push(t('system.globalSettings.audit.target.passwordGenerated'))
      break
    case 'system.user_deleted':
      if (typeof details.memberships_removed === 'number') {
        notes.push(t('system.globalSettings.audit.target.membershipsRemoved', {
          count: String(details.memberships_removed),
        }))
      }
      break
    case 'system.user_role_changed':
      if (typeof details.from === 'string' && typeof details.to === 'string') {
        notes.push(t('system.globalSettings.audit.target.roleChange', { from: systemRoleLabel(details.from), to: systemRoleLabel(details.to) }))
      }
      if (details.changed === false) notes.push(t('system.globalSettings.audit.target.roleUnchanged'))
      break
    case 'system.user_password_reset':
      if (details.sessions_revoked === true) notes.push(t('system.globalSettings.audit.target.sessionsRevoked'))
      break
  }
  return notes
}

const USER_TARGET_ACTIONS = new Set<string>([
  'system.admin_promoted',
  'system.admin_revoked',
  'system.user_password_reset',
  'system.user_created',
  'system.user_deleted',
  'system.user_role_changed',
])

const QUEUE_TASK_ACTIONS = new Set<string>([
  'system.queue_task_retried',
  'system.queue_task_run_now',
  'system.queue_task_cancelled',
  'system.queue_task_deleted',
])

function auditTargetView(row: AuditLog): AuditTargetView | null {
  const details = auditDetailsObject(row)
  const view = (label: string, code = '', diff = ''): AuditTargetView | null => {
    if (!label && !code && !diff) return null
    return { label, code: code === label ? '' : code, diff }
  }

  if (row.action === 'system.setting_changed') {
    if (row.target_type === 'tenant_storage_quota') {
      const affected = details?.affected
      const gb = details?.quota_gb
      const diff = typeof affected === 'number' && typeof gb === 'number'
        ? t('system.globalSettings.audit.target.bulkQuotaDiff', { count: String(affected), gb: String(gb) })
        : ''
      return view(t('system.globalSettings.audit.target.bulkQuota'), 'tenant.default_storage_quota_gb', diff)
    }
    if (row.target_type === 'model_catalog') {
      const from = details?.previous_version
      const to = details?.version
      const diff = from !== undefined && to !== undefined
        ? t('system.globalSettings.audit.target.modelCatalogDiff', { from: String(from), to: String(to) })
        : ''
      return view(settingKeyLabel('model_catalog'), 'model_catalog', diff)
    }
    const key = detailString(details, 'key') || row.target_id || row.target_type || ''
    return view(settingKeyLabel(key), key, details ? formatSettingDiff(key, details) : '')
  }

  if (USER_TARGET_ACTIONS.has(row.action)) {
    const notes = details ? auditUserNotes(row, details) : []
    return view(auditTargetUser(row, details), '', notes.join(' · '))
  }

  if (QUEUE_TASK_ACTIONS.has(row.action)) {
    const queue = detailString(details, 'queue')
    const taskID = detailString(details, 'task_id') || row.target_id
    const taskType = detailString(details, 'task_type')
    const label = [taskType ? taskTypeLabel(taskType) : '', queue ? queueLabel(queue) : '']
      .filter(Boolean)
      .join(' · ')
    const notes: string[] = []
    const fromState = detailString(details, 'from_state')
    if (fromState) {
      notes.push(t('system.globalSettings.audit.target.fromState', { state: taskStateLabel(fromState) }))
    }
    if (isTruthyFlag(details?.orphan_purge)) notes.push(t('system.globalSettings.audit.target.orphanPurge'))
    const code = queue && taskID ? `${queue}:${taskID}` : taskID || queue
    return view(label, code, notes.join(' · '))
  }

  if (row.action === 'system.queue_archived_purged') {
    const queue = detailString(details, 'queue') || row.target_id || ''
    const deleted = detailString(details, 'deleted')
    const diff = deleted ? t('system.globalSettings.audit.target.purgedCount', { count: deleted }) : ''
    return view(queue ? queueLabel(queue) : '', queue, diff)
  }

  if (row.action === 'system.workspace_updated') {
    const from = detailString(details, 'old_name')
    const to = detailString(details, 'new_name')
    return view(to, row.target_id ? `tenant:${row.target_id}` : '', from && to ? `${from} → ${to}` : '')
  }

  if (row.action === 'system.workspace_deleted') {
    return view(detailString(details, 'workspace_name'), row.target_id ? `tenant:${row.target_id}` : '')
  }

  if (row.action === 'system.api_key_created' || row.action === 'system.api_key_revoked') {
    const caps = Array.isArray(details?.capabilities) ? (details!.capabilities as unknown[]).map(String) : []
    const diff = caps.length
      ? t('system.globalSettings.audit.target.capabilities', { list: caps.map(capabilityLabel).join(', ') })
      : ''
    return view(
      t('system.globalSettings.audit.target.platformApiKey', { id: row.target_id }),
      `api_key:${row.target_id}`,
      diff,
    )
  }

  if (row.action === 'rbac.access_denied') {
    const role = detailString(details, 'required_role')
    const diff = role ? t('system.globalSettings.audit.target.requiredRole', { role: roleLabel(role) }) : ''
    return view('', row.request_path || '', diff)
  }

  if (row.target_user_id) return view(auditTargetUser(row, details))
  if (row.target_id) {
    return view('', row.target_type ? `${row.target_type}:${row.target_id}` : row.target_id)
  }
  return null
}

const auditTargetViews = computed(() => {
  const map = new Map<number, AuditTargetView>()
  for (const row of auditEntries.value) {
    const v = auditTargetView(row)
    if (v) map.set(row.id, v)
  }
  return map
})

// Readable rendering of one `details` entry for the drawer. Returns the
// display value plus the raw value when the two differ.
function auditDetailValue(
  row: AuditLog,
  details: Record<string, unknown>,
  key: string,
  v: unknown,
): { value: string; code?: string } {
  const raw = rawValueString(v)
  const settingKey = detailString(details, 'key') || row.target_id
  let value: string
  switch (key) {
    case 'key':
      value = settingKeyLabel(raw)
      break
    case 'value_type':
      value = optionalT(`system.globalSettings.audit.valueTypes.${raw}`) ?? raw
      break
    case 'old_value':
    case 'new_value':
      value = settingValueLabel(settingKey, v)
      break
    case 'queue':
      value = queueLabel(raw)
      break
    case 'task_type':
      value = taskTypeLabel(raw)
      break
    case 'from_state':
      value = taskStateLabel(raw)
      break
    case 'action':
      value = runtimeActionLabel(raw)
      break
    case 'scope_type':
      value = optionalT(`system.globalSettings.audit.scopeTypes.${raw}`) ?? raw
      break
    case 'required_role':
      value = roleLabel(raw)
      break
    case 'capabilities':
      value = Array.isArray(v) && v.length
        ? v.map(c => capabilityLabel(String(c))).join(', ')
        : t('system.globalSettings.audit.value.empty')
      return { value, code: Array.isArray(v) && v.length ? v.map(String).join(', ') : undefined }
    case 'quota_bytes':
      value = typeof v === 'number' ? `${new Intl.NumberFormat(locale.value).format(v)} B` : raw
      return { value }
    default:
      if (typeof v === 'boolean' || v === 'true' || v === 'false') {
        value = t(isTruthyFlag(v) ? 'system.globalSettings.audit.value.yes' : 'system.globalSettings.audit.value.no')
        return { value }
      }
      if (v === null || v === undefined) {
        value = t('system.globalSettings.audit.target.valueNull')
        return { value }
      }
      return { value: raw }
  }
  return { value, code: raw && raw !== value ? raw : undefined }
}

function auditDetailFields(row: AuditLog): AuditDetailField[] {
  const details = auditDetailsObject(row)
  if (!details) return []
  return Object.entries(details).map(([key, v]) => {
    const { value, code } = auditDetailValue(row, details, key, v)
    return {
      key,
      label: optionalT(`system.globalSettings.audit.detailFields.${key}`) ?? key,
      value,
      code,
    }
  })
}

function formatAuditDateTime(s: string | undefined): string {
  if (!s) return '—'
  const date = formatAuditDatePart(s)
  const time = formatAuditTimePart(s)
  return time ? `${date} ${time}` : date
}

function auditActorDisplay(row: AuditLog): string {
  if (!row.actor_user_id) {
    return t('system.globalSettings.audit.systemActor')
  }
  const name = auditActorLabel(row.actor_user_id)
  return row.actor_role ? `${name} (${auditActorRoleLabel(row.actor_role)})` : name
}

function auditSummaryFields(row: AuditLog): AuditDetailField[] {
  const fields: AuditDetailField[] = [
    {
      key: 'time',
      label: t('system.globalSettings.audit.columns.time'),
      value: formatAuditDateTime(row.created_at),
    },
    {
      key: 'actor',
      label: t('system.globalSettings.audit.columns.actor'),
      value: auditActorDisplay(row),
    },
    {
      key: 'action',
      label: t('system.globalSettings.audit.columns.action'),
      value: formatAuditAction(row.action),
    },
    {
      key: 'outcome',
      label: t('system.globalSettings.audit.columns.outcome'),
      value: t('system.globalSettings.audit.outcome.' + row.outcome),
    },
  ]

  const target = auditTargetView(row)
  if (target && (target.label || target.code)) {
    fields.push({
      key: 'target',
      label: t('system.globalSettings.audit.columns.target'),
      value: target.label || target.code,
      code: target.label ? target.code || undefined : undefined,
    })
  }
  if (target?.diff) {
    fields.push({
      key: 'targetDiff',
      label: t('system.globalSettings.audit.drawer.targetChange'),
      value: target.diff,
    })
  }

  return fields
}

function auditIdentifierFields(row: AuditLog): AuditDetailField[] {
  const fields: AuditDetailField[] = []
  if (row.actor_user_id) {
    fields.push({
      key: 'actorId',
      label: t('system.globalSettings.audit.expanded.actorId'),
      value: row.actor_user_id,
    })
  }
  if (row.target_user_id) {
    fields.push({
      key: 'targetUserId',
      label: t('system.globalSettings.audit.expanded.targetUserId'),
      value: row.target_user_id,
    })
  }
  if (row.target_type) {
    fields.push({
      key: 'targetType',
      label: t('system.globalSettings.audit.expanded.targetType'),
      value: row.target_type,
    })
  }
  if (row.target_id) {
    fields.push({
      key: 'targetId',
      label: t('system.globalSettings.audit.expanded.targetId'),
      value: row.target_id,
    })
  }
  return fields
}

function auditRequestFields(row: AuditLog): AuditDetailField[] {
  const fields: AuditDetailField[] = []
  if (row.request_method) {
    fields.push({
      key: 'method',
      label: t('system.globalSettings.audit.drawer.requestMethod'),
      value: row.request_method,
    })
  }
  if (row.request_path) {
    fields.push({
      key: 'path',
      label: t('system.globalSettings.audit.columns.path'),
      value: row.request_path,
    })
  }
  return fields
}

const auditDetailTitle = computed(() =>
  selectedAuditEntry.value ? formatAuditAction(selectedAuditEntry.value.action) : '',
)

const auditDetailDescription = computed(() =>
  selectedAuditEntry.value ? formatAuditDateTime(selectedAuditEntry.value.created_at) : '',
)

function openAuditDetail(context: { row: AuditLog }) {
  selectedAuditEntry.value = context.row
  auditDetailVisible.value = true
}

function auditDetailsJSON(row: AuditLog): string {
  if (row.details === null || row.details === undefined) return '{}'
  if (typeof row.details === 'string') return row.details
  try {
    return JSON.stringify(row.details, null, 2)
  } catch {
    return String(row.details)
  }
}

async function loadAuditLog(reset: boolean) {
  if (auditLoading.value) return
  if (!reset && !auditHasMore.value) return

  auditLoading.value = true
  auditError.value = ''
  try {
    const resp = await listSystemAuditLog({
      after_id: reset ? undefined : auditCursor.value || undefined,
      limit: AUDIT_PAGE_SIZE,
    })
    if (resp.success) {
      const rows = resp.data || []
      auditEntries.value = reset ? rows : [...auditEntries.value, ...rows]
      auditCursor.value = resp.next_cursor || 0
      auditHasMore.value = !!resp.next_cursor && rows.length > 0
    } else {
      auditError.value = resp.message || t('system.globalSettings.audit.errors.generic')
    }
  } catch (err: any) {
    const status = err?.status
    if (status === 403) {
      auditError.value = t('system.globalSettings.audit.forbidden')
    } else {
      auditError.value = err?.message || t('system.globalSettings.audit.errors.generic')
    }
  } finally {
    auditLoading.value = false
  }
}

function detachAuditInfiniteScroll() {
  auditScrollObserver?.disconnect()
  auditScrollObserver = null
}

function attachAuditInfiniteScroll() {
  detachAuditInfiniteScroll()
  const root = auditScrollRoot.value
  const sentinel = auditLoadSentinelEl.value
  if (!root || !sentinel) return

  auditScrollObserver = new IntersectionObserver(
    (entries) => {
      const hitBottom = entries.some((e) => e.isIntersecting)
      if (!hitBottom || !auditHasMore.value || auditLoading.value) return
      void loadAuditLog(false)
    },
    { root, rootMargin: '100px 0px', threshold: 0 },
  )
  auditScrollObserver.observe(sentinel)
}

function reloadAuditLog() {
  auditCursor.value = 0
  auditHasMore.value = true
  void loadAuditLog(true)
}

watch(
  () => [auditEntries.value.length, auditError.value],
  async () => {
    await nextTick()
    if (auditError.value) {
      detachAuditInfiniteScroll()
      return
    }
    attachAuditInfiniteScroll()
  },
  { flush: 'post' },
)

onMounted(async () => {
  await loadAuditLog(true)
  await nextTick()
  attachAuditInfiniteScroll()
})

onUnmounted(() => {
  detachAuditInfiniteScroll()
})
</script>

<style lang="less" scoped>
@import (reference) '@/components/css/settings-section.less';

.system-audit-log {
  width: 100%;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.section-header {
  .settings-section-header();
}

.audit-page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.audit-page-header__row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.audit-page-header h2 {
  margin: 0;
  font-size: var(--app-text-3xl);
  font-weight: 500;
  color: var(--td-text-color-primary);
}

.rq-refresh {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 20px;
  height: 20px;
  padding: 0;
  border: none;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: pointer;
  transition: color var(--app-motion-base) cubic-bezier(0.16, 1, 0.3, 1), background var(--app-motion-base) cubic-bezier(0.16, 1, 0.3, 1);

  :deep(.t-icon) {
    font-size: var(--app-text-sm);
  }

  &:hover:not(:disabled) {
    color: var(--td-brand-color);
    background: var(--td-bg-color-secondarycontainer);
  }

  &:active:not(:disabled) {
    background: var(--td-bg-color-secondarycontainer);
  }

  &:disabled {
    cursor: default;
    opacity: 0.7;
  }
}

.rq-refresh-spin {
  animation: wk-spin 0.8s linear infinite;
}

.audit-page-body {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.audit-page-branch {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.audit-page-branch--error {
  justify-content: flex-start;
}

.audit-page-branch--empty {
  justify-content: center;
  align-items: center;
  min-height: 280px;
}

.audit-scroll-area {
  flex: 1 1 auto;
  min-height: 0;
  max-height: calc(100vh - 260px);
  overflow-x: hidden;
  overflow-y: auto;
}

.audit-load-sentinel {
  height: 1px;
  width: 100%;
  pointer-events: none;
}

.audit-loading-more {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 12px;
  font-size: var(--app-text-sm);
  color: var(--td-text-color-secondary);
}

.audit-end-hint {
  text-align: center;
  font-size: var(--app-text-sm);
  color: var(--td-text-color-disabled);
  padding: 8px 0 14px;
  margin: 0;
}

.audit-time {
  display: flex;
  flex-direction: column;
  gap: 2px;
  line-height: 1.3;

  .audit-time-date {
    font-size: var(--app-text-sm);
    color: var(--td-text-color-secondary);
  }

  .audit-time-clock {
    font-size: var(--app-text-md);
    font-weight: 500;
    color: var(--td-text-color-primary);
    font-variant-numeric: tabular-nums;
  }
}

.audit-actor {
  display: flex;
  flex-direction: column;
  gap: 2px;
  line-height: 1.3;
  min-width: 0;

  .audit-actor-name {
    font-size: var(--app-text-md);
    font-weight: 500;
    color: var(--td-text-color-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .audit-actor-role {
    font-size: var(--app-text-sm);
    color: var(--td-text-color-secondary);
  }
}

.audit-target {
  display: flex;
  flex-direction: column;
  gap: 4px;
  line-height: 1.35;
  min-width: 0;
  padding: 2px 0;

  .audit-target-label {
    font-size: var(--app-text-md);
    font-weight: 500;
    color: var(--td-text-color-primary);
    word-break: break-word;
  }

  .audit-target-key {
    font-size: var(--app-text-xs);
    color: var(--td-text-color-placeholder);
    word-break: break-all;
    font-family: var(--td-font-family-mono);
  }

  .audit-target-label + .audit-target-key {
    margin-top: -2px;
  }

  .audit-target-diff {
    font-size: var(--app-text-sm);
    color: var(--td-text-color-secondary);
    word-break: break-word;
    line-height: 1.4;
  }

  .audit-target-empty {
    color: var(--td-text-color-placeholder);
  }
}

.audit-detail-fields {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin: 0;
}

.audit-detail-field {
  display: grid;
  grid-template-columns: 136px minmax(0, 1fr);
  gap: 12px;
  align-items: baseline;
  margin: 0;

  dt {
    margin: 0;
    color: var(--td-text-color-placeholder);
    font-size: var(--app-text-sm);
    line-height: 1.45;
    white-space: nowrap;
  }

  dd {
    margin: 0;
    color: var(--td-text-color-primary);
    font-size: var(--app-text-md);
    line-height: 1.55;
    word-break: break-word;
  }
}

.audit-detail-code {
  display: block;
  font-size: var(--app-text-xs);
  color: var(--td-text-color-placeholder);
  word-break: break-all;
  line-height: 1.4;
}

.audit-detail-json {
  margin: 0;
  padding: 12px 14px;
  font-size: var(--app-text-sm);
  line-height: 1.55;
  color: var(--td-text-color-primary);
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-md);
  white-space: pre-wrap;
  word-break: break-all;
  max-height: min(420px, 50vh);
  overflow: auto;
}

.mono {
  font-family: var(--td-font-family-mono);
}

.data-table-shell {
  overflow-x: auto;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--td-component-stroke);
  background-color: var(--td-bg-color-container);

  &:deep(thead th) {
    font-weight: 600;
    font-size: var(--app-text-md);
    background-color: var(--td-bg-color-secondarycontainer) !important;
  }

  &:deep(.t-table td),
  &:deep(.t-table th) {
    padding-top: 14px;
    padding-bottom: 14px;
    vertical-align: middle;
  }
}

.audit-table-shell {
  &:deep(thead th) {
    position: sticky;
    top: 0;
    z-index: 2;
    box-shadow: inset 0 -1px 0 var(--td-component-stroke);
  }

  &:deep(.t-table tbody tr) {
    cursor: pointer;
  }

  &:deep(.t-table tbody tr:hover > td) {
    background-color: var(--td-bg-color-container-hover);
  }

  // t-tag is a fixed-height nowrap pill by default, so a label wider than
  // the cell was clipped at the column edge. Let it grow and wrap instead.
  &:deep(.audit-action-tag) {
    max-width: 100%;
    height: auto;
    min-height: 18px;
    white-space: normal;
    text-align: left;
    line-height: 16px;
    padding-top: 1px;
    padding-bottom: 1px;

    .t-tag--text {
      overflow: visible;
      white-space: normal;
      text-overflow: clip;
    }
  }
}
</style>
