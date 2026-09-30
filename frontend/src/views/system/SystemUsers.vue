<template>
  <div class="system-users">
    <header class="section-header">
      <h2>Kullanıcılar</h2>
      <p class="section-description">Sisteme kayıtlı tüm kullanıcılar ve sistem rolleri. Çalışma alanı üyelikleri için Yönet'e tıklayın.</p>
    </header>

    <div class="users-toolbar">
      <t-input v-model="search" clearable class="users-toolbar__search" placeholder="Ad, e-posta veya telefon ara">
        <template #prefix-icon><t-icon name="search" /></template>
      </t-input>
      <div class="users-toolbar__meta">
        <span>{{ total }} kullanıcı</span>
        <button type="button" class="users-refresh" :disabled="loading" title="Yenile" aria-label="Yenile" @click="load()">
          <t-icon :name="loading ? 'loading' : 'refresh'" :class="{ 'users-refresh--spin': loading }" />
        </button>
      </div>
    </div>

    <div v-if="error" class="users-state">
      <t-alert theme="error" :message="error">
        <template #operation>
          <t-button size="small" @click="load()">Tekrar dene</t-button>
        </template>
      </t-alert>
    </div>

    <div v-else class="data-table-shell users-table-shell"
      :class="{ 'users-table-shell--single-page': total <= PAGE_SIZE }">
      <t-table row-key="id" :data="items" :columns="columns" :pagination="pagination" :loading="loading"
        disable-data-page size="medium" hover table-layout="fixed"
        @page-change="(info: { current: number }) => goToPage(info.current)">
        <template #user="{ row }">
          <div class="users-identity">
            <span class="users-identity__name" :title="displayName(row)">
              {{ displayName(row) }}
              <span v-if="row.id === auth.currentUserId" class="users-muted">(siz)</span>
            </span>
            <span class="users-identity__email" :title="row.email">{{ row.email }}</span>
          </div>
        </template>
        <template #phone="{ row }">
          <a v-if="row.phone" class="users-phone" :href="`tel:${row.phone}`">{{ formatPhone(row.phone) }}</a>
          <span v-else class="users-muted">—</span>
        </template>
        <template #system_role="{ row }">
          <t-tag size="small" variant="light" :theme="systemRoleMeta(row).theme">
            {{ systemRoleMeta(row).label }}
          </t-tag>
        </template>
        <template #status="{ row }">
          <t-tag size="small" variant="light" :theme="row.is_active ? 'success' : 'default'">
            {{ row.is_active ? 'Aktif' : 'Pasif' }}
          </t-tag>
        </template>
        <template #presence="{ row }">
          <div class="users-presence">
            <span class="users-presence__state" :class="{ 'users-presence__state--online': row.presence?.online }">
              <span class="users-presence__dot" aria-hidden="true" />
              {{ row.presence?.online ? 'Çevrimiçi' : 'Çevrimdışı' }}
            </span>
            <span class="users-presence__detail" :title="presenceTitle(row)">{{ presenceDetail(row) }}</span>
          </div>
        </template>
        <template #created_at="{ row }">
          <span class="users-cell-text">{{ formatDate(row.created_at) }}</span>
        </template>
        <template #actions="{ row }">
          <t-button variant="outline" theme="primary" size="small" @click.stop="openManage(row)">
            <template #icon><t-icon name="setting" /></template>
            Yönet
          </t-button>
        </template>
        <template #empty>
          <t-empty description="Kullanıcı bulunamadı." />
        </template>
      </t-table>
    </div>

    <SettingDrawer :visible="drawerVisible" title="Kullanıcı yönetimi"
      description="Hesap bilgileri, çalışma alanı üyelikleri ve platform yetkileri" icon="user-setting" width="640px"
      storage-key="setting-drawer:width:system-user-manage"
      @update:visible="onDrawerVisible">
      <template v-if="selected">
        <section class="setting-drawer__section">
          <div class="manage-profile">
            <div class="manage-profile__main">
              <div class="manage-avatar">
                <img v-if="selected.avatar" :src="selected.avatar" :alt="displayName(selected)" />
                <span v-else>{{ initials(selected) }}</span>
                <span class="manage-avatar__presence" :class="{ 'manage-avatar__presence--online': selected.presence?.online }"
                  :title="selected.presence?.online ? 'Çevrimiçi' : 'Çevrimdışı'" />
              </div>
              <div class="manage-profile__identity">
                <div class="manage-profile__name" :title="displayName(selected)">
                  <span class="manage-profile__name-text">{{ displayName(selected) }}</span>
                  <span v-if="isSelf" class="manage-profile__self">Siz</span>
                </div>
                <div class="manage-profile__email" :title="selected.email">{{ selected.email }}</div>
                <div class="manage-profile__tags">
                  <t-tag size="small" variant="light" :theme="systemRoleMeta(selected).theme">
                    {{ systemRoleMeta(selected).label }}
                  </t-tag>
                  <t-tag size="small" variant="light" :theme="selected.is_active ? 'success' : 'default'">
                    {{ selected.is_active ? 'Aktif hesap' : 'Pasif hesap' }}
                  </t-tag>
                </div>
              </div>
            </div>
            <dl class="manage-stats">
              <div>
                <dt>Çalışma alanı</dt>
                <dd>{{ sortedMemberships.length }}</dd>
              </div>
              <div>
                <dt>Kayıt</dt>
                <dd>{{ formatDay(selected.created_at) }}</dd>
              </div>
              <div>
                <dt>Bağlantı</dt>
                <dd :class="{ 'manage-stats__online': selected.presence?.online }" :title="presenceTitle(selected)">
                  {{ selected.presence?.online ? 'Çevrimiçi' : lastSeenShort(selected) }}
                </dd>
              </div>
            </dl>
          </div>
        </section>

        <section class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">Hesap bilgileri</h4>
          <dl class="manage-fields">
            <div class="manage-field">
              <dt><t-icon name="user-circle" />Kullanıcı adı</dt>
              <dd>
                <span class="manage-field__value">{{ selected.username || '—' }}</span>
                <button v-if="selected.username" type="button" class="manage-copy" title="Kopyala"
                  aria-label="Kullanıcı adını kopyala" @click="copy(selected.username, 'Kullanıcı adı')">
                  <t-icon name="copy" />
                </button>
              </dd>
            </div>
            <div class="manage-field">
              <dt><t-icon name="mail" />E-posta</dt>
              <dd>
                <a class="manage-field__value manage-link" :href="`mailto:${selected.email}`">{{ selected.email }}</a>
                <button type="button" class="manage-copy" title="Kopyala" aria-label="E-postayı kopyala"
                  @click="copy(selected.email, 'E-posta')">
                  <t-icon name="copy" />
                </button>
              </dd>
            </div>
            <div class="manage-field">
              <dt><t-icon name="call" />Telefon</dt>
              <dd>
                <template v-if="selected.phone">
                  <a class="manage-field__value manage-link" :href="`tel:${selected.phone}`">{{ formatPhone(selected.phone) }}</a>
                  <button type="button" class="manage-copy" title="Kopyala" aria-label="Telefonu kopyala"
                    @click="copy(selected.phone, 'Telefon')">
                    <t-icon name="copy" />
                  </button>
                </template>
                <span v-else class="users-muted">Kayıtlı değil</span>
              </dd>
            </div>
            <div class="manage-field">
              <dt><t-icon name="time" />Son etkinlik</dt>
              <dd class="manage-field__stack">
                <span class="users-presence__state" :class="{ 'users-presence__state--online': selected.presence?.online }">
                  <span class="users-presence__dot" aria-hidden="true" />
                  {{ selected.presence?.online ? 'Çevrimiçi' : 'Çevrimdışı' }}
                </span>
                <span class="users-muted">{{ presenceTitle(selected) }}</span>
              </dd>
            </div>
            <div class="manage-field">
              <dt><t-icon name="calendar" />Kayıt tarihi</dt>
              <dd><span class="manage-field__value">{{ formatDate(selected.created_at) }}</span></dd>
            </div>
            <div class="manage-field">
              <dt><t-icon name="fingerprint" />Kullanıcı kimliği</dt>
              <dd>
                <code class="manage-field__value manage-field__id" :title="selected.id">{{ selected.id }}</code>
                <button type="button" class="manage-copy" title="Kopyala" aria-label="Kullanıcı kimliğini kopyala"
                  @click="copy(selected.id, 'Kullanıcı kimliği')">
                  <t-icon name="copy" />
                </button>
              </dd>
            </div>
          </dl>
        </section>

        <section class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">
            Çalışma alanları
            <span class="manage-count">{{ sortedMemberships.length }}</span>
          </h4>
          <div v-if="!sortedMemberships.length" class="manage-empty">
            <t-icon name="layers" />
            <span>Bu kullanıcı henüz hiçbir çalışma alanına üye değil.</span>
          </div>
          <ul v-else class="manage-workspaces">
            <li v-for="m in sortedMemberships" :key="m.tenant_id">
              <span class="manage-workspaces__badge" aria-hidden="true">{{ workspaceInitial(m) }}</span>
              <span class="manage-workspaces__text">
                <span class="manage-workspaces__name" :title="m.tenant_name">{{ m.tenant_name || `Çalışma alanı #${m.tenant_id}` }}</span>
                <span class="manage-workspaces__id">#{{ m.tenant_id }}</span>
              </span>
              <t-tag size="small" variant="light" :theme="roleTheme(m.role)">{{ roleLabel(m.role) }}</t-tag>
            </li>
          </ul>
        </section>

        <section class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">
            Sistem rolü
            <RoleMatrixPopover title="Sistem rolü yetkileri"
              description="Sistem rolü, çalışma alanlarındaki rollerden bağımsızdır ve platform genelindeki yetkileri belirler."
              hint="Sistem rolü yetkilerini görüntüle" :roles="SYSTEM_ROLE_MATRIX" :highlight="currentRole"
              highlight-label="Mevcut" />
          </h4>
          <div v-if="isSelf" class="manage-notice">
            <t-icon name="info-circle" />
            <span>Kendi sistem rolünüzü değiştiremezsiniz. Bu işlem için başka bir sistem yöneticisine başvurun.</span>
          </div>
          <t-select v-model="roleDraft" class="users-role-select" aria-label="Sistem rolü"
            :disabled="isSelf || roleSubmitting">
            <t-option v-for="option in SYSTEM_ROLE_OPTIONS" :key="option.value" :value="option.value" :label="option.label">
              <span class="users-role-option">
                {{ option.label }}
                <span v-if="option.value === currentRole" class="users-role__current">Mevcut</span>
              </span>
            </t-option>
          </t-select>
          <p class="users-muted users-role-hint">{{ SYSTEM_ROLE_BY_VALUE[roleDraft].description }}</p>
        </section>

        <section class="setting-drawer__section">
          <div class="manage-danger">
            <span class="manage-danger__icon" aria-hidden="true"><t-icon name="error-circle" /></span>
            <div class="manage-danger__body">
              <div class="manage-danger__title">Kullanıcıyı sil</div>
              <p class="manage-danger__text">
                {{ isSelf
                  ? 'Kendi hesabınızı silemezsiniz.'
                  : 'Kullanıcı tüm çalışma alanlarından çıkarılır, oturumları sonlandırılır ve hesabı kalıcı olarak silinir. Bu işlem geri alınamaz.' }}
              </p>
            </div>
            <t-button theme="danger" variant="outline" :loading="deleteSubmitting" :disabled="isSelf || roleSubmitting"
              @click="confirmDelete">
              <template #icon><DeleteIcon /></template>
              Sil
            </t-button>
          </div>
        </section>
      </template>

      <template #footer-left>
        <span v-if="roleDirty && !isSelf" class="manage-unsaved">
          <span class="manage-unsaved__dot" aria-hidden="true" />
          Kaydedilmemiş rol değişikliği: <strong>{{ SYSTEM_ROLE_BY_VALUE[roleDraft].label }}</strong>
        </span>
      </template>
      <template #footer-right>
        <t-button v-if="roleDirty && !isSelf" theme="default" variant="outline" :disabled="roleSubmitting"
          @click="resetRoleDraft">Geri al</t-button>
        <t-button v-else theme="default" variant="outline" :disabled="busy" @click="onDrawerVisible(false)">Kapat</t-button>
        <t-button v-if="!isSelf" theme="primary" :disabled="!roleDirty" :loading="roleSubmitting" @click="confirmRoleChange">
          Rolü kaydet
        </t-button>
      </template>
    </SettingDrawer>
  </div>
</template>

<script setup lang="ts">
import { computed, h, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next'
import { DeleteIcon } from 'tdesign-icons-vue-next'
import type { TableProps } from 'tdesign-vue-next'
import { parsePhoneNumberFromString } from 'libphonenumber-js/max'
import SettingDrawer from '@/components/settings/SettingDrawer.vue'
import RoleMatrixPopover, { type RoleMatrixEntry } from '@/components/settings/RoleMatrixPopover.vue'
import { useAuthStore } from '@/stores/auth'
import {
  deleteSystemUser,
  listSystemUsers,
  setSystemUserRole,
  systemRoleOf,
  type SystemRole,
  type SystemUser,
  type SystemUserMembership,
} from '@/api/system'

const PAGE_SIZE = 20

const auth = useAuthStore()

const search = ref('')
const items = ref<SystemUser[]>([])
const page = ref(1)
const total = ref(0)
const loading = ref(false)
const error = ref('')
let requestNumber = 0

const pagination = computed(() => ({ current: page.value, pageSize: PAGE_SIZE, total: total.value, showPageSize: false }))
// Fixed layout: every column gets a share of the width instead of the user
// column absorbing all the slack; long names/emails ellipsize.
const columns: TableProps['columns'] = [
  { colKey: 'user', title: 'Kullanıcı', width: 240 },
  { colKey: 'phone', title: 'Telefon', width: 170 },
  { colKey: 'system_role', title: 'Sistem rolü', width: 190 },
  { colKey: 'status', title: 'Durum', width: 100 },
  { colKey: 'presence', title: 'Bağlantı', width: 200 },
  { colKey: 'created_at', title: 'Kayıt tarihi', width: 170 },
  { colKey: 'actions', title: '', width: 110, align: 'right' },
]

type TagTheme = 'default' | 'primary' | 'warning' | 'danger' | 'success'
const SYSTEM_ROLE_OPTIONS: { value: SystemRole; label: string; description: string; theme: TagTheme; icon: string }[] = [
  {
    value: 'user',
    label: 'Kullanıcı',
    description: 'Platform yetkisi yok. Yalnızca üye olduğu çalışma alanlarında, oradaki rolüyle çalışır.',
    theme: 'default',
    icon: 'user',
  },
  {
    value: 'workspace_admin',
    label: 'Çalışma alanı yöneticisi',
    description: 'Üye olmadığı alanlar dahil tüm çalışma alanlarına yönetici olarak girebilir ve yeni alan açabilir. Sistem yönetimine erişemez.',
    theme: 'primary',
    icon: 'usergroup',
  },
  {
    value: 'system_admin',
    label: 'Sistem yöneticisi',
    description: 'Bu yönetim ekranına erişir: kullanıcılar, çalışma alanları, sistem ayarları, modeller ve denetim kaydı.',
    theme: 'warning',
    icon: 'system-setting',
  },
  {
    value: 'super_admin',
    label: 'Süper yönetici',
    description: 'Sistem yöneticisi ve çalışma alanı yöneticisi yetkilerinin tamamı.',
    theme: 'danger',
    icon: 'secured',
  },
]
const SYSTEM_ROLE_BY_VALUE = Object.fromEntries(SYSTEM_ROLE_OPTIONS.map(o => [o.value, o])) as
  Record<SystemRole, (typeof SYSTEM_ROLE_OPTIONS)[number]>

function systemRoleMeta(user: SystemUser) { return SYSTEM_ROLE_BY_VALUE[systemRoleOf(user)] }

// (i) kartındaki rol → yetki matrisi; SYSTEM_ROLE_OPTIONS açıklamalarıyla uyumlu tutulmalı.
const SYSTEM_PERMS = [
  { key: 'memberWorkspaces', label: 'Üye olduğu çalışma alanlarında, oradaki rolüyle çalışma' },
  { key: 'allWorkspaces', label: 'Tüm çalışma alanlarına yönetici olarak girme ve yeni alan açma' },
  { key: 'manageUsers', label: 'Kullanıcıları ve çalışma alanlarını yönetme' },
  { key: 'systemSettings', label: 'Sistem ayarları, modeller ve denetim kaydı' },
] as const
const SYSTEM_ROLE_GRANTS: Record<SystemRole, (typeof SYSTEM_PERMS)[number]['key'][]> = {
  user: ['memberWorkspaces'],
  workspace_admin: ['memberWorkspaces', 'allWorkspaces'],
  system_admin: ['memberWorkspaces', 'manageUsers', 'systemSettings'],
  super_admin: ['memberWorkspaces', 'allWorkspaces', 'manageUsers', 'systemSettings'],
}
const SYSTEM_ROLE_MATRIX: RoleMatrixEntry[] = SYSTEM_ROLE_OPTIONS.map(o => ({
  key: o.value,
  label: o.label,
  icon: o.icon,
  perms: SYSTEM_PERMS.map(p => ({ label: p.label, has: SYSTEM_ROLE_GRANTS[o.value].includes(p.key) })),
}))

const ROLE_LABELS: Record<SystemUserMembership['role'], string> = {
  owner: 'Sahip',
  admin: 'Yönetici',
  contributor: 'Katkıda Bulunan',
  viewer: 'Görüntüleyici',
}
const ROLE_THEMES: Record<SystemUserMembership['role'], 'primary' | 'warning' | 'success' | 'default'> = {
  owner: 'warning',
  admin: 'primary',
  contributor: 'success',
  viewer: 'default',
}

// Backend guard messages are English; show the operator a Turkish version.
const ERROR_MESSAGES: Record<string, string> = {
  'Cannot delete your own account': 'Kendi hesabınızı silemezsiniz.',
  'Cannot delete the last remaining system administrator': 'Son sistem yöneticisi silinemez.',
  'Cannot revoke your own system admin privileges': 'Kendi sistem yöneticisi yetkinizi kaldıramazsınız.',
  'Cannot change your own system role': 'Kendi sistem rolünüzü değiştiremezsiniz.',
  'Cannot demote the last remaining system administrator': 'Son sistem yöneticisinin yetkisi kaldırılamaz.',
  'Invalid system role': 'Geçersiz sistem rolü.',
  'User not found': 'Kullanıcı bulunamadı.',
}

function roleLabel(role: SystemUserMembership['role']) { return ROLE_LABELS[role] ?? role }
function roleTheme(role: SystemUserMembership['role']) { return ROLE_THEMES[role] ?? 'default' }
function displayName(user: SystemUser) {
  const full = [user.first_name, user.last_name].filter(Boolean).join(' ').trim()
  return full || user.username || user.email
}
// Phones are stored in E.164 (+905321234567); show them the way people read them.
function formatPhone(value: string) { return parsePhoneNumberFromString(value)?.formatInternational() ?? value }
function formatDate(value: string) { return new Intl.DateTimeFormat('tr-TR', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)) }
function formatDay(value: string) { return new Intl.DateTimeFormat('tr-TR', { dateStyle: 'medium' }).format(new Date(value)) }
function initials(user: SystemUser) {
  const parts = displayName(user).split(/[\s@._-]+/).filter(Boolean)
  return ((parts[0]?.[0] ?? '') + (parts.length > 1 ? parts[1][0] : '')).toLocaleUpperCase('tr-TR') || '?'
}
function workspaceInitial(m: SystemUserMembership) {
  return (m.tenant_name?.trim()[0] ?? '#').toLocaleUpperCase('tr-TR')
}

// Ticks so "N dk'dır çevrimiçi" advances between list refreshes.
const now = ref(Date.now())

function formatOnlineDuration(since: string) {
  const minutes = Math.max(0, Math.floor((now.value - new Date(since).getTime()) / 60000))
  if (minutes < 1) return 'Az önce bağlandı'
  if (minutes < 60) return `${minutes} dk'dır çevrimiçi`
  const hours = Math.floor(minutes / 60)
  const rest = minutes % 60
  return rest ? `${hours} sa ${rest} dk'dır çevrimiçi` : `${hours} sa'dir çevrimiçi`
}
function presenceDetail(user: SystemUser) {
  const p = user.presence
  if (p?.online && p.online_since) return formatOnlineDuration(p.online_since)
  if (p?.last_seen_at) return `Son görülme: ${formatDate(p.last_seen_at)}`
  return 'Henüz görülmedi'
}
function presenceTitle(user: SystemUser) {
  const p = user.presence
  if (p?.online && p.online_since) return `${formatOnlineDuration(p.online_since)} (${formatDate(p.online_since)} itibarıyla)`
  return presenceDetail(user)
}
// Compact "last seen" for the profile summary strip.
function lastSeenShort(user: SystemUser) {
  const seen = user.presence?.last_seen_at
  if (!seen) return 'Görülmedi'
  const minutes = Math.max(0, Math.floor((now.value - new Date(seen).getTime()) / 60000))
  if (minutes < 60) return `${Math.max(1, minutes)} dk önce`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours} sa önce`
  const days = Math.floor(hours / 24)
  return days < 30 ? `${days} gün önce` : formatDay(seen)
}
async function copy(value: string, label: string) {
  try {
    await navigator.clipboard.writeText(value)
    MessagePlugin.success(`${label} kopyalandı.`)
  } catch {
    MessagePlugin.error('Panoya kopyalanamadı.')
  }
}
function errorMessage(err: any, fallback: string) {
  const msg = typeof err?.message === 'string' ? err.message : ''
  return ERROR_MESSAGES[msg] || msg || fallback
}

function goToPage(next: number) { if (next === page.value) return; page.value = next; void load() }

let searchTimer: ReturnType<typeof setTimeout> | undefined
watch(search, () => {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => { page.value = 1; void load() }, 300)
})
onBeforeUnmount(() => clearTimeout(searchTimer))

// silent: background presence refresh — no spinner, and a failure keeps the
// current rows instead of replacing the table with an error.
async function load(options: { silent?: boolean } = {}) {
  const current = ++requestNumber
  const silent = options.silent === true
  if (!silent) {
    loading.value = true
    error.value = ''
  }
  try {
    const res = await listSystemUsers({ offset: (page.value - 1) * PAGE_SIZE, limit: PAGE_SIZE, q: search.value.trim() || undefined })
    if (current !== requestNumber) return
    now.value = Date.now()
    items.value = res.users ?? []
    total.value = res.total ?? 0
    // Keep the open drawer in sync with the refreshed row.
    if (selected.value) {
      const fresh = items.value.find(u => u.id === selected.value!.id)
      if (fresh) selected.value = fresh
    }
    // Deleting the last row of a later page would otherwise leave an empty page.
    if (!items.value.length && page.value > 1 && total.value > 0) {
      page.value = Math.ceil(total.value / PAGE_SIZE)
      void load()
    }
  } catch (err) {
    if (current === requestNumber && !silent) error.value = errorMessage(err, 'Kullanıcılar yüklenemedi.')
  } finally {
    if (current === requestNumber) loading.value = false
  }
}

// Presence: tick the clock every 30 s for the "N dk'dır" label and re-fetch
// the page every minute so users going on/offline show up without a reload.
const PRESENCE_REFRESH_MS = 60 * 1000
let clockTimer: ReturnType<typeof setInterval> | undefined
let presenceTimer: ReturnType<typeof setInterval> | undefined
onMounted(() => {
  clockTimer = setInterval(() => { now.value = Date.now() }, 30 * 1000)
  presenceTimer = setInterval(() => {
    if (document.visibilityState !== 'visible' || loading.value || error.value) return
    void load({ silent: true })
  }, PRESENCE_REFRESH_MS)
})
onBeforeUnmount(() => {
  clearInterval(clockTimer)
  clearInterval(presenceTimer)
})

// ---------- Yönet çekmecesi ----------

const drawerVisible = ref(false)
const selected = ref<SystemUser>()
const isSelf = computed(() => !!selected.value && selected.value.id === auth.currentUserId)
const busy = computed(() => roleSubmitting.value || deleteSubmitting.value)
// Highest-privilege workspaces first, then alphabetical.
const MEMBERSHIP_ORDER: Record<SystemUserMembership['role'], number> = { owner: 0, admin: 1, contributor: 2, viewer: 3 }
const sortedMemberships = computed(() => [...(selected.value?.memberships ?? [])].sort((a, b) =>
  (MEMBERSHIP_ORDER[a.role] ?? 9) - (MEMBERSHIP_ORDER[b.role] ?? 9)
  || (a.tenant_name || '').localeCompare(b.tenant_name || '', 'tr-TR')))

function openManage(row: SystemUser) {
  selected.value = row
  resetRoleDraft()
  drawerVisible.value = true
}

function onDrawerVisible(visible: boolean) {
  // Keep the drawer open while a write is in flight so its result is visible.
  if (!visible && busy.value) return
  drawerVisible.value = visible
}

// Sistem rolü

const roleSubmitting = ref(false)
const roleDraft = ref<SystemRole>('user')
const currentRole = computed<SystemRole>(() => (selected.value ? systemRoleOf(selected.value) : 'user'))
const roleDirty = computed(() => roleDraft.value !== currentRole.value)

function resetRoleDraft() { roleDraft.value = currentRole.value }

// A background refresh can change the stored role; follow it unless the
// operator has an unsaved choice.
watch(currentRole, (next, prev) => { if (roleDraft.value === prev) roleDraft.value = next })

function confirmRoleChange() {
  const user = selected.value
  const target = roleDraft.value
  if (!user || isSelf.value || !roleDirty.value) return
  const from = SYSTEM_ROLE_BY_VALUE[currentRole.value]
  const to = SYSTEM_ROLE_BY_VALUE[target]
  // Losing system admin locks the user out of this screen; flag it red.
  const losesSystemAdmin = user.is_system_admin && (target === 'user' || target === 'workspace_admin')
  const dialog = DialogPlugin.confirm({
    theme: losesSystemAdmin ? 'danger' : 'warning',
    header: 'Sistem rolü değiştirilsin mi?',
    body: `${displayName(user)} için sistem rolü "${from.label}" → "${to.label}" olarak değiştirilecek. ${to.description}`,
    confirmBtn: { content: 'Rolü değiştir', theme: losesSystemAdmin ? 'danger' : 'primary' },
    onConfirm: async () => {
      dialog.destroy()
      roleSubmitting.value = true
      try {
        await setSystemUserRole(user.id, target)
        MessagePlugin.success(`Sistem rolü "${to.label}" olarak güncellendi.`)
        await load()
      } catch (err) {
        MessagePlugin.error(errorMessage(err, 'Sistem rolü güncellenemedi.'))
      } finally {
        roleSubmitting.value = false
      }
    },
    onCancel: () => dialog.destroy(),
  })
}

// Kullanıcıyı sil

const deleteSubmitting = ref(false)

function confirmDelete() {
  const user = selected.value
  if (!user || isSelf.value) return
  const workspaceCount = user.memberships?.length ?? 0
  const dialog = DialogPlugin.confirm({
    theme: 'danger',
    header: 'Kullanıcı silinsin mi?',
    body: `${displayName(user)} (${user.email}) kalıcı olarak silinecek`
      + (workspaceCount ? ` ve ${workspaceCount} çalışma alanından çıkarılacak.` : '.')
      + ' Bu işlem geri alınamaz.',
    confirmBtn: { content: 'Kullanıcıyı sil', theme: 'danger' },
    onConfirm: async () => {
      dialog.destroy()
      deleteSubmitting.value = true
      try {
        await deleteSystemUser(user.id)
        MessagePlugin.success('Kullanıcı silindi.')
        drawerVisible.value = false
        selected.value = undefined
        await load()
      } catch (err: any) {
        if (err?.code === 'sole_workspace_owner') showSoleOwnerDialog(user, err.workspaces ?? [])
        else MessagePlugin.error(errorMessage(err, 'Kullanıcı silinemedi.'))
      } finally {
        deleteSubmitting.value = false
      }
    },
    onCancel: () => dialog.destroy(),
  })
}

// The backend refuses to delete the only owner of a workspace; explain which
// workspaces block the delete so the operator can transfer ownership first.
function showSoleOwnerDialog(user: SystemUser, workspaces: SystemUserMembership[]) {
  const names = workspaces.map(w => w.tenant_name || `#${w.tenant_id}`)
  const dialog = DialogPlugin.alert({
    theme: 'warning',
    header: 'Kullanıcı silinemiyor',
    body: () => h('div', { class: 'users-sole-owner' }, [
      h('p', `${displayName(user)} aşağıdaki çalışma alanlarının tek sahibi. Silmeden önce bu alanlarda başka bir üyeyi sahip yapın veya çalışma alanını silin.`),
      h('ul', { style: 'margin: 8px 0 0; padding-left: 20px;' }, names.map(name => h('li', name))),
    ]),
    confirmBtn: 'Tamam',
    onConfirm: () => dialog.destroy(),
    onClose: () => dialog.destroy(),
  })
}

onMounted(() => { void load() })
</script>

<style lang="less" scoped>
@import (reference) '@/components/css/settings-section.less';

.system-users {
  width: 100%;
}

.section-header {
  .settings-section-header();
}

.users-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.users-toolbar__search {
  width: 320px;
}

.users-toolbar__meta {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  margin-left: auto;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  white-space: nowrap;
}

.users-refresh {
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

.users-refresh--spin {
  animation: wk-spin 0.8s linear infinite;
}

.users-state {
  min-height: 160px;
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

.users-table-shell {
  &--single-page :deep(.t-table__pagination) {
    display: none;
  }

  &:deep(.t-table td:last-child .t-button) {
    white-space: nowrap;
  }
}

.users-identity {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.users-identity__name,
.users-identity__email {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.users-identity__name {
  color: var(--td-text-color-primary);
  font-size: var(--app-text-md);
  font-weight: 500;
}

.users-identity__email {
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  line-height: 1.4;
}

.users-phone {
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-md);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
  text-decoration: none;

  &:hover {
    color: var(--td-brand-color);
  }
}

.users-cell-text {
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-md);
}

.users-presence {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.users-presence__state {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-md);
  white-space: nowrap;

  &--online {
    color: var(--td-success-color);
    font-weight: 500;
  }
}

.users-presence__dot {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--td-text-color-disabled);

  .users-presence__state--online & {
    background: var(--td-success-color);
    box-shadow: 0 0 0 3px var(--td-success-color-light);
  }
}

.users-presence__detail {
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  line-height: 1.4;
}

.users-muted {
  margin: 0;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  font-weight: 400;
  line-height: 1.5;
}

// ---------- Çekmece içeriği ----------

.manage-profile {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 18px;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-lg);
  background:
    linear-gradient(135deg, var(--td-brand-color-light) 0%, transparent 70%),
    var(--td-bg-color-container);
}

.manage-profile__main {
  display: flex;
  align-items: center;
  gap: 16px;
  min-width: 0;
}

.manage-avatar {
  position: relative;
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  border-radius: 50%;
  background: var(--td-brand-color);
  color: var(--td-text-color-anti);
  font-size: 20px;
  font-weight: 600;
  letter-spacing: 0.02em;
  box-shadow: 0 0 0 3px var(--td-bg-color-container);

  img {
    width: 100%;
    height: 100%;
    border-radius: 50%;
    object-fit: cover;
  }
}

.manage-avatar__presence {
  position: absolute;
  right: 1px;
  bottom: 1px;
  width: 14px;
  height: 14px;
  border: 2px solid var(--td-bg-color-container);
  border-radius: 50%;
  background: var(--td-text-color-disabled);

  &--online {
    background: var(--td-success-color);
  }
}

.manage-profile__identity {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.manage-profile__name {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  color: var(--td-text-color-primary);
  font-size: var(--app-text-lg, 16px);
  font-weight: 600;
  line-height: 1.35;
}

.manage-profile__name-text,
.manage-profile__email {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.manage-profile__self {
  flex: none;
  padding: 0 6px;
  border-radius: var(--app-radius-sm);
  background: var(--td-brand-color-light);
  color: var(--td-brand-color);
  font-size: var(--app-text-xs);
  font-weight: 500;
  line-height: 18px;
}

.manage-profile__email {
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-sm);
}

.manage-profile__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 4px;
}

.manage-stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  margin: 0;
  border-top: 1px solid var(--td-component-stroke);
  padding-top: 14px;

  > div {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
    padding: 0 12px;

    &:first-child {
      padding-left: 0;
    }

    & + div {
      border-left: 1px solid var(--td-component-stroke);
    }
  }

  dt {
    color: var(--td-text-color-placeholder);
    font-size: var(--app-text-xs);
  }

  dd {
    margin: 0;
    overflow: hidden;
    color: var(--td-text-color-primary);
    font-size: var(--app-text-md);
    font-weight: 600;
    font-variant-numeric: tabular-nums;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.manage-stats__online {
  color: var(--td-success-color) !important;
}

.manage-fields {
  display: flex;
  flex-direction: column;
  margin: 0;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-md);
  overflow: hidden;
}

.manage-field {
  display: grid;
  grid-template-columns: 150px minmax(0, 1fr);
  gap: 12px;
  align-items: center;
  min-height: 44px;
  padding: 8px 14px;

  & + & {
    border-top: 1px solid var(--td-component-stroke);
  }

  dt {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    color: var(--td-text-color-secondary);
    font-size: var(--app-text-sm);

    .t-icon {
      color: var(--td-text-color-placeholder);
      font-size: 16px;
    }
  }

  dd {
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
    margin: 0;
    color: var(--td-text-color-primary);
    font-size: var(--app-text-md);

    &:hover .manage-copy {
      opacity: 1;
    }
  }
}

.manage-field__stack {
  flex-direction: column;
  align-items: flex-start !important;
  gap: 2px !important;
}

.manage-field__value {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.manage-field__id {
  padding: 1px 6px;
  border-radius: var(--app-radius-sm);
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font-family: var(--td-font-family-mono, ui-monospace, SFMono-Regular, Menlo, monospace);
  font-size: var(--app-text-xs);
}

.manage-link {
  color: var(--td-text-color-primary);
  text-decoration: none;
  font-variant-numeric: tabular-nums;

  &:hover {
    color: var(--td-brand-color);
  }
}

.manage-copy {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  padding: 0;
  border: none;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: pointer;
  opacity: 0;
  transition: opacity var(--app-motion-base) ease, color var(--app-motion-base) ease, background var(--app-motion-base) ease;

  &:hover {
    color: var(--td-brand-color);
    background: var(--td-bg-color-secondarycontainer);
  }

  &:focus-visible {
    opacity: 1;
    outline: 2px solid var(--td-brand-color-focus);
  }
}

.manage-count {
  padding: 0 7px;
  border-radius: 9px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);
  font-weight: 500;
  line-height: 18px;
}

.manage-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 24px 16px;
  border: 1px dashed var(--td-component-stroke);
  border-radius: var(--app-radius-md);
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  text-align: center;

  .t-icon {
    font-size: 24px;
  }
}

.manage-workspaces {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-md);
  overflow: hidden;

  li {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 14px;
    transition: background var(--app-motion-base) ease;

    & + li {
      border-top: 1px solid var(--td-component-stroke);
    }

    &:hover {
      background: var(--td-bg-color-container-hover);
    }
  }
}

.manage-workspaces__badge {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: var(--app-radius-md);
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-sm);
  font-weight: 600;
}

.manage-workspaces__text {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
}

.manage-workspaces__name {
  overflow: hidden;
  color: var(--td-text-color-primary);
  font-size: var(--app-text-md);
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.manage-workspaces__id {
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-xs);
  font-variant-numeric: tabular-nums;
}

.manage-notice {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 10px 12px;
  border-radius: var(--app-radius-md);
  background: var(--td-brand-color-light);
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-sm);
  line-height: 1.5;

  .t-icon {
    flex: none;
    margin-top: 2px;
    color: var(--td-brand-color);
  }
}

.users-role-select {
  width: 100%;
}

.users-role-option {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.users-role-hint {
  margin: 6px 0 0;
  font-size: var(--app-text-sm);
  line-height: 1.5;
}

.users-role__current {
  padding: 0 6px;
  border-radius: var(--app-radius-sm);
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);
  font-weight: 400;
  line-height: 18px;
}

.manage-danger {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 16px;
  border: 1px solid var(--td-error-color-3, var(--td-error-color-light));
  border-radius: var(--app-radius-md);
  background: var(--td-error-color-light);
}

.manage-danger__icon {
  flex: none;
  display: inline-flex;
  color: var(--td-error-color);
  font-size: 20px;
}

.manage-danger__body {
  flex: 1;
  min-width: 0;
}

.manage-danger__title {
  color: var(--td-error-color);
  font-size: var(--app-text-md);
  font-weight: 600;
}

.manage-danger__text {
  margin: 2px 0 0;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-sm);
  line-height: 1.5;
}

.manage-unsaved {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-sm);
  text-overflow: ellipsis;
  white-space: nowrap;

  strong {
    color: var(--td-text-color-primary);
    font-weight: 600;
  }
}

.manage-unsaved__dot {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--td-warning-color);
}

@media (max-width: 720px) {
  .users-toolbar__search {
    width: 100%;
  }

  .users-toolbar__meta {
    margin-left: 0;
  }

  .manage-field {
    grid-template-columns: 1fr;
    gap: 4px;
  }

  .manage-danger {
    flex-wrap: wrap;
  }
}
</style>
