<template>
  <div class="tenant-info">
    <div class="section-header">
      <h2>{{ $t('tenant.title') }}</h2>
      <p class="section-description">{{ $t('tenant.sectionDescription') }}</p>
    </div>

    <!-- Loading state -->
    <div v-if="loading" class="shell" aria-busy="true" :aria-label="$t('tenant.loadingInfo')">
      <div class="core hero hero--skeleton">
        <t-skeleton animation="gradient" :row-col="[{ type: 'rect', size: '72px' }]" />
        <div class="skeleton-lines">
          <t-skeleton animation="gradient" :row-col="[{ width: '40%', height: '22px' }, { width: '65%' }, { width: '50%' }]" />
        </div>
      </div>
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="error-inline">
      <t-alert theme="error" :message="error">
        <template #operation>
          <t-button size="small" @click="loadInfo">{{ $t('tenant.retry') }}</t-button>
        </template>
      </t-alert>
    </div>

    <template v-else>
      <!-- Kimlik kartı: ad + açıklama yerinde düzenlenir, durum / rol / tarih meta çiplerinde özetlenir -->
      <section class="shell" style="--i: 0" aria-labelledby="tenant-identity-name">
        <div class="core hero">
          <div class="ws-mark" aria-hidden="true">
            <span class="ws-mark-initials">{{ initials }}</span>
          </div>

          <div class="hero-text">
            <!-- Salt okunur durum: adı + düzenleme düğmesini gösterir (düzenleme girişi yalnızca owner tarafından görülebilir).
               Açılır pencere yerine yerinde düzenleme: bir kat daha az görsel kesinti sağlar. -->
            <div v-if="!editing" class="hero-name-row">
              <h3 id="tenant-identity-name" class="hero-name">{{ tenantInfo?.name || '-' }}</h3>
              <t-tooltip v-if="canEditTenant" :content="$t('tenant.details.editName')">
                <button type="button" class="icon-btn" :aria-label="$t('tenant.details.editName')"
                  @click="startEditName">
                  <EditIcon size="16px" aria-hidden="true" />
                </button>
              </t-tooltip>
            </div>
            <!-- Düzenleme durumu: girdi alanı + kaydet/iptal. Enter kaydeder, Esc iptal eder.-->
            <div v-else class="inline-edit">
              <t-input v-model="editName" :placeholder="$t('tenant.details.editNamePlaceholder')" :maxlength="64"
                :disabled="saving" autofocus class="inline-edit-input" @enter="saveTenantName"
                @keydown="onEditKeydown" />
              <div class="inline-edit-actions">
                <t-button theme="default" variant="outline" size="small" :disabled="saving" @click="cancelEditName">
                  {{ $t('tenant.details.editNameCancel') }}
                </t-button>
                <t-button theme="primary" size="small" :loading="saving" :disabled="!canSubmit"
                  @click="saveTenantName">
                  {{ $t('tenant.details.editNameConfirm') }}
                </t-button>
              </div>
            </div>

            <!-- Açıklama: adla aynı "yerinde düzenleme" modu. Esc iptal eder, Ctrl/⌘+Enter kaydeder;
               textarea üzerinde Enter'ın varsayılan olarak yeni satır eklemesi daha kullanışlıdır. -->
            <div v-if="!editingDescription" class="hero-desc-row">
              <p class="hero-desc" :class="{ 'is-empty': !tenantInfo?.description }">
                {{ tenantInfo?.description || $t('tenant.overview.noDescription') }}
              </p>
              <t-tooltip v-if="canEditTenant" :content="$t('tenant.details.editDescription')">
                <button type="button" class="icon-btn icon-btn--sm"
                  :aria-label="$t('tenant.details.editDescription')" @click="startEditDescription">
                  <EditIcon size="14px" aria-hidden="true" />
                </button>
              </t-tooltip>
            </div>
            <div v-else class="inline-edit inline-edit--stacked">
              <t-textarea v-model="editDescription" :placeholder="$t('tenant.details.editDescriptionPlaceholder')"
                :maxlength="512" :autosize="{ minRows: 2, maxRows: 6 }" :disabled="savingDescription" autofocus
                class="inline-edit-textarea" @keydown="onEditDescriptionKeydown" />
              <div class="inline-edit-actions">
                <t-button theme="default" variant="outline" size="small" :disabled="savingDescription"
                  @click="cancelEditDescription">
                  {{ $t('tenant.details.editNameCancel') }}
                </t-button>
                <t-button theme="primary" size="small" :loading="savingDescription"
                  :disabled="!canSubmitDescription" @click="saveTenantDescription">
                  {{ $t('tenant.details.editNameConfirm') }}
                </t-button>
              </div>
            </div>

            <ul class="hero-meta">
              <li class="meta-chip" :class="`meta-chip--${statusTone}`">
                <span class="status-dot" aria-hidden="true" />
                <span>{{ getStatusText(tenantInfo?.status) }}</span>
              </li>
              <li v-if="roleLabel" class="meta-chip">
                <t-icon :name="roleIcon(currentTenantRole) || 'user'" aria-hidden="true" />
                <span>{{ $t('tenant.overview.yourRole', { role: roleLabel }) }}</span>
              </li>
              <li v-if="tenantInfo?.created_at" class="meta-chip">
                <t-icon name="calendar" aria-hidden="true" />
                <span>{{ $t('tenant.overview.createdOn', { date: formatDay(tenantInfo.created_at) }) }}</span>
              </li>
            </ul>
          </div>
        </div>
      </section>

      <div class="bento" :class="{ 'bento--single': !hasStorage }">
        <!-- Ayrıntılar -->
        <section class="shell" style="--i: 1" aria-labelledby="tenant-details-title">
          <div class="core panel">
            <header class="panel-head">
              <span class="panel-icon" aria-hidden="true"><t-icon name="info-circle" /></span>
              <div class="panel-titles">
                <h3 id="tenant-details-title">{{ $t('tenant.overview.detailsTitle') }}</h3>
                <p>{{ $t('tenant.overview.detailsHint') }}</p>
              </div>
            </header>
            <dl class="detail-list">
              <div class="detail-row">
                <dt>{{ $t('tenant.details.idLabel') }}</dt>
                <dd class="detail-id">
                  <code class="mono">{{ tenantInfo?.id || '-' }}</code>
                  <t-tooltip v-if="tenantInfo?.id" :content="$t('tenant.overview.copyId')">
                    <button type="button" class="icon-btn icon-btn--sm" :aria-label="$t('tenant.overview.copyId')"
                      @click="copyTenantId">
                      <t-icon name="copy" />
                    </button>
                  </t-tooltip>
                </dd>
              </div>
              <div v-if="tenantInfo?.business" class="detail-row">
                <dt>{{ $t('tenant.details.businessLabel') }}</dt>
                <dd>{{ tenantInfo.business }}</dd>
              </div>
              <div class="detail-row">
                <dt>{{ $t('tenant.details.createdAtLabel') }}</dt>
                <dd>{{ formatDate(tenantInfo?.created_at) }}</dd>
              </div>
              <div v-if="tenantInfo?.updated_at" class="detail-row">
                <dt>{{ $t('tenant.overview.updatedAtLabel') }}</dt>
                <dd>{{ formatDate(tenantInfo.updated_at) }}</dd>
              </div>
            </dl>
          </div>
        </section>

        <!-- Depolama -->
        <section v-if="hasStorage" class="shell" style="--i: 2" aria-labelledby="tenant-storage-title">
          <div class="core panel">
            <header class="panel-head">
              <span class="panel-icon panel-icon--brand" aria-hidden="true"><t-icon name="server" /></span>
              <div class="panel-titles">
                <h3 id="tenant-storage-title">{{ $t('tenant.storage.usageLabel') }}</h3>
                <p>{{ $t('tenant.overview.storageHint') }}</p>
              </div>
            </header>

            <div class="storage-figure">
              <span class="storage-used">{{ formatBytes(tenantInfo?.storage_used || 0) }}</span>
              <span class="storage-quota">
                {{ isUnlimited
                  ? $t('tenant.overview.storageUnlimited')
                  : $t('tenant.overview.storageUsedOf', { quota: formatBytes(tenantInfo?.storage_quota || 0) }) }}
              </span>
            </div>

            <template v-if="!isUnlimited">
              <div class="meter" :class="`meter--${usageTone}`" role="progressbar" :aria-valuenow="usagePercentage"
                aria-valuemin="0" aria-valuemax="100" :aria-label="$t('tenant.storage.usageLabel')">
                <span class="meter-fill" :style="{ width: `${meterWidth}%` }" />
              </div>
              <div class="storage-foot">
                <span class="storage-percent" :class="`is-${usageTone}`">{{ usagePercentLabel }}</span>
                <span>{{ $t('tenant.overview.storageRemaining', { size: formatBytes(remainingBytes) }) }}</span>
              </div>
            </template>
          </div>
        </section>
      </div>

      <!-- Tehlikeli bölge -->
      <section v-if="showLeaveDangerZone || showDeleteDangerZone" class="shell shell--danger" style="--i: 3"
        aria-labelledby="tenant-danger-title">
        <div class="core panel">
          <header class="panel-head">
            <span class="panel-icon panel-icon--danger" aria-hidden="true"><t-icon name="error-circle" /></span>
            <div class="panel-titles">
              <h3 id="tenant-danger-title">{{ $t('tenant.overview.dangerTitle') }}</h3>
              <p>{{ $t('tenant.overview.dangerHint') }}</p>
            </div>
          </header>

          <div class="danger-list">
            <div v-if="showLeaveDangerZone" class="danger-row">
              <div class="danger-text">
                <span class="danger-title">{{ $t('tenant.leaveDangerZone.title') }}</span>
                <p class="danger-desc">{{ $t('tenant.leaveDangerZone.desc') }}</p>
              </div>
              <t-button theme="danger" variant="outline" class="danger-action" @click="confirmLeaveTenant">
                {{ $t('tenant.leaveDangerZone.button') }}
              </t-button>
            </div>

            <div v-if="showDeleteDangerZone" class="danger-row">
              <div class="danger-text">
                <span class="danger-title">{{ $t('tenant.deleteDangerZone.title') }}</span>
                <p class="danger-desc">{{ $t('tenant.deleteDangerZone.desc') }}</p>
              </div>
              <t-button theme="danger" class="danger-action" @click="confirmDeleteTenant">
                {{ $t('tenant.deleteDangerZone.button') }}
              </t-button>
            </div>
          </div>
        </div>
      </section>
    </template>

    <t-dialog v-model:visible="deleteTenantVisible" :header="$t('tenant.deleteDangerZone.confirmTitle')"
      :confirm-btn="{
        content: $t('tenant.deleteDangerZone.confirm'),
        theme: 'danger',
        disabled: deleteConfirmName.trim() !== (tenantInfo?.name || ''),
        loading: deletingTenant,
      }" :cancel-btn="$t('common.cancel')" :close-on-overlay-click="!deletingTenant"
      :close-btn="!deletingTenant" @confirm="deleteCurrentTenant">
      <div class="delete-tenant-confirm">
        <p class="delete-tenant-confirm-body">
          {{ $t('tenant.deleteDangerZone.confirmBody', { name: tenantInfo?.name || '' }) }}
        </p>
        <p class="delete-tenant-confirm-hint">
          {{ $t('tenant.deleteDangerZone.confirmHint', { name: tenantInfo?.name || '' }) }}
        </p>
        <t-input v-model="deleteConfirmName" :placeholder="tenantInfo?.name || ''" :disabled="deletingTenant"
          clearable />
      </div>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next'
import { EditIcon } from 'tdesign-icons-vue-next'
import { getCurrentUser, type TenantInfo } from '@/api/auth'
import { deleteTenant as deleteTenantApi, updateTenant as updateTenantApi } from '@/api/tenant'
import {
  leaveTenant,
  fetchAllTenantMembers,
  type TenantMember,
  type TenantRole,
} from '@/api/tenant/members'
import { useAuthStore } from '@/stores/auth'
import { useI18n } from 'vue-i18n'
import { useRoleLabel, useHomeTenant } from '@/composables/useRoleLabel'
import {
  navigateAfterTenantSwitch,
  persistLastActiveTenantPreference,
  stashTenantSwitchToast,
} from '@/utils/tenantSwitch'
import { copyWithToast } from '@/utils/clipboard'

const { t, locale } = useI18n()
const { formatRole, roleIcon } = useRoleLabel()
const { homeTenantId } = useHomeTenant()
const authStore = useAuthStore()

// Reactive state
const tenantInfo = ref<TenantInfo | null>(null)
const loading = ref(true)
const error = ref('')

// Yalnızca owner alan adını değiştirebilir (`router.go` içindeki `g.Owner()` korumasıyla uyumlu;
// Yetki konusunda nihai hakem her zaman sunucudur; burası yalnızca UI girişinin gösterilip gösterilmeyeceğini belirler).
const canEditTenant = computed(() => authStore.hasRole('owner'))

/** Özgün `TenantMembers.vue` ile uyumlu olarak: son Owner için ayrılma seçeneği gösterilmez; böylece sunucudaki last-owner ile hizalama hatası önlenir.*/
const activeTenantNumericId = computed(() => Number(authStore.currentTenantId ?? 0))

const leaveMembersSnap = ref<TenantMember[]>([])
const leaveGateReady = ref(false)
const leaveGateLoading = ref(false)

const currentTenantRole = computed<TenantRole | ''>(() => (authStore.currentTenantRole || '') as TenantRole | '')

const canLeaveSpace = computed(() => {
  const r = currentTenantRole.value
  if (!r || !tenantInfo.value?.id) return false
  if (r !== 'owner') return true
  return leaveMembersSnap.value.filter((m) => m.role === 'owner').length > 1
})

/** Ana içerik başarıyla yüklendiğinde, `listMembers` izin kuralı hazır olduğunda ve ayrılmaya izin verildiğinde görünür.*/
const showLeaveDangerZone = computed(() => {
  if (loading.value || error.value || !tenantInfo.value) return false
  if (!leaveGateReady.value || leaveGateLoading.value) return false
  if (!currentTenantRole.value) return false
  if (Number(tenantInfo.value.id) !== activeTenantNumericId.value) return false
  return canLeaveSpace.value
})

const showDeleteDangerZone = computed(() => {
  if (loading.value || error.value || !tenantInfo.value) return false
  if (Number(tenantInfo.value.id) !== activeTenantNumericId.value) return false
  return authStore.hasRole('owner')
})

async function evaluateLeaveGate(): Promise<void> {
  leaveGateReady.value = false
  leaveMembersSnap.value = []
  leaveGateLoading.value = false

  const infoId = tenantInfo.value?.id != null ? Number(tenantInfo.value.id) : 0
  if (!infoId || !activeTenantNumericId.value || infoId !== activeTenantNumericId.value) {
    leaveGateReady.value = true
    return
  }

  const role = currentTenantRole.value
  if (!role) {
    leaveGateReady.value = true
    return
  }
  if (role !== 'owner') {
    leaveGateReady.value = true
    return
  }

  leaveGateLoading.value = true
  try {
    leaveMembersSnap.value = await fetchAllTenantMembers(infoId)
  } finally {
    leaveGateLoading.value = false
    leaveGateReady.value = true
  }
}

function confirmLeaveTenant() {
  const tid = Number(tenantInfo.value?.id ?? 0)
  if (!tid) return

  const dlg = DialogPlugin.confirm({
    header: t('tenantMember.leave.confirmTitle'),
    body: t('tenantMember.leave.confirmBody'),
    confirmBtn: { content: t('tenantMember.leave.confirm'), theme: 'danger' },
    cancelBtn: t('common.cancel'),
    onConfirm: async () => {
      try {
        const resp = await leaveTenant(tid)
        if (resp.success) {
          MessagePlugin.success(t('tenantMember.leave.success'))
          authStore.logout()
          window.location.href = '/login'
        } else {
          MessagePlugin.error(resp.message || t('tenantMember.errors.generic'))
        }
      } catch (err: any) {
        const status = err?.status
        if (status === 409) {
          MessagePlugin.error(t('tenantMember.errors.lastOwner'))
        } else {
          MessagePlugin.error(err?.message || t('tenantMember.errors.generic'))
        }
      } finally {
        dlg.destroy()
      }
    },
    onClose: () => dlg.destroy(),
  })
}

function confirmDeleteTenant() {
  const tid = Number(tenantInfo.value?.id ?? 0)
  const tenantName = tenantInfo.value?.name || ''
  if (!tid || !tenantName) return
  deleteConfirmName.value = ''
  deleteTenantVisible.value = true
}

async function deleteCurrentTenant() {
  const tid = Number(tenantInfo.value?.id ?? 0)
  const tenantName = tenantInfo.value?.name || ''
  if (!tid || !tenantName) return
  if (deleteConfirmName.value.trim() !== tenantName) {
    MessagePlugin.warning(t('tenant.deleteDangerZone.nameMismatch'))
    return
  }
  try {
    deletingTenant.value = true
    const resp = await deleteTenantApi(tid)
    if (resp.success) {
      MessagePlugin.success(t('tenant.deleteDangerZone.success'))
      authStore.setMemberships(
        (authStore.memberships ?? []).filter((m) => m.tenant_id !== tid),
      )
      await authStore.refreshFromAuthMe()
      const next =
        authStore.memberships.find((m) => m.tenant_id === homeTenantId.value) ??
        authStore.memberships[0]
      if (next) {
        const switchingToHome =
          homeTenantId.value !== null && homeTenantId.value === next.tenant_id
        const name = next.tenant_name?.trim() || `#${next.tenant_id}`
        authStore.setSelectedTenant(next.tenant_id, name)
        stashTenantSwitchToast({
          name,
          role: formatRole(next.role) || undefined,
          roleEnum: next.role || undefined,
        })
        const persist = persistLastActiveTenantPreference(
          switchingToHome ? null : next.tenant_id,
        )
        await Promise.race([persist, new Promise((r) => setTimeout(r, 400))])
        navigateAfterTenantSwitch()
        return
      }
      authStore.logout()
      window.location.href = '/login'
    } else {
      MessagePlugin.error(resp.message || t('tenant.deleteDangerZone.failed'))
    }
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('tenant.deleteDangerZone.failed'))
  } finally {
    deletingTenant.value = false
    deleteConfirmName.value = ''
    deleteTenantVisible.value = false
  }
}

watch(
  [() => tenantInfo.value?.id, () => authStore.currentTenantId, () => authStore.currentTenantRole],
  () => {
    if (!loading.value && tenantInfo.value && !error.value) {
      void evaluateLeaveGate()
    }
  },
)

// Alan adını yerinde düzenleme: `editing`, satır içi salt okunur / düzenleme biçimleri arasındaki geçişi yönetir.
// Burada yalnızca bir alan olduğu için dialog kullanılmaz; açılır pencere yapılandırma inceleme akışını gereksiz yere keser.
const editing = ref(false)
const editName = ref('')
const saving = ref(false)
const deleteConfirmName = ref('')
const deleteTenantVisible = ref(false)
const deletingTenant = ref(false)
const editNameTrimmed = computed(() => editName.value.trim())
// Kaydet düğmesinin etkin olma koşulları: boş olmaması, içeriğin değiştirilmiş olması ve kaydetme işleminin sürmemesi.
// Arka uçtaki `name` alanında ne `uniqueIndex` ne de yinelenen ad denetimi var; bu nedenle burada "zaten var mı" kontrolü yapılmaz;
// Arka uçtaki service de yalnızca create sırasında boş değeri reddeder, update sırasında doğrulamaz; ön yüzün boş olmama koruması yeterlidir.
const canSubmit = computed(
  () => !saving.value && !!editNameTrimmed.value && editNameTrimmed.value !== tenantInfo.value?.name,
)

const startEditName = () => {
  editName.value = tenantInfo.value?.name || ''
  editing.value = true
}

const cancelEditName = () => {
  if (saving.value) return
  editing.value = false
  editName.value = ''
}

// `t-input` kendi başına esc olayını yaymaz; burada elle işlenir (`enter` deneyimiyle simetrik).
const onEditKeydown = (_value: any, ctx: { e: KeyboardEvent }) => {
  if (ctx?.e?.key === 'Escape') {
    cancelEditName()
  }
}

// Alan açıklamasını yerinde düzenleme: adla simetrik `editing` / `editValue` / `saving` üç durumu.
// Açıklama boş olabilir (iş açısından isteğe bağlı bir alandır); bu nedenle gönderim koşulu boş olmamasını gerektirmez, yalnızca içeriğin değişmiş olması yeterlidir.
const editingDescription = ref(false)
const editDescription = ref('')
const savingDescription = ref(false)
const editDescriptionTrimmed = computed(() => editDescription.value.trim())
const canSubmitDescription = computed(
  () => !savingDescription.value && editDescriptionTrimmed.value !== (tenantInfo.value?.description || ''),
)

const startEditDescription = () => {
  editDescription.value = tenantInfo.value?.description || ''
  editingDescription.value = true
}

const cancelEditDescription = () => {
  if (savingDescription.value) return
  editingDescription.value = false
  editDescription.value = ''
}

// textarea üzerinde Enter varsayılan olarak yeni satır ekler; gönderim Ctrl/⌘+Enter ile yapılır; Esc iptal eder.
const onEditDescriptionKeydown = (_value: any, ctx: { e: KeyboardEvent }) => {
  const e = ctx?.e
  if (!e) return
  if (e.key === 'Escape') {
    cancelEditDescription()
    return
  }
  if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
    e.preventDefault()
    void saveTenantDescription()
  }
}

const saveTenantDescription = async () => {
  if (!tenantInfo.value?.id) return
  const newDesc = editDescriptionTrimmed.value
  if (newDesc === (tenantInfo.value.description || '')) {
    editingDescription.value = false
    return
  }

  try {
    savingDescription.value = true
    const resp = await updateTenantApi(Number(tenantInfo.value.id), { description: newDesc })
    if (resp.success) {
      // `/auth/me` gidiş dönüşünü beklememek için yerelde hemen yansıtılır. Açıklama, ad gibi alan değiştirici vb.
      // üst bileşenlerde görünmez; bu nedenle `authStore.tenant` / `memberships` eşitlemesi gerekmez.
      if (tenantInfo.value) {
        tenantInfo.value = { ...tenantInfo.value, description: newDesc }
      }
      MessagePlugin.success(t('tenant.details.editDescriptionSuccess'))
      editingDescription.value = false
    } else {
      MessagePlugin.error(resp.message || t('tenant.details.editDescriptionFailed'))
    }
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('tenant.details.editDescriptionFailed'))
  } finally {
    savingDescription.value = false
  }
}

const saveTenantName = async () => {
  const newName = editNameTrimmed.value
  if (!newName) {
    MessagePlugin.warning(t('tenant.details.editNameRequired'))
    return
  }
  if (!tenantInfo.value?.id) return
  if (newName === tenantInfo.value.name) {
    editing.value = false
    return
  }

  try {
    saving.value = true
    const resp = await updateTenantApi(Number(tenantInfo.value.id), { name: newName })
    if (resp.success) {
      // `/auth/me` gidiş dönüşünü beklememek için yerelde hemen yansıtılır; oturum durumundaki tenant
      // önbelleği de yenilenir (mevcut etkin alan home tenant ise, üstteki alan değiştirici vb. yerler de güncellenir).
      if (tenantInfo.value) {
        tenantInfo.value = { ...tenantInfo.value, name: newName }
      }
      if (authStore.tenant && String(authStore.tenant.id) === String(tenantInfo.value?.id)) {
        authStore.setTenant({ ...authStore.tenant, name: newName })
      }
      // `memberships` içindeki `tenant_name`, alan değiştiricinin okuduğu alandır; eski adın gösterilmesini önlemek için birlikte eşitlenir.
      if (authStore.memberships?.length) {
        const next = authStore.memberships.map((m) =>
          String(m.tenant_id) === String(tenantInfo.value?.id)
            ? { ...m, tenant_name: newName }
            : m,
        )
        authStore.setMemberships(next)
      }
      MessagePlugin.success(t('tenant.details.editNameSuccess'))
      editing.value = false
    } else {
      MessagePlugin.error(resp.message || t('tenant.details.editNameFailed'))
    }
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('tenant.details.editNameFailed'))
  } finally {
    saving.value = false
  }
}

// Methods
const loadInfo = async () => {
  try {
    loading.value = true
    error.value = ''

    const userResponse = await getCurrentUser()

    const data = userResponse?.data as { tenant?: TenantInfo } | undefined
    if ((userResponse as any).success && data?.tenant) {
      tenantInfo.value = data.tenant
    } else {
      error.value = userResponse.message || t('tenant.messages.fetchFailed')
    }
  } catch (err: any) {
    error.value = err?.message || t('tenant.messages.networkError')
  } finally {
    loading.value = false
  }
  // `loading=false` sonrasında değerlendirilmelidir: aksi halde ayrılma girişi `showLeaveDangerZone` içindeki `loading` koşulu tarafından engellenir,
  // ayrıca bazı ortamlarda rol hydration işlemi `/auth/me` dönüşünden biraz sonra tamamlanır.
  if (tenantInfo.value && !error.value) {
    await evaluateLeaveGate()
  }
}

const getStatusText = (status: string | undefined) => {
  switch (status) {
    case 'active':
      return t('tenant.statusActive')
    case 'inactive':
      return t('tenant.statusInactive')
    case 'suspended':
      return t('tenant.statusSuspended')
    default:
      return t('tenant.statusUnknown')
  }
}

const statusTone = computed(() => {
  switch (tenantInfo.value?.status) {
    case 'active':
      return 'success'
    case 'inactive':
      return 'warning'
    case 'suspended':
      return 'danger'
    default:
      return 'neutral'
  }
})

const roleLabel = computed(() => formatRole(currentTenantRole.value))

// Kimlik kartındaki monogram: adın ilk iki kelimesinin baş harfleri (tek kelimede ilk iki harf).
const initials = computed(() => {
  const words = (tenantInfo.value?.name || '').trim().split(/\s+/).filter(Boolean)
  if (!words.length) return '#'
  const letters = words.length > 1 ? words[0][0] + words[1][0] : words[0].slice(0, 2)
  return letters.toLocaleUpperCase(locale.value || 'tr-TR')
})

const copyTenantId = () => copyWithToast(tenantInfo.value?.id != null ? String(tenantInfo.value.id) : '', 'tenant.overview.copied')

// storage_quota <= 0 sunucuda "sınırsız" anlamına gelir (bkz. internal/handler/tenant.go); yüzde çubuğu yerine sınırsız etiketi gösterilir.
const hasStorage = computed(() => tenantInfo.value?.storage_quota !== undefined)
const isUnlimited = computed(() => (tenantInfo.value?.storage_quota ?? 0) <= 0)
const remainingBytes = computed(() =>
  Math.max((tenantInfo.value?.storage_quota || 0) - (tenantInfo.value?.storage_used || 0), 0),
)
const usagePercentage = computed(() => getUsagePercentage())
const usagePercentLabel = computed(() =>
  new Intl.NumberFormat(locale.value || 'tr-TR', { style: 'percent', maximumFractionDigits: 1 }).format(
    usagePercentage.value / 100,
  ),
)
// Çok küçük ama sıfırdan büyük kullanım da çubukta görünür kalsın.
const meterWidth = computed(() => (usagePercentage.value > 0 ? Math.max(usagePercentage.value, 2) : 0))
const usageTone = computed(() => {
  if (usagePercentage.value >= 90) return 'danger'
  if (usagePercentage.value >= 75) return 'warning'
  return 'success'
})

const formatDay = (dateStr: string | undefined) => {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  if (Number.isNaN(date.getTime())) return ''
  return new Intl.DateTimeFormat(locale.value || 'tr-TR', { year: 'numeric', month: 'long', day: 'numeric' }).format(date)
}

const formatDate = (dateStr: string | undefined) => {
  if (!dateStr) return t('tenant.unknown')

  try {
    const date = new Date(dateStr)
    const formatter = new Intl.DateTimeFormat(locale.value || 'tr-TR', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    })
    return formatter.format(date)
  } catch {
    return t('tenant.formatError')
  }
}

const formatBytes = (bytes: number) => {
  if (bytes === 0) return '0 B'

  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))

  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const getUsagePercentage = () => {
  if (!tenantInfo.value?.storage_quota || tenantInfo.value.storage_quota === 0) {
    return 0
  }

  const used = tenantInfo.value.storage_used || 0
  const percentage = (used / tenantInfo.value.storage_quota) * 100
  return Math.min(Math.round(percentage * 100) / 100, 100)
}

// Lifecycle
onMounted(() => {
  loadInfo()
})
</script>


<style lang="less" scoped>
@import (reference) '@/components/css/settings-section.less';

@ease: cubic-bezier(0.32, 0.72, 0, 1);
@shell-pad: 5px;

/* UserProfile.vue ile aynı "çift çerçeve" yüzey dili; ayarlar sayfaları arasında görsel süreklilik sağlar. */
.tenant-info {
  width: 100%;
  container-type: inline-size;
  display: flex;
  flex-direction: column;
  gap: var(--app-space-4);

  --ti-shell: var(--app-surface-muted);
  --ti-card: var(--td-bg-color-container);
  --ti-ring: color-mix(in srgb, var(--td-component-stroke) 70%, transparent);
  --ti-highlight: inset 0 1px 0 rgba(255, 255, 255, 0.9);
  --ti-lift: 0 1px 2px rgba(0, 0, 0, 0.03), 0 10px 28px -18px rgba(0, 0, 0, 0.14);
}

:root[theme-mode='dark'] .tenant-info {
  --ti-shell: #1a1a1a;
  --ti-card: #242424;
  --ti-ring: rgba(255, 255, 255, 0.07);
  --ti-highlight: inset 0 1px 0 rgba(255, 255, 255, 0.05);
  --ti-lift: 0 1px 2px rgba(0, 0, 0, 0.3), 0 12px 32px -18px rgba(0, 0, 0, 0.7);
}

.section-header {
  .settings-section-header();
  margin-bottom: var(--app-space-2);
}

.error-inline {
  padding: var(--app-space-2) 0;
}

/* ---------- Surfaces ---------- */

.shell {
  padding: @shell-pad;
  border-radius: var(--app-radius-2xl);
  background: var(--ti-shell);
  box-shadow: 0 0 0 1px var(--ti-ring);
  animation: tenant-rise 560ms @ease both;
  animation-delay: calc(var(--i, 0) * 70ms);
}

.core {
  border-radius: calc(var(--app-radius-2xl) - @shell-pad);
  background: var(--ti-card);
  box-shadow: 0 0 0 1px var(--ti-ring), var(--ti-highlight), var(--ti-lift);
}

/* ---------- Identity hero ---------- */

.hero {
  position: relative;
  display: flex;
  align-items: flex-start;
  gap: var(--app-space-5);
  padding: var(--app-space-6);
  overflow: hidden;
  background:
    radial-gradient(120% 160% at 0% 0%, color-mix(in srgb, var(--td-brand-color) 9%, transparent) 0%, transparent 55%),
    var(--ti-card);
}

.hero--skeleton {
  align-items: center;

  .skeleton-lines {
    flex: 1;
    min-width: 0;
  }
}

.ws-mark {
  flex: none;
  width: 72px;
  height: 72px;
  padding: 3px;
  border-radius: 20px;
  background: linear-gradient(
    140deg,
    color-mix(in srgb, var(--td-brand-color) 55%, transparent),
    color-mix(in srgb, var(--td-brand-color) 10%, transparent)
  );
  user-select: none;
}

.ws-mark-initials {
  display: grid;
  place-items: center;
  width: 100%;
  height: 100%;
  border-radius: 17px;
  background: linear-gradient(150deg, var(--td-brand-color-6) 0%, var(--td-brand-color-8) 100%);
  box-shadow: 0 0 0 3px var(--ti-card), inset 0 1px 0 rgba(255, 255, 255, 0.25);
  color: #fff;
  font-family: var(--app-font-heading);
  font-size: var(--app-text-4xl);
  font-weight: 600;
  letter-spacing: 0.02em;
  line-height: 1;
}

.hero-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: var(--app-space-1);
  padding-top: 2px;
}

.hero-name-row,
.hero-desc-row {
  display: flex;
  align-items: center;
  gap: var(--app-space-1);
  min-width: 0;

  .icon-btn {
    opacity: 0.55;
  }

  &:hover .icon-btn,
  .icon-btn:focus-visible {
    opacity: 1;
  }
}

.hero-desc-row {
  align-items: flex-start;
}

.hero-name {
  margin: 0;
  min-width: 0;
  font-family: var(--app-font-heading);
  font-size: var(--app-text-4xl);
  font-weight: 600;
  line-height: 1.25;
  letter-spacing: -0.015em;
  color: var(--td-text-color-primary);
  overflow-wrap: anywhere;
}

.hero-desc {
  margin: 0;
  min-width: 0;
  max-width: 64ch;
  font-size: var(--app-text-md);
  line-height: 1.55;
  color: var(--td-text-color-secondary);
  white-space: pre-wrap;
  overflow-wrap: anywhere;

  &.is-empty {
    color: var(--td-text-color-placeholder);
    font-style: italic;
  }
}

.hero-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin: var(--app-space-2) 0 0;
  padding: 0;
  list-style: none;
}

.meta-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 24px;
  padding: 0 10px;
  border-radius: var(--app-radius-pill);
  background: var(--ti-shell);
  box-shadow: inset 0 0 0 1px var(--ti-ring);
  font-size: var(--app-text-sm);
  font-weight: 500;
  color: var(--td-text-color-secondary);
  white-space: nowrap;

  .t-icon {
    font-size: var(--app-text-md);
    color: var(--td-text-color-placeholder);
  }

  .status-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--td-text-color-placeholder);
  }

  &--success {
    --tone: var(--td-success-color);
  }

  &--warning {
    --tone: var(--td-warning-color);
  }

  &--danger {
    --tone: var(--td-error-color);
  }

  &--success,
  &--warning,
  &--danger {
    background: color-mix(in srgb, var(--tone) 10%, transparent);
    box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--tone) 24%, transparent);
    color: var(--tone);

    .status-dot {
      background: var(--tone);
      box-shadow: 0 0 0 3px color-mix(in srgb, var(--tone) 20%, transparent);
    }
  }
}

.icon-btn {
  flex: none;
  display: grid;
  place-items: center;
  width: 30px;
  height: 30px;
  padding: 0;
  border: 0;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: pointer;
  transition: background-color var(--app-motion-fast) @ease, color var(--app-motion-fast) @ease,
    opacity var(--app-motion-fast) @ease;

  &:hover {
    background: var(--ti-shell);
    color: var(--td-text-color-primary);
  }

  &:focus-visible {
    outline: 2px solid var(--app-ring-soft);
    outline-offset: 1px;
  }

  &--sm {
    width: 26px;
    height: 26px;
  }
}

/* ---------- Inline edit ---------- */

.inline-edit {
  display: flex;
  align-items: center;
  gap: var(--app-space-2);
  width: 100%;
  max-width: 460px;
}

.inline-edit-input {
  flex: 1;
  min-width: 0;

  :deep(.t-input) {
    height: 36px;
    border-radius: var(--app-radius-md);
    font-size: var(--app-text-lg);
    font-weight: 500;
  }
}

.inline-edit--stacked {
  flex-direction: column;
  align-items: stretch;
  max-width: 560px;
  margin-top: var(--app-space-1);
}

.inline-edit-textarea {
  width: 100%;

  :deep(.t-textarea__inner) {
    border-radius: var(--app-radius-md);
  }
}

.inline-edit-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--app-space-2);
  flex-shrink: 0;
}

/* ---------- Panels ---------- */

.panel {
  display: flex;
  flex-direction: column;
  gap: var(--app-space-5);
  padding: var(--app-space-6);
  min-width: 0;
}

.panel-head {
  display: flex;
  align-items: flex-start;
  gap: var(--app-space-3);
}

.panel-icon {
  flex: none;
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  border-radius: var(--app-radius-md);
  background: var(--ti-shell);
  box-shadow: inset 0 0 0 1px var(--ti-ring);
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xl);

  &--brand {
    background: color-mix(in srgb, var(--td-brand-color) 10%, transparent);
    box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--td-brand-color) 22%, transparent);
    color: var(--td-brand-color);
  }

  &--danger {
    background: color-mix(in srgb, var(--td-error-color) 9%, transparent);
    box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--td-error-color) 22%, transparent);
    color: var(--td-error-color);
  }
}

.panel-titles {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding-top: 1px;

  h3 {
    margin: 0;
    font-family: var(--app-font-heading);
    font-size: var(--app-text-lg);
    font-weight: 600;
    line-height: 1.35;
    color: var(--td-text-color-primary);
  }

  p {
    margin: 0;
    max-width: 60ch;
    font-size: var(--app-text-sm);
    line-height: 1.5;
    color: var(--td-text-color-secondary);
  }
}

/* ---------- Bento ---------- */

.bento {
  display: grid;
  grid-template-columns: minmax(0, 1.25fr) minmax(0, 1fr);
  gap: var(--app-space-4);
  align-items: stretch;

  &--single {
    grid-template-columns: minmax(0, 1fr);
  }

  > .shell {
    display: flex;

    > .core {
      flex: 1;
    }
  }
}

.detail-list {
  margin: 0;
  display: flex;
  flex-direction: column;
}

.detail-row {
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: var(--app-space-3) 0;

  & + & {
    border-top: 1px dashed var(--ti-ring);
  }

  &:first-child {
    padding-top: 0;
  }

  &:last-child {
    padding-bottom: 0;
  }

  dt {
    font-size: var(--app-text-xs);
    font-weight: 500;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    color: var(--td-text-color-placeholder);
  }

  dd {
    margin: 0;
    font-size: var(--app-text-base);
    color: var(--td-text-color-primary);
    font-variant-numeric: tabular-nums;
    overflow-wrap: anywhere;
  }
}

.detail-id {
  display: flex;
  align-items: center;
  gap: var(--app-space-2);
  min-width: 0;

  .mono {
    min-width: 0;
    font-family: var(--app-font-family-mono);
    font-size: var(--app-text-sm);
    color: var(--td-text-color-secondary);
    overflow-wrap: anywhere;
  }
}

/* ---------- Storage ---------- */

.storage-figure {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-top: auto;
}

.storage-used {
  font-family: var(--app-font-heading);
  font-size: var(--app-text-4xl);
  font-weight: 600;
  line-height: 1.15;
  letter-spacing: -0.02em;
  color: var(--td-text-color-primary);
  font-variant-numeric: tabular-nums;
}

.storage-quota {
  font-size: var(--app-text-sm);
  color: var(--td-text-color-secondary);
}

.meter {
  position: relative;
  height: 8px;
  border-radius: var(--app-radius-pill);
  background: var(--ti-shell);
  box-shadow: inset 0 0 0 1px var(--ti-ring);
  overflow: hidden;

  --tone: var(--td-success-color);

  &--warning {
    --tone: var(--td-warning-color);
  }

  &--danger {
    --tone: var(--td-error-color);
  }
}

.meter-fill {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, color-mix(in srgb, var(--tone) 70%, transparent), var(--tone));
  transition: width var(--app-motion-slow) @ease;
}

.storage-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--app-space-3);
  margin-top: calc(var(--app-space-3) * -1);
  font-size: var(--app-text-sm);
  color: var(--td-text-color-secondary);
  font-variant-numeric: tabular-nums;
}

.storage-percent {
  font-weight: 600;

  &.is-success {
    color: var(--td-text-color-primary);
  }

  &.is-warning {
    color: var(--td-warning-color);
  }

  &.is-danger {
    color: var(--td-error-color);
  }
}

/* ---------- Danger zone ---------- */

.shell--danger {
  background: color-mix(in srgb, var(--td-error-color) 5%, var(--ti-shell));
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--td-error-color) 18%, var(--ti-ring));
}

.danger-list {
  display: flex;
  flex-direction: column;
}

.danger-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--app-space-5);
  padding: var(--app-space-4) 0;

  & + & {
    border-top: 1px dashed var(--ti-ring);
  }

  &:first-child {
    padding-top: 0;
  }

  &:last-child {
    padding-bottom: 0;
  }
}

.danger-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.danger-title {
  font-size: var(--app-text-base);
  font-weight: 500;
  color: var(--td-text-color-primary);
}

.danger-desc {
  margin: 0;
  max-width: 60ch;
  font-size: var(--app-text-sm);
  line-height: 1.55;
  color: var(--td-text-color-secondary);
}

.danger-action {
  flex: none;
  min-width: 150px;
  height: 36px;
  padding: 0 var(--app-space-4);
  border-radius: var(--app-radius-pill);
  transition: transform var(--app-motion-fast) @ease;

  &:active:not(.t-is-disabled) {
    transform: scale(0.98);
  }
}

/* ---------- Delete dialog ---------- */

.delete-tenant-confirm-body {
  margin: 0 0 10px;
  color: var(--td-text-color-primary);
  line-height: 1.6;
}

.delete-tenant-confirm-hint {
  margin: 0 0 12px;
  color: var(--td-text-color-secondary);
  line-height: 1.5;
}

@keyframes tenant-rise {
  from {
    opacity: 0;
    transform: translateY(10px);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* ---------- Narrow layouts ---------- */

@container (max-width: 720px) {
  .bento {
    grid-template-columns: minmax(0, 1fr);
  }
}

@container (max-width: 560px) {
  .hero {
    flex-direction: column;
    gap: var(--app-space-4);
    padding: var(--app-space-5);
  }

  .ws-mark {
    width: 60px;
    height: 60px;
    border-radius: 17px;
  }

  .ws-mark-initials {
    border-radius: 14px;
    font-size: var(--app-text-3xl);
  }

  .hero-name {
    font-size: var(--app-text-3xl);
  }

  .inline-edit {
    flex-direction: column;
    align-items: stretch;
  }

  .panel {
    padding: var(--app-space-5);
  }

  .danger-row {
    flex-direction: column;
    align-items: stretch;
    gap: var(--app-space-3);
  }

  .danger-action {
    width: 100%;
  }
}

@media (prefers-reduced-motion: reduce) {
  .shell {
    animation: none;
  }

  .icon-btn,
  .meter-fill,
  .danger-action {
    transition: none;
  }
}
</style>
