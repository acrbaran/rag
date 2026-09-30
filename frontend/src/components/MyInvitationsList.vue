<template>
  <!-- Pending workspace invitations. Rendered inside MyInvitationsDialog
       (onboarding) and, in compact form, inside the sidebar bell popover
       next to the announcements tab. The list reloads each time `active`
       turns true so accept/decline actions taken in another tab show up. -->
  <div class="my-invitations" :class="{ 'my-invitations--compact': compact }">
    <div v-if="compact" class="my-invitations__header">
      <h2 class="my-invitations__title">{{ $t('tenantInvitation.myInbox.title') }}</h2>
      <button type="button" class="ann-btn ann-btn--ghost ann-btn--icon-sm" :aria-label="$t('announcements.dismiss')"
        @click="emit('close')">
        <LucideIcon name="x" />
      </button>
    </div>
    <p class="my-invitations-desc">{{ $t('tenantInvitation.myInbox.description') }}</p>

    <div class="my-invitations__body">
      <div v-if="loading" class="loading-inline">
        <t-loading size="small" />
        <span>{{ $t('tenantMember.loading') }}</span>
      </div>

      <div v-else-if="error" class="error-inline">
        <t-alert theme="error" :message="error">
          <template #operation>
            <t-button size="small" @click="reload">{{ $t('tenantMember.retry') }}</t-button>
          </template>
        </t-alert>
      </div>

      <p v-else-if="invitations.length === 0 && compact" class="my-invitations__empty">
        {{ $t('tenantInvitation.myInbox.empty') }}
      </p>
      <div v-else-if="invitations.length === 0" class="empty-state">
        <t-empty :description="$t('tenantInvitation.myInbox.empty')" />
      </div>

      <ul v-else class="invitation-list">
        <li v-for="row in invitations" :key="row.id" class="invitation-card">
          <div class="invitation-card-main">
            <div class="invitation-card-header">
              <span class="tenant-name">
                {{ row.tenant_name || $t('tenantInvitation.myInbox.tenantLabel') + ' #' + row.tenant_id }}
              </span>
              <t-tag :theme="roleTagTheme(row.role)" size="small">
                {{ $t('tenantMember.role.' + row.role) }}
              </t-tag>
            </div>
            <div class="invitation-card-meta">
              <span class="meta-row">
                <t-icon name="user" size="14px" class="meta-icon" />
                <span class="meta-label">{{ $t('tenantInvitation.myInbox.from') }}：</span>
                <span class="meta-value">{{ inviterDisplay(row) }}</span>
              </span>
              <span class="meta-row">
                <t-icon name="time" size="14px" class="meta-icon" />
                <span class="meta-value">
                  {{ $t('tenantInvitation.myInbox.expiresIn', { date: formatDate(row.expires_at) }) }}
                </span>
              </span>
              <span v-if="row.message" class="meta-row meta-row--message">
                <t-icon name="chat" size="14px" class="meta-icon" />
                <span class="meta-label">{{ $t('tenantInvitation.myInbox.messageLabel') }}：</span>
                <span class="meta-value">{{ row.message }}</span>
              </span>
            </div>
          </div>
          <div class="invitation-card-actions">
            <t-button theme="primary" size="small" :loading="acting === row.id" @click="onAccept(row)">
              {{ $t('tenantInvitation.myInbox.acceptButton') }}
            </t-button>
            <t-button theme="default" variant="outline" size="small" :loading="acting === row.id"
              @click="onDecline(row)">
              {{ $t('tenantInvitation.myInbox.declineButton') }}
            </t-button>
          </div>
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { MessagePlugin } from 'tdesign-vue-next'
import { useAuthStore } from '@/stores/auth'
import {
  listMyInvitations,
  acceptInvitation,
  declineInvitation,
  type TenantInvitation,
} from '@/api/tenant/invitations'
import type { TenantRole } from '@/api/tenant/members'
import LucideIcon from '@/components/announcements/LucideIcon.vue'

const props = withDefaults(defineProps<{ active: boolean; compact?: boolean }>(), { compact: false })
const emit = defineEmits<{ (e: 'close'): void }>()

const { t, locale } = useI18n()
const authStore = useAuthStore()

const invitations = ref<TenantInvitation[]>([])
const loading = ref(false)
const error = ref('')
const acting = ref<number | null>(null)

function roleTagTheme(role: TenantRole): 'primary' | 'warning' | 'success' | 'default' {
  switch (role) {
    case 'owner':
      return 'primary'
    case 'admin':
      return 'warning'
    case 'contributor':
      return 'success'
    default:
      return 'default'
  }
}

function inviterDisplay(row: TenantInvitation): string {
  return row.inviter_name?.trim() || row.inviter_email?.trim() || row.invited_by || '—'
}

function formatDate(s: string): string {
  if (!s) return '-'
  try {
    return new Intl.DateTimeFormat(locale.value || 'tr-TR', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    }).format(new Date(s))
  } catch {
    return s
  }
}

async function reload() {
  loading.value = true
  error.value = ''
  try {
    const resp = await listMyInvitations()
    if (resp.success && resp.data) {
      invitations.value = resp.data.invitations
      authStore.setPendingInvitationCount(
        invitations.value.filter((i) => i.status === 'pending').length,
      )
    } else {
      error.value = resp.message || t('tenantInvitation.errors.generic')
    }
  } catch (err: any) {
    error.value = err?.message || t('tenantInvitation.errors.generic')
  } finally {
    loading.value = false
  }
}

function actionError(err: any) {
  const status = err?.status
  if (status === 404) MessagePlugin.error(t('tenantInvitation.errors.notFound'))
  else if (status === 403) MessagePlugin.error(t('tenantInvitation.errors.forbidden'))
  else if (status === 409) MessagePlugin.error(err?.message || t('tenantInvitation.errors.notPending'))
  else MessagePlugin.error(err?.message || t('tenantInvitation.errors.generic'))
}

async function onAccept(row: TenantInvitation) {
  acting.value = row.id
  try {
    const resp = await acceptInvitation(row.id)
    if (resp.success) {
      invitations.value = invitations.value.filter((x) => x.id !== row.id)
      authStore.setPendingInvitationCount(Math.max(0, authStore.pendingInvitationCount - 1))
      await authStore.refreshFromAuthMe()
      MessagePlugin.success(
        t('tenantInvitation.myInbox.acceptSuccess', {
          tenant: row.tenant_name || `#${row.tenant_id}`,
        }),
      )
    } else {
      MessagePlugin.error(resp.message || t('tenantInvitation.errors.generic'))
    }
  } catch (err: any) {
    actionError(err)
  } finally {
    acting.value = null
  }
}

async function onDecline(row: TenantInvitation) {
  acting.value = row.id
  try {
    const resp = await declineInvitation(row.id)
    if (resp.success) {
      invitations.value = invitations.value.filter((x) => x.id !== row.id)
      authStore.setPendingInvitationCount(Math.max(0, authStore.pendingInvitationCount - 1))
      MessagePlugin.success(t('tenantInvitation.myInbox.declineSuccess'))
    } else {
      MessagePlugin.error(resp.message || t('tenantInvitation.errors.generic'))
    }
  } catch (err: any) {
    actionError(err)
  } finally {
    acting.value = null
  }
}

watch(
  () => props.active,
  (v) => {
    if (v) reload()
  },
  { immediate: true },
)
</script>

<style lang="less" scoped>
.my-invitations-desc {
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-md);
  line-height: 1.55;
  margin: 0 0 16px 0;
}

.loading-inline,
.error-inline {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 0;
}

.empty-state {
  padding: 16px 0 8px;
  display: flex;
  justify-content: center;
}

.invitation-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
  /* Cap height so a user with many invitations can scroll within
     the dialog rather than the dialog growing past the viewport. */
  max-height: 60vh;
  overflow-y: auto;
}

.invitation-card {
  display: flex;
  align-items: stretch;
  gap: 12px;
  padding: 12px 14px;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-md);

  .invitation-card-main {
    flex: 1 1 auto;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .invitation-card-header {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;

    .tenant-name {
      font-size: var(--app-text-base);
      font-weight: 500;
      color: var(--td-text-color-primary);
    }
  }

  .invitation-card-meta {
    display: flex;
    flex-direction: column;
    gap: 3px;
    color: var(--td-text-color-secondary);
    font-size: var(--app-text-sm);

    .meta-row {
      display: inline-flex;
      align-items: center;
      gap: 4px;

      .meta-icon {
        color: var(--td-text-color-placeholder);
        flex-shrink: 0;
      }

      .meta-value {
        color: var(--td-text-color-primary);
      }

      &--message {
        align-items: flex-start;

        .meta-icon {
          margin-top: 2px;
        }

        .meta-value {
          word-break: break-word;
        }
      }
    }
  }

  .invitation-card-actions {
    display: flex;
    flex-direction: column;
    align-items: stretch;
    justify-content: center;
    gap: 6px;
    flex-shrink: 0;
  }
}

// ---------- Compact (bell popover) — same frame as the announcements tab ----------
.my-invitations--compact {
  color: var(--td-text-color-primary);
  font-size: var(--app-text-base);
  text-align: left;

  .my-invitations-desc {
    margin: 0;
    padding: 12px 16px;
    border-bottom: 1px solid var(--td-component-stroke);
    font-size: var(--app-text-xs);
  }

  .my-invitations__body {
    max-height: min(28rem, 60dvh);
    overflow-y: auto;
    overscroll-behavior: contain;
  }

  .loading-inline,
  .error-inline {
    padding: 16px;
  }

  .invitation-list {
    gap: 0;
    max-height: none;
    overflow: visible;

    > li + li {
      border-top: 1px solid var(--td-component-stroke);
    }
  }

  .invitation-card {
    flex-direction: column;
    gap: 12px;
    padding: 16px;
    border: 0;
    border-radius: 0;
    background: transparent;

    .invitation-card-actions {
      flex-direction: row;
      justify-content: flex-start;
    }
  }
}

.my-invitations__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.my-invitations__title {
  margin: 0;
  font-size: var(--app-text-base);
  font-weight: 600;
  line-height: 1.5;
}

.my-invitations__empty {
  margin: 0;
  padding: 32px 16px;
  color: var(--td-text-color-secondary);
  text-align: center;
}
</style>
