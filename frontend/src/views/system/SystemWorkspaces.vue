<template>
  <div class="system-workspaces">
    <header class="section-header">
      <h2>Çalışma alanları</h2>
      <p class="section-description">Sistemdeki tüm çalışma alanları, sahipleri ve üyeleri.</p>
    </header>

    <div class="ws-toolbar">
      <t-input v-model="search" clearable class="ws-toolbar__search" placeholder="Ad, açıklama veya ID ara">
        <template #prefix-icon><t-icon name="search" /></template>
      </t-input>
      <div class="ws-toolbar__meta">
        <span>{{ total }} çalışma alanı</span>
        <button type="button" class="ws-refresh" :disabled="loading" title="Yenile" aria-label="Yenile" @click="load">
          <t-icon :name="loading ? 'loading' : 'refresh'" :class="{ 'ws-refresh--spin': loading }" />
        </button>
      </div>
    </div>

    <div v-if="error" class="ws-state">
      <t-alert theme="error" :message="error">
        <template #operation>
          <t-button size="small" @click="load">Tekrar dene</t-button>
        </template>
      </t-alert>
    </div>

    <div v-else class="data-table-shell ws-table-shell"
      :class="{ 'ws-table-shell--single-page': total <= PAGE_SIZE }">
      <t-table row-key="id" :data="items" :columns="columns" :pagination="pagination" :loading="loading"
        disable-data-page size="medium" hover
        @page-change="(info: { current: number }) => goToPage(info.current)">
        <template #workspace="{ row }">
          <div class="ws-identity">
            <span class="ws-identity__name">
              {{ row.name || `#${row.id}` }}
              <t-tag v-if="row.id === auth.effectiveTenantId" size="small" variant="outline">Şu anki</t-tag>
            </span>
            <span class="ws-identity__meta">#{{ row.id }}<template v-if="row.description"> · {{ row.description }}</template></span>
          </div>
        </template>
        <template #owners="{ row }">
          <div v-if="row.owners?.length" class="ws-tags">
            <t-tag v-for="o in row.owners.slice(0, MAX_INLINE_OWNERS)" :key="o.user_id" size="small" variant="light"
              theme="warning" :title="o.email">
              {{ ownerName(o) }}
            </t-tag>
            <t-tag v-if="row.owners.length > MAX_INLINE_OWNERS" size="small" variant="outline">
              +{{ row.owners.length - MAX_INLINE_OWNERS }}
            </t-tag>
          </div>
          <span v-else class="ws-muted">Sahip yok</span>
        </template>
        <template #member_count="{ row }">
          <span class="ws-cell-text">{{ row.member_count }}</span>
        </template>
        <template #created_at="{ row }">
          <span class="ws-cell-text">{{ formatDate(row.created_at) }}</span>
        </template>
        <template #actions="{ row }">
          <t-button variant="outline" theme="primary" size="small" @click.stop="openManage(row)">
            <template #icon><t-icon name="setting" /></template>
            Yönet
          </t-button>
        </template>
        <template #empty>
          <t-empty description="Çalışma alanı bulunamadı." />
        </template>
      </t-table>
    </div>

    <SettingDrawer :visible="drawerVisible" :title="selected ? (selected.name || `#${selected.id}`) : 'Çalışma alanı'"
      :description="selected ? `#${selected.id} · ${formatDate(selected.created_at)}` : ''" icon="layers" width="640px"
      storage-key="setting-drawer:width:system-workspace-manage" hide-footer
      @update:visible="onDrawerVisible">
      <template v-if="selected">
        <section class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">Ad</h4>
          <div class="ws-inline-form">
            <t-input v-model="renameValue" :maxlength="NAME_MAX_LENGTH" placeholder="Çalışma alanı adı"
              :disabled="renameSubmitting" @enter="submitRename" />
            <t-button theme="primary" :loading="renameSubmitting" :disabled="!renameChanged" @click="submitRename">
              Kaydet
            </t-button>
          </div>
        </section>

        <section class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">
            Üye ekle
            <TenantRolePermissionsPopover placement="bottom-start" />
          </h4>
          <div class="ws-inline-form">
            <t-input v-model="addEmail" type="email" placeholder="kullanici@ornek.com" autocomplete="off"
              :disabled="addSubmitting" @enter="submitAdd" />
            <t-select v-model="addRole" :options="roleOptions" class="ws-role-select" :disabled="addSubmitting" />
            <t-button theme="primary" :loading="addSubmitting" :disabled="!addEmail.trim()" @click="submitAdd">
              <template #icon><t-icon name="add" /></template>
              Ekle
            </t-button>
          </div>
          <p class="ws-muted ws-hint">Yalnızca sisteme kayıtlı kullanıcılar eklenebilir.</p>
        </section>

        <section class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">Paylaşım daveti bağlantısı oluştur</h4>
          <p class="ws-muted ws-share-desc">
            Birden fazla kişinin kullanabileceği bir kayıt bağlantısı oluşturun. Bağlantıyı açan herkes kendi
            e-posta adresiyle kaydolup bu çalışma alanına seçilen rolle katılır. Bağlantı
            {{ INVITATION_TTL_DAYS }} gün sonra sona erer.
          </p>
          <div class="ws-inline-form">
            <t-select v-model="shareLinkRole" :options="roleOptions" class="ws-role-select"
              :disabled="shareLinkSubmitting" />
            <t-button theme="primary" variant="outline" :loading="shareLinkSubmitting" @click="submitShareLink">
              <template #icon><t-icon name="link" /></template>
              Bağlantı oluştur
            </t-button>
          </div>
          <div v-if="shareLinkURL" class="ws-inline-form ws-share-result">
            <t-input :model-value="shareLinkURL" readonly />
            <t-button theme="primary" @click="copyShareLink">
              <template #icon><t-icon name="copy" /></template>
              Kopyala
            </t-button>
          </div>
          <p v-if="shareLinkURL" class="ws-muted ws-hint">
            Bağlantı aşağıdaki bekleyen davetiyeler listesinde de görünür; oradan yeniden kopyalayabilir veya
            iptal edebilirsiniz.
          </p>
        </section>

        <section class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">Bekleyen davetiyeler ({{ invitationsTotal }})</h4>
          <p class="ws-muted ws-share-desc">
            Davet edilenin kabul etmesi bekleniyor. {{ INVITATION_TTL_DAYS }} gün sonra otomatik olarak sona erer.
          </p>
          <t-alert v-if="invitationsError" theme="error" :message="invitationsError">
            <template #operation>
              <t-button size="small" @click="loadInvitations">Tekrar dene</t-button>
            </template>
          </t-alert>
          <div v-else-if="invitationsLoading && !invitations.length" class="ws-members-loading">
            <t-loading size="small" />
          </div>
          <p v-else-if="!invitationsTotal" class="ws-muted">Bekleyen davetiye yok.</p>
          <template v-else>
            <t-table row-key="id" :data="invitations" :columns="invitationColumns" :loading="invitationsLoading"
              size="small" hover bordered class="ws-invitations">
              <template #invitee="{ row }">
                <div class="ws-identity">
                  <template v-if="row.is_share_link">
                    <span class="ws-identity__name">
                      <t-icon name="link" class="ws-link-icon" />
                      Bağlantı yoluyla davet
                    </span>
                    <span class="ws-identity__meta">
                      {{ (row.accepted_count ?? 0) > 0 ? `${row.accepted_count} katıldı` : 'Henüz kimse katılmadı' }}
                    </span>
                  </template>
                  <template v-else>
                    <span class="ws-identity__name" :title="inviteePrimary(row)">{{ inviteePrimary(row) }}</span>
                    <span v-if="row.invitee_email && row.invitee_name" class="ws-identity__meta">
                      {{ row.invitee_email }}
                    </span>
                  </template>
                </div>
              </template>
              <template #role="{ row }">{{ ROLE_LABELS[row.role as TenantRole] || row.role }}</template>
              <template #inviter="{ row }">
                <span class="ws-ellipsis" :title="inviterPrimary(row)">{{ inviterPrimary(row) }}</span>
              </template>
              <template #expires_at="{ row }">{{ formatDate(row.expires_at) }}</template>
              <template #status="{ row }">
                <t-tag size="small" variant="light" :theme="INVITATION_STATUS_THEMES[row.status] || 'default'">
                  {{ row.is_share_link && row.status === 'pending'
                    ? 'Aktif' : (INVITATION_STATUS_LABELS[row.status] || row.status) }}
                </t-tag>
              </template>
              <template #actions="{ row }">
                <div class="ws-row-actions">
                  <t-tooltip v-if="row.status === 'pending' && row.invite_url" content="Davet bağlantısını kopyala">
                    <t-button shape="square" variant="text" size="small"
                      @click="copyText(absoluteInviteURL(row.invite_url))">
                      <template #icon><t-icon name="copy" /></template>
                    </t-button>
                  </t-tooltip>
                  <t-popconfirm v-if="row.status === 'pending'" theme="warning" placement="left"
                    :content="row.is_share_link
                      ? 'İptal işlemi, henüz kaydolmamış kişilerin bu bağlantıyı kullanmasını engelleyecektir. Yeniden paylaşmak için yeni bir tane oluşturun.'
                      : `İptal ettikten sonra ${row.invitee_email || row.invitee_user_id} artık bu daveti kabul edemez.`"
                    :confirm-btn="{ content: 'İptal et', theme: 'danger' }" cancel-btn="Vazgeç"
                    @confirm="revokeInvitationRow(row)">
                    <t-tooltip content="İptal et">
                      <t-button shape="square" variant="text" size="small" :loading="revokingId === row.id"
                        :disabled="revokingId !== null">
                        <template #icon><t-icon name="close" /></template>
                      </t-button>
                    </t-tooltip>
                  </t-popconfirm>
                </div>
              </template>
            </t-table>
            <t-pagination v-if="invitationsTotal > invitationsPageSize" v-model="invitationsPage"
              v-model:page-size="invitationsPageSize" :total="invitationsTotal" size="small"
              :page-size-options="INVITATIONS_PAGE_SIZE_OPTIONS" class="ws-invitations-pager"
              @change="loadInvitations" />
          </template>
        </section>

        <section class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">Üyeler ({{ members.length }})</h4>
          <t-alert v-if="membersError" theme="error" :message="membersError">
            <template #operation>
              <t-button size="small" @click="loadMembers">Tekrar dene</t-button>
            </template>
          </t-alert>
          <div v-else-if="membersLoading && !members.length" class="ws-members-loading">
            <t-loading size="small" />
          </div>
          <p v-else-if="!members.length" class="ws-muted">Bu çalışma alanında üye yok.</p>
          <ul v-else class="ws-members">
            <li v-for="m in members" :key="m.user_id">
              <div class="ws-identity">
                <span class="ws-identity__name">
                  {{ m.username || m.email || m.user_id }}
                  <span v-if="m.user_id === auth.currentUserId" class="ws-muted">(siz)</span>
                </span>
                <span class="ws-identity__meta">{{ m.email }}</span>
              </div>
              <div class="ws-members__actions">
                <t-select :model-value="m.role" :options="roleOptions" size="small" class="ws-role-select"
                  :disabled="memberBusy !== null" :loading="memberBusy === m.user_id"
                  @change="(v: unknown) => changeRole(m, v as TenantRole)" />
                <t-button variant="text" theme="danger" size="small"
                  :disabled="memberBusy !== null || m.user_id === auth.currentUserId"
                  :title="m.user_id === auth.currentUserId ? 'Kendinizi buradan çıkaramazsınız' : 'Çalışma alanından çıkar'"
                  @click="confirmRemove(m)">
                  Çıkar
                </t-button>
              </div>
            </li>
          </ul>
        </section>

        <section class="setting-drawer__section">
          <div class="ws-danger">
            <span class="ws-danger__icon" aria-hidden="true"><t-icon name="error-circle" /></span>
            <div class="ws-danger__body">
              <div class="ws-danger__title">Çalışma alanını sil</div>
              <p class="ws-danger__text">
                Çalışma alanı ve tüm üyelikleri silinir; üyeler bu alana artık erişemez.
                Bu işlem geri alınamaz.
              </p>
            </div>
            <t-button theme="danger" variant="outline" :loading="deleteSubmitting" @click="confirmDelete">
              <template #icon><DeleteIcon /></template>
              Sil
            </t-button>
          </div>
        </section>
      </template>
    </SettingDrawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next'
import { DeleteIcon } from 'tdesign-icons-vue-next'
import type { TableProps } from 'tdesign-vue-next'
import SettingDrawer from '@/components/settings/SettingDrawer.vue'
import TenantRolePermissionsPopover from '@/components/settings/TenantRolePermissionsPopover.vue'
import { useAuthStore } from '@/stores/auth'
import { useHomeTenant } from '@/composables/useRoleLabel'
import { navigateAfterTenantSwitch, persistLastActiveTenantPreference } from '@/utils/tenantSwitch'
import { copyToClipboard } from '@/utils/clipboard'
import type { TenantMember, TenantRole } from '@/api/tenant/members'
import type { TenantInvitation } from '@/api/tenant/invitations'
import {
  addSystemWorkspaceMember,
  createSystemWorkspaceInviteLink,
  deleteSystemWorkspace,
  listSystemWorkspaceInvitations,
  listSystemWorkspaceMembers,
  listSystemWorkspaces,
  removeSystemWorkspaceMember,
  revokeSystemWorkspaceInvitation,
  updateSystemWorkspace,
  updateSystemWorkspaceMemberRole,
  type SystemWorkspace,
  type SystemWorkspaceOwner,
} from '@/api/system'

const PAGE_SIZE = 20
const MAX_INLINE_OWNERS = 3
const NAME_MAX_LENGTH = 128
const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
// Mirrors defaultInvitationTTL in the backend (same constant as TenantMembers.vue).
const INVITATION_TTL_DAYS = 7

const auth = useAuthStore()
const { homeTenantId } = useHomeTenant()

const search = ref('')
const items = ref<SystemWorkspace[]>([])
const page = ref(1)
const total = ref(0)
const loading = ref(false)
const error = ref('')
let requestNumber = 0

const pagination = computed(() => ({ current: page.value, pageSize: PAGE_SIZE, total: total.value, showPageSize: false }))
const columns: TableProps['columns'] = [
  // Proportional widths so no single column swallows the leftover space;
  // the actions column stays fixed to fit its button.
  { colKey: 'workspace', title: 'Çalışma alanı', width: '34%' },
  { colKey: 'owners', title: 'Sahipler', width: '26%' },
  { colKey: 'member_count', title: 'Üye sayısı', width: '12%' },
  { colKey: 'created_at', title: 'Oluşturulma tarihi', width: '18%' },
  { colKey: 'actions', title: '', width: 128, align: 'right' },
]

const ROLE_LABELS: Record<TenantRole, string> = {
  owner: 'Sahip',
  admin: 'Yönetici',
  contributor: 'Katkıda Bulunan',
  viewer: 'Görüntüleyici',
}
const roleOptions = (Object.keys(ROLE_LABELS) as TenantRole[]).map(value => ({ value, label: ROLE_LABELS[value] }))

// Backend guard messages are English; show the operator a Turkish version.
const ERROR_MESSAGES: Record<string, string> = {
  'Workspace not found': 'Çalışma alanı bulunamadı.',
  'Workspace name cannot be blank': 'Çalışma alanı adı boş olamaz.',
  'Workspace name is too long': `Çalışma alanı adı en fazla ${NAME_MAX_LENGTH} karakter olabilir.`,
  'No registered user with this email': 'Bu e-posta ile kayıtlı kullanıcı yok. Kullanıcının önce kayıt olması gerekir.',
  'User is already a member of this workspace': 'Kullanıcı zaten bu çalışma alanının üyesi.',
  'Cannot demote or remove the last owner of the workspace': 'Çalışma alanının son sahibinin rolü değiştirilemez veya çıkarılamaz. Önce başka bir üyeyi sahip yapın.',
  'Membership not found': 'Üyelik bulunamadı.',
  'Invalid workspace role': 'Geçersiz rol.',
  'Cannot remove yourself from a workspace here': 'Kendinizi buradan çıkaramazsınız.',
  'Invalid request body': 'Geçersiz istek.',
  'Invitation not found': 'Davet bulunamadı.',
  'Invitation is no longer pending': 'Bu davet artık beklemede değil.',
}

function ownerName(o: SystemWorkspaceOwner) { return o.username || o.email || o.user_id }
function formatDate(value: string) { return new Intl.DateTimeFormat('tr-TR', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)) }
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

async function load() {
  const current = ++requestNumber
  loading.value = true
  error.value = ''
  try {
    const res = await listSystemWorkspaces({ page: page.value, page_size: PAGE_SIZE, q: search.value.trim() || undefined })
    if (current !== requestNumber) return
    items.value = res.workspaces ?? []
    total.value = res.total ?? 0
    // Keep the open drawer in sync with the refreshed row.
    if (selected.value) {
      const fresh = items.value.find(w => w.id === selected.value!.id)
      if (fresh) selected.value = fresh
    }
    // Deleting the last row of a later page would otherwise leave an empty page.
    if (!items.value.length && page.value > 1 && total.value > 0) {
      page.value = Math.ceil(total.value / PAGE_SIZE)
      void load()
    }
  } catch (err) {
    if (current === requestNumber) error.value = errorMessage(err, 'Çalışma alanları yüklenemedi.')
  } finally {
    if (current === requestNumber) loading.value = false
  }
}

// ---------- Yönet çekmecesi ----------

const drawerVisible = ref(false)
const selected = ref<SystemWorkspace>()
const busy = computed(() =>
  renameSubmitting.value || addSubmitting.value || deleteSubmitting.value || shareLinkSubmitting.value
  || memberBusy.value !== null || revokingId.value !== null)

function openManage(row: SystemWorkspace) {
  selected.value = row
  renameValue.value = row.name
  addEmail.value = ''
  addRole.value = 'contributor'
  shareLinkRole.value = 'contributor'
  shareLinkURL.value = ''
  members.value = []
  invitations.value = []
  invitationsTotal.value = 0
  invitationsPage.value = 1
  invitationsError.value = ''
  drawerVisible.value = true
  void loadMembers()
  void loadInvitations()
}

function onDrawerVisible(visible: boolean) {
  // Keep the drawer open while a write is in flight so its result is visible.
  if (!visible && busy.value) return
  drawerVisible.value = visible
}

// Üyeler

const members = ref<TenantMember[]>([])
const membersLoading = ref(false)
const membersError = ref('')
const memberBusy = ref<string | null>(null)
let membersRequestNumber = 0

async function loadMembers() {
  const ws = selected.value
  if (!ws) return
  const current = ++membersRequestNumber
  membersLoading.value = true
  membersError.value = ''
  try {
    const res = await listSystemWorkspaceMembers(ws.id)
    if (current !== membersRequestNumber) return
    members.value = sortMembers(res.members ?? [])
  } catch (err) {
    if (current === membersRequestNumber) membersError.value = errorMessage(err, 'Üyeler yüklenemedi.')
  } finally {
    if (current === membersRequestNumber) membersLoading.value = false
  }
}

const ROLE_ORDER: Record<TenantRole, number> = { owner: 0, admin: 1, contributor: 2, viewer: 3 }
function sortMembers(list: TenantMember[]) {
  return [...list].sort((a, b) =>
    (ROLE_ORDER[a.role] ?? 9) - (ROLE_ORDER[b.role] ?? 9)
    || (a.username || a.email).localeCompare(b.username || b.email, 'tr'))
}

// Membership edits change owners / member count in the table and, when they
// touch the operator's own active workspace, their role-gated UI.
async function afterMembershipChange(userId?: string) {
  await Promise.all([loadMembers(), load()])
  if (userId && userId === auth.currentUserId && selected.value?.id === auth.effectiveTenantId) {
    await auth.refreshFromAuthMe()
  }
}

async function changeRole(member: TenantMember, role: TenantRole) {
  const ws = selected.value
  if (!ws || role === member.role || memberBusy.value) return
  memberBusy.value = member.user_id
  try {
    await updateSystemWorkspaceMemberRole(ws.id, member.user_id, role)
    MessagePlugin.success(`Rol "${ROLE_LABELS[role]}" olarak güncellendi.`)
    await afterMembershipChange(member.user_id)
  } catch (err) {
    MessagePlugin.error(errorMessage(err, 'Rol güncellenemedi.'))
  } finally {
    memberBusy.value = null
  }
}

function confirmRemove(member: TenantMember) {
  const ws = selected.value
  if (!ws || memberBusy.value || member.user_id === auth.currentUserId) return
  const name = member.username || member.email
  const dialog = DialogPlugin.confirm({
    theme: 'danger',
    header: 'Üye çıkarılsın mı?',
    body: `${name}, "${ws.name}" çalışma alanından çıkarılacak ve açık oturumları sonlandırılacak.`,
    confirmBtn: { content: 'Çıkar', theme: 'danger' },
    onConfirm: async () => {
      dialog.destroy()
      memberBusy.value = member.user_id
      try {
        await removeSystemWorkspaceMember(ws.id, member.user_id)
        MessagePlugin.success('Üye çalışma alanından çıkarıldı.')
        await afterMembershipChange()
      } catch (err) {
        MessagePlugin.error(errorMessage(err, 'Üye çıkarılamadı.'))
      } finally {
        memberBusy.value = null
      }
    },
    onCancel: () => dialog.destroy(),
  })
}

// Üye ekle

const addEmail = ref('')
const addRole = ref<TenantRole>('contributor')
const addSubmitting = ref(false)

async function submitAdd() {
  const ws = selected.value
  const email = addEmail.value.trim()
  if (!ws || !email || addSubmitting.value) return
  if (!EMAIL_PATTERN.test(email)) {
    MessagePlugin.warning('Geçerli bir e-posta adresi girin.')
    return
  }
  addSubmitting.value = true
  try {
    const member = await addSystemWorkspaceMember(ws.id, email, addRole.value)
    MessagePlugin.success(`${member.username || member.email} çalışma alanına eklendi.`)
    addEmail.value = ''
    addRole.value = 'contributor'
    await afterMembershipChange(member.user_id)
  } catch (err) {
    MessagePlugin.error(errorMessage(err, 'Üye eklenemedi.'))
  } finally {
    addSubmitting.value = false
  }
}

// Paylaşım daveti bağlantısı

const shareLinkRole = ref<TenantRole>('contributor')
const shareLinkSubmitting = ref(false)
const shareLinkURL = ref('')

// The backend returns "/register?token=…" when FRONTEND_BASE_URL is unset;
// resolve it against the SPA origin so the copied link works as-is.
function absoluteInviteURL(raw: string) {
  if (!raw || /^https?:\/\//i.test(raw)) return raw
  return window.location.origin + (raw.startsWith('/') ? raw : `/${raw}`)
}

async function submitShareLink() {
  const ws = selected.value
  if (!ws || shareLinkSubmitting.value) return
  shareLinkSubmitting.value = true
  try {
    const inv = await createSystemWorkspaceInviteLink(ws.id, shareLinkRole.value)
    shareLinkURL.value = absoluteInviteURL(inv.invite_url || '')
    if (!shareLinkURL.value) {
      MessagePlugin.error('Davet bağlantısı oluşturulamadı.')
      return
    }
    MessagePlugin.success(`"${ROLE_LABELS[shareLinkRole.value]}" rolü için davet bağlantısı oluşturuldu.`)
    invitationsPage.value = 1
    void loadInvitations()
  } catch (err) {
    MessagePlugin.error(errorMessage(err, 'Davet bağlantısı oluşturulamadı.'))
  } finally {
    shareLinkSubmitting.value = false
  }
}

function copyShareLink() {
  return copyText(shareLinkURL.value)
}

async function copyText(value: string) {
  if (await copyToClipboard(value)) MessagePlugin.success('Bağlantı panoya kopyalandı.')
  else MessagePlugin.error('Kopyalanamadı; bağlantıyı elle seçip kopyalayın.')
}

// Bekleyen davetiyeler

const INVITATIONS_PAGE_SIZE_OPTIONS = [10, 20, 50, 100]
const INVITATION_STATUS_LABELS: Record<string, string> = {
  pending: 'Beklemede',
  accepted: 'Kabul edildi',
  declined: 'Reddedildi',
  revoked: 'İptal edildi',
  expired: 'Süresi dolmuş',
}
const INVITATION_STATUS_THEMES: Record<string, 'success' | 'primary' | 'warning' | 'danger' | 'default'> = {
  pending: 'success',
  accepted: 'primary',
  declined: 'warning',
  revoked: 'warning',
  expired: 'danger',
}
const invitationColumns: TableProps['columns'] = [
  { colKey: 'invitee', title: 'Davetli', minWidth: 160, ellipsis: true },
  { colKey: 'role', title: 'Rol', width: 120 },
  { colKey: 'inviter', title: 'Davet eden', minWidth: 110, ellipsis: true },
  { colKey: 'expires_at', title: 'Süresi doluyor', width: 150 },
  { colKey: 'status', title: 'Durum', width: 100 },
  { colKey: 'actions', title: '', width: 80, align: 'center' },
]

const invitations = ref<TenantInvitation[]>([])
const invitationsTotal = ref(0)
const invitationsPage = ref(1)
const invitationsPageSize = ref(10)
const invitationsLoading = ref(false)
const invitationsError = ref('')
const revokingId = ref<number | null>(null)
let invitationsRequestNumber = 0

function inviteePrimary(row: TenantInvitation) {
  return row.invitee_name?.trim() || row.invitee_email?.trim() || row.invitee_user_id
}

function inviterPrimary(row: TenantInvitation) {
  return row.inviter_name?.trim() || row.inviter_email?.trim() || row.invited_by || '—'
}

async function loadInvitations() {
  const ws = selected.value
  if (!ws) return
  const current = ++invitationsRequestNumber
  invitationsLoading.value = true
  invitationsError.value = ''
  try {
    const res = await listSystemWorkspaceInvitations(ws.id, {
      page: invitationsPage.value,
      page_size: invitationsPageSize.value,
    })
    if (current !== invitationsRequestNumber) return
    const count = res.total ?? 0
    // Revoking the last row of a later page would otherwise leave an empty page.
    const maxPage = Math.max(1, Math.ceil(count / invitationsPageSize.value))
    if (invitationsPage.value > maxPage) {
      invitationsPage.value = maxPage
      return loadInvitations()
    }
    invitations.value = res.invitations ?? []
    invitationsTotal.value = count
  } catch (err) {
    if (current === invitationsRequestNumber) invitationsError.value = errorMessage(err, 'Davetiyeler yüklenemedi.')
  } finally {
    if (current === invitationsRequestNumber) invitationsLoading.value = false
  }
}

async function revokeInvitationRow(row: TenantInvitation) {
  const ws = selected.value
  if (!ws || revokingId.value !== null) return
  revokingId.value = row.id
  try {
    await revokeSystemWorkspaceInvitation(ws.id, row.id)
    MessagePlugin.success('Davet iptal edildi.')
    if (shareLinkURL.value && row.invite_url && absoluteInviteURL(row.invite_url) === shareLinkURL.value) {
      shareLinkURL.value = ''
    }
    await loadInvitations()
  } catch (err) {
    MessagePlugin.error(errorMessage(err, 'Davet iptal edilemedi.'))
    void loadInvitations()
  } finally {
    revokingId.value = null
  }
}

// Yeniden adlandır

const renameValue = ref('')
const renameSubmitting = ref(false)
const renameChanged = computed(() => {
  const next = renameValue.value.trim()
  return !!next && next !== selected.value?.name
})

async function submitRename() {
  const ws = selected.value
  if (!ws || !renameChanged.value || renameSubmitting.value) return
  const name = renameValue.value.trim()
  renameSubmitting.value = true
  try {
    await updateSystemWorkspace(ws.id, name)
    MessagePlugin.success('Çalışma alanı adı güncellendi.')
    syncAuthWorkspaceName(ws.id, name)
    selected.value = { ...ws, name }
    renameValue.value = name
    await load()
  } catch (err) {
    MessagePlugin.error(errorMessage(err, 'Çalışma alanı adı güncellenemedi.'))
  } finally {
    renameSubmitting.value = false
  }
}

// The workspace switcher reads names from the auth store; mirror the rename
// there so the operator doesn't see the old name until the next /auth/me.
function syncAuthWorkspaceName(id: number, name: string) {
  if (auth.tenant && Number(auth.tenant.id) === id) auth.setTenant({ ...auth.tenant, name })
  if (auth.memberships?.some(m => m.tenant_id === id)) {
    auth.setMemberships(auth.memberships.map(m => (m.tenant_id === id ? { ...m, tenant_name: name } : m)))
  }
  // Same id → no tenant switch side effects, only the cached name changes.
  if (auth.selectedTenantId === id) auth.setSelectedTenant(id, name)
}

// Çalışma alanını sil

const deleteSubmitting = ref(false)

function confirmDelete() {
  const ws = selected.value
  if (!ws) return
  const isActive = ws.id === auth.effectiveTenantId
  const dialog = DialogPlugin.confirm({
    theme: 'danger',
    header: 'Çalışma alanı silinsin mi?',
    body: `"${ws.name || `#${ws.id}`}" çalışma alanı`
      + (ws.member_count ? ` ve ${ws.member_count} üyeliği` : '')
      + ' kalıcı olarak silinecek. Bu işlem geri alınamaz.'
      + (isActive ? ' Şu anda bu çalışma alanındasınız; silindikten sonra başka bir alana geçirileceksiniz.' : ''),
    confirmBtn: { content: 'Çalışma alanını sil', theme: 'danger' },
    onConfirm: async () => {
      dialog.destroy()
      deleteSubmitting.value = true
      try {
        await deleteSystemWorkspace(ws.id)
        MessagePlugin.success('Çalışma alanı silindi.')
        drawerVisible.value = false
        selected.value = undefined
        if (isActive) {
          await leaveDeletedWorkspace(ws.id)
          return
        }
        if (auth.memberships?.some(m => m.tenant_id === ws.id)) {
          auth.setMemberships(auth.memberships.filter(m => m.tenant_id !== ws.id))
        }
        await load()
      } catch (err) {
        MessagePlugin.error(errorMessage(err, 'Çalışma alanı silinemedi.'))
      } finally {
        deleteSubmitting.value = false
      }
    },
    onCancel: () => dialog.destroy(),
  })
}

// Every request carries X-Tenant-ID of the active workspace, so after
// deleting it the operator must be moved elsewhere (same flow as deleting a
// workspace from its own settings page).
async function leaveDeletedWorkspace(id: number) {
  auth.setMemberships((auth.memberships ?? []).filter(m => m.tenant_id !== id))
  // Point X-Tenant-ID away from the deleted workspace before calling
  // /auth/me, otherwise that request itself is rejected.
  const fallback = auth.memberships.find(m => m.tenant_id === homeTenantId.value) ?? auth.memberships[0]
  if (fallback) auth.setSelectedTenant(fallback.tenant_id, fallback.tenant_name?.trim() || `#${fallback.tenant_id}`)
  else auth.setSelectedTenant(null)
  try {
    await auth.refreshFromAuthMe()
  } catch {
    // Fall through to the membership check below.
  }
  const next = auth.memberships.find(m => m.tenant_id === homeTenantId.value) ?? auth.memberships[0]
  if (!next) {
    auth.logout()
    window.location.href = '/login'
    return
  }
  auth.setSelectedTenant(next.tenant_id, next.tenant_name?.trim() || `#${next.tenant_id}`)
  const persist = persistLastActiveTenantPreference(homeTenantId.value === next.tenant_id ? null : next.tenant_id)
  await Promise.race([persist, new Promise(r => setTimeout(r, 400))])
  navigateAfterTenantSwitch()
}

onMounted(() => { void load() })
</script>

<style lang="less" scoped>
@import (reference) '@/components/css/settings-section.less';

.system-workspaces {
  width: 100%;
}

.section-header {
  .settings-section-header();
}

.ws-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.ws-toolbar__search {
  width: 320px;
}

.ws-toolbar__meta {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  margin-left: auto;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  white-space: nowrap;
}

.ws-refresh {
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

.ws-refresh--spin {
  animation: wk-spin 0.8s linear infinite;
}

.ws-state {
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

.ws-table-shell {
  &--single-page :deep(.t-table__pagination) {
    display: none;
  }

  &:deep(.t-table td:last-child .t-button) {
    white-space: nowrap;
  }
}

.ws-identity {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.ws-identity__name {
  display: inline-flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  color: var(--td-text-color-primary);
  font-size: var(--app-text-md);
  font-weight: 500;
  overflow-wrap: anywhere;
}

.ws-identity__meta {
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  line-height: 1.4;
  overflow-wrap: anywhere;
}

.ws-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;

  :deep(.t-tag) {
    max-width: 220px;
  }
}

.ws-cell-text {
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-md);
}

.ws-muted {
  margin: 0;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  font-weight: 400;
  line-height: 1.5;
}

// ---------- Çekmece içeriği ----------

.ws-inline-form {
  display: flex;
  align-items: center;
  gap: 8px;

  > .t-input__wrap,
  > :deep(.t-input__wrap) {
    flex: 1;
    min-width: 0;
  }
}

.ws-role-select {
  width: 160px;
  flex-shrink: 0;
}

.ws-hint {
  margin-top: 8px;
}

.ws-share-desc {
  margin-bottom: 10px;
}

.ws-share-result {
  margin-top: 10px;
}

.ws-link-icon {
  color: var(--td-brand-color);
}

.ws-ellipsis {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ws-row-actions {
  display: inline-flex;
  align-items: center;
  gap: 2px;
}

.ws-invitations-pager {
  margin-top: 10px;
}

.ws-members-loading {
  display: flex;
  justify-content: center;
  padding: 16px 0;
}

.ws-members {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-md);

  li {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 10px 14px;

    & + li {
      border-top: 1px solid var(--td-component-stroke);
    }
  }
}

.ws-members__actions {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.ws-danger {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 16px;
  border: 1px solid var(--td-error-color-3);
  border-radius: var(--app-radius-md);
  background: var(--td-error-color-light);
}

.ws-danger__icon {
  flex: none;
  display: inline-flex;
  color: var(--td-error-color);
  font-size: var(--app-text-3xl);
}

.ws-danger__body {
  flex: 1;
  min-width: 0;
}

.ws-danger__title {
  color: var(--td-error-color);
  font-size: var(--app-text-md);
  font-weight: 600;
}

.ws-danger__text {
  margin: 2px 0 0;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-sm);
  line-height: 1.5;
}

@media (max-width: 720px) {
  .ws-toolbar__search {
    width: 100%;
  }

  .ws-toolbar__meta {
    margin-left: 0;
  }

  .ws-inline-form,
  .ws-members li {
    flex-direction: column;
    align-items: stretch;
  }

  .ws-danger {
    flex-wrap: wrap;
  }
}
</style>
