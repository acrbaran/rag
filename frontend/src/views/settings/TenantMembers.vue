<template>
  <div class="tenant-members">
    <!-- Section header. The (i) permission speed-look popover lives
         next to the title so it reads as meta-info about *this
         section*. The audit-log entry sits on the right of the header
         row — secondary navigation that opens the audit drawer; gated
         to Admin+ so non-managers don't see a button they can't use. -->
    <div class="section-header">
      <div class="section-header-row">
        <div class="section-header-titlewrap">
          <h2>{{ $t('tenantMember.title') }}</h2>
          <TenantRolePermissionsPopover :current-role="currentRole" />
          <!-- Audit log entry sits inline with the title: title (i)
               [审计日志]. Keeping all section-level affordances on the
               left edge avoids the "lonely right-aligned button"
               pattern in narrow settings panels. -->
          <t-button v-if="canViewAudit" variant="text" size="small" class="header-audit-btn" @click="openAuditDrawer">
            <template #icon><t-icon name="history" /></template>
            {{ $t('tenantMember.audit.tabLabel') }}
          </t-button>
        </div>
      </div>
      <p class="section-description">
        {{ $t('tenantMember.sectionDescription') }}
      </p>
    </div>

    <div class="members-tab-layout">
      <!-- Toolbar, 「Alan üyeleri」 liste başlığına birleştirildi: arama kutusu liste başlığının hemen sağında
           yer alır, davet düğmesi onun da bir simge sağına konur; 「bu listeye yönelik」 tüm denetimler
           aynı satırda toplanır, bağımsız toolbar artık yoktur. -->

      <!-- Pending invitations. Shown only to managers because the
               viewer/contributor roles don't have an action surface
               here. Even when empty we still render the header so
               operators get a stable "is there anything pending?"
               affordance after they hit "Send invitation". -->
      <div v-if="canManage" class="pending-invitations-section">
        <div class="pending-invitations-header">
          <div class="pending-invitations-titlewrap">
            <span class="pending-invitations-title">
              {{ $t('tenantInvitation.pendingSectionTitle') }}
            </span>
            <!-- Same count-badge style as the «空间成员» list header
                 so the two list titles read at parity. -->
            <span class="members-list-count-badge">{{ invitationsTotal }}</span>
          </div>
          <span class="pending-invitations-desc">
            {{ $t('tenantInvitation.pendingSectionDesc', { days: INVITATION_TTL_DAYS }) }}
          </span>
        </div>
        <div v-if="invitationsLoading" class="loading-inline">
          <t-loading size="small" />
          <span>{{ $t('tenantMember.loading') }}</span>
        </div>
        <div v-else-if="invitationsError" class="error-inline">
          <t-alert theme="error" :message="invitationsError">
            <template #operation>
              <t-button size="small" @click="loadInvitations">{{ $t('tenantMember.retry') }}</t-button>
            </template>
          </t-alert>
        </div>
        <div v-else-if="invitationsTotal === 0" class="pending-invitations-empty">
          {{ $t('tenantInvitation.pendingEmpty') }}
        </div>
        <div v-else class="data-table-shell data-table-shell--with-footer members-table-shell pending-invitations-table">
          <div class="data-table-shell__scroll">
            <t-table row-key="id" :data="invitations" :columns="invitationColumns" size="medium" hover>
              <template #invitee="{ row }">
                <div class="member-cell">
                  <span class="member-avatar" :class="{ 'member-avatar--link': row.is_share_link }">
                    <t-icon v-if="row.is_share_link" name="link" size="16px" />
                    <span v-else>{{ inviteeInitials(row) }}</span>
                  </span>
                  <div class="member-stack">
                    <template v-if="row.is_share_link">
                      <span class="member-name share-link-title"
                        :title="$t('tenantInvitation.shareLink.cellTitle')">
                        {{ $t('tenantInvitation.shareLink.cellTitle') }}
                      </span>
                      <span class="member-sub">
                        {{ (row.accepted_count ?? 0) > 0
                          ? $t('tenantInvitation.shareLink.cellAccepted', { count: row.accepted_count })
                          : $t('tenantInvitation.shareLink.cellEmpty') }}
                      </span>
                    </template>
                    <template v-else>
                      <span class="member-name" :title="inviteePrimary(row)">{{ inviteePrimary(row) }}</span>
                      <span v-if="row.invitee_email && row.invitee_name" class="member-sub"
                        :title="row.invitee_email">{{ row.invitee_email }}</span>
                    </template>
                  </div>
                </div>
              </template>
              <template #role="{ row }">
                <span class="role-pill" :class="`role-pill--${row.role}`">
                  {{ $t('tenantMember.role.' + row.role) }}
                </span>
              </template>
              <template #inviter="{ row }">
                <span class="member-email" :title="inviterPrimary(row)">{{ inviterPrimary(row) }}</span>
              </template>
              <template #expires_at="{ row }">
                <span class="member-date">{{ formatDate(row.expires_at) }}</span>
              </template>
              <template #status="{ row }">
                <span class="role-pill" :class="`status-pill--${row.status}`">
                  {{ row.is_share_link && row.status === 'pending'
                    ? $t('tenantInvitation.status.shareLinkActive')
                    : $t('tenantInvitation.status.' + row.status) }}
                </span>
              </template>
              <template #actions="{ row }">
                <!-- Per-row "copy link" for active share-link rows.
                     Icon-only with tooltip so two actions ("copy" +
                     "revoke") fit inside the actions column without
                     clipping; the full label was too wide. -->
                <t-tooltip v-if="row.status === 'pending' && row.invite_url"
                  :content="$t('tenantInvitation.copyLink')" placement="top">
                  <t-button class="member-action-btn member-action-btn--neutral" shape="square" variant="text"
                    size="small" @click="copyText(absoluteInviteURL(row.invite_url))">
                    <template #icon><t-icon name="copy" /></template>
                  </t-button>
                </t-tooltip>
                <!-- Inline popconfirm anchored to the revoke button.
                       Avoids spawning a top-level modal for a simple
                       yes/no decision; the popover stays inside the
                       table cell so the user keeps spatial context. -->
                <t-popconfirm v-if="row.status === 'pending'" theme="warning"
                  :content="row.is_share_link
                    ? $t('tenantInvitation.shareLink.revokeConfirm')
                    : $t('tenantInvitation.revoke.confirmBody', {
                        email: row.invitee_email || row.invitee_user_id,
                      })"
                  :confirm-btn="{ content: $t('tenantInvitation.revoke.confirm'), theme: 'danger' }"
                  :cancel-btn="$t('common.cancel')" placement="left" @confirm="doRevokeInvitation(row)">
                  <t-tooltip :content="$t('tenantInvitation.revoke.button')" placement="top">
                    <t-button class="member-action-btn" theme="default" shape="square" variant="text" size="small">
                      <template #icon><t-icon name="close" /></template>
                    </t-button>
                  </t-tooltip>
                </t-popconfirm>
              </template>
            </t-table>
          </div>
          <div v-if="invitationsTotal > 0" class="data-table-shell__pager">
            <span class="members-pager-info">
              {{ $t('tenantMember.pageIndicator', { page: invitationsPage, total: invitationsPageCount }) }}
            </span>
            <t-pagination v-model="invitationsPage" v-model:page-size="invitationsPageSize" :total="invitationsTotal"
              size="small" :total-content="false" show-page-number show-page-size
              :page-size-options="INVITATIONS_PAGE_SIZE_OPTIONS" @change="onInvitationsPageChange" />
          </div>
        </div>
      </div>

      <!-- Member list. Liste başlığı (başlık / sayaç / arama kutusu / davet düğmesi) her zaman
           render edilir; loading / error / empty / tablo ise aşağıdaki içerik durumları olarak değişir.
           Böylece arama sırasında giriş kutusu kaldırılmaz; odak kaybı ve sayfa titremesi önlenir. -->
      <div class="members-list-wrap">
        <div class="members-list-header">
          <div class="members-list-titlewrap">
            <span class="members-list-title">{{ $t('tenantMember.listTitle') }}</span>
            <span class="members-list-count-badge">{{ membersTotal }}</span>
          </div>
          <div class="members-list-actions">
            <div class="members-list-search">
              <t-input v-model="searchQuery" size="small" :placeholder="$t('tenantMember.searchPlaceholder')" clearable>
                <template #prefix-icon><t-icon name="search" /></template>
              </t-input>
            </div>
            <t-popup v-if="canManage" v-model="invitePopupVisible" trigger="click" placement="bottom-end"
              destroy-on-close overlay-class-name="wk-popover wk-popover--form member-invite-popup-overlay">
              <t-button theme="primary" variant="outline" shape="square" size="small" class="members-list-add-btn"
                :title="$t('tenantMember.add.button')" :aria-label="$t('tenantMember.add.button')">
                <template #icon><t-icon name="user-add" /></template>
              </t-button>
              <template #content>
                <div class="member-invite-popup-inner" @click.stop>
                  <div class="member-invite-popup-title">
                    {{
                      addDialogStep === 'form'
                        ? $t('tenantMember.add.dialogTitle')
                        : $t('tenantInvitation.confirmInviteTitle')
                    }}
                  </div>
                  <t-form v-if="addDialogStep === 'form'" ref="addFormRef" :data="addForm" :rules="addFormRules"
                    :label-width="80" label-align="left" class="member-invite-form">
                    <t-form-item :label="$t('tenantMember.add.emailLabel')" name="email">
                      <t-input v-model="addForm.email" :placeholder="$t('tenantMember.add.emailPlaceholder')"
                        clearable />
                    </t-form-item>
                    <t-form-item :label="$t('tenantMember.add.roleLabel')" name="role">
                      <t-select v-model="addForm.role" :options="roleOptions" :popup-props="roleSelectPopupProps" />
                    </t-form-item>
                  </t-form>
                  <div v-else class="invite-confirm-body">
                    {{ $t('tenantInvitation.confirmInviteBody', {
                      email: addConfirmEmail,
                      role: addConfirmRoleLabel,
                    }) }}
                  </div>
                  <div class="invite-popup-footer">
                    <t-button v-if="addDialogStep === 'form'" variant="outline" :disabled="adding"
                      @click="invitePopupVisible = false">
                      {{ $t('common.cancel') }}
                    </t-button>
                    <t-button v-else variant="outline" :disabled="adding" @click="goBackToForm">
                      {{ $t('common.back') }}
                    </t-button>
                    <t-button theme="primary" :loading="adding" @click="submitAdd">
                      {{ dialogConfirmLabel }}
                    </t-button>
                  </div>
                </div>
              </template>
            </t-popup>
            <!-- Share-link generator. Sits next to the invite-by-email
                 popup so the two flows live side-by-side: "I know who"
                 (email input) vs "I don't" (one link, group chat). -->
            <t-popup v-if="canManage" v-model="shareLinkPopupVisible" trigger="click" placement="bottom-end"
              destroy-on-close overlay-class-name="wk-popover wk-popover--form member-invite-popup-overlay">
              <t-button theme="default" variant="outline" shape="square" size="small" class="members-list-add-btn"
                :title="$t('tenantInvitation.shareLink.button')"
                :aria-label="$t('tenantInvitation.shareLink.button')">
                <template #icon><t-icon name="link" /></template>
              </t-button>
              <template #content>
                <div class="member-invite-popup-inner" @click.stop>
                  <div class="member-invite-popup-title">
                    {{
                      shareLinkResult
                        ? $t('tenantInvitation.shareLink.resultTitle')
                        : $t('tenantInvitation.shareLink.dialogTitle')
                    }}
                  </div>
                  <div v-if="!shareLinkResult" class="member-invite-form">
                    <p class="invite-confirm-body">
                      {{ $t('tenantInvitation.shareLink.description', { days: INVITATION_TTL_DAYS }) }}
                    </p>
                    <t-form :data="shareLinkForm" :label-width="80" label-align="left">
                      <t-form-item :label="$t('tenantMember.add.roleLabel')" name="role">
                        <t-select v-model="shareLinkForm.role" :options="roleOptions"
                          :popup-props="roleSelectPopupProps" />
                      </t-form-item>
                    </t-form>
                  </div>
                  <div v-else class="share-link-result">
                    <p class="invite-confirm-body">
                      {{ $t('tenantInvitation.shareLink.resultBody') }}
                    </p>
                    <div class="share-link-row">
                      <input class="share-link-row__input"
                        :value="absoluteInviteURL(shareLinkResult.invite_url || '')"
                        readonly @click="($event.target as HTMLInputElement).select()" />
                      <t-button size="small" theme="primary" variant="outline"
                        @click="copyText(absoluteInviteURL(shareLinkResult.invite_url || ''))">
                        <template #icon><t-icon name="copy" /></template>
                        {{ $t('tenantInvitation.copyLink') }}
                      </t-button>
                    </div>
                  </div>
                  <div class="invite-popup-footer">
                    <t-button v-if="!shareLinkResult" variant="outline" :disabled="creatingShareLink"
                      @click="shareLinkPopupVisible = false">
                      {{ $t('common.cancel') }}
                    </t-button>
                    <t-button v-else variant="outline" @click="shareLinkPopupVisible = false">
                      {{ $t('common.close') }}
                    </t-button>
                    <t-button v-if="!shareLinkResult" theme="primary" :loading="creatingShareLink"
                      @click="submitShareLink">
                      {{ $t('tenantInvitation.shareLink.generate') }}
                    </t-button>
                  </div>
                </div>
              </template>
            </t-popup>
          </div>
        </div>
        <div v-if="loading && members.length === 0" class="members-table-skeleton" :aria-label="$t('tenantMember.loading')"
          role="status">
          <span v-for="i in 3" :key="i" class="members-table-skeleton__bar" />
        </div>
        <div v-else-if="error" class="error-inline">
          <t-alert theme="error" :message="error">
            <template #operation>
              <t-button size="small" @click="loadMembers">{{ $t('tenantMember.retry') }}</t-button>
            </template>
          </t-alert>
        </div>
        <div v-else-if="membersTotal === 0" class="members-table-shell members-table-empty">
          <span class="members-table-empty__icon"><t-icon name="usergroup" /></span>
          <span class="members-table-empty__text">
            {{ searchQuery.trim()
              ? $t('tenantMember.emptySearch', { q: searchQuery })
              : $t('tenantMember.empty') }}
          </span>
        </div>
        <div v-else class="data-table-shell data-table-shell--with-footer members-table-shell">
          <div class="data-table-shell__scroll">
            <t-table row-key="user_id" :data="members" :columns="columns" size="medium" hover :loading="loading">
              <template #member="{ row }">
                <div class="member-cell">
                  <span class="member-avatar">
                    <img v-if="row.avatar" :src="row.avatar" alt="" />
                    <span v-else>{{ memberInitials(row) }}</span>
                  </span>
                  <span class="member-name">{{ memberPrimary(row) }}</span>
                  <span v-if="row.user_id === currentUserId" class="member-you">({{ $t('tenantMember.you') }})</span>
                </div>
              </template>
              <template #email="{ row }">
                <span class="member-email">{{ row.email || '—' }}</span>
              </template>
              <template #role="{ row }">
                <div class="role-cell">
                  <t-select v-if="canManage && row.user_id !== currentUserId" :model-value="row.role"
                    class="member-role-select" :class="`role-pill--${row.role}`" size="small"
                    :popup-props="roleSelectPopupProps"
                    @change="(val: string) => onRoleChange(row, val)">
                    <t-option v-for="opt in roleOptions" :key="opt.value" :value="opt.value" :label="opt.label">
                      <span class="role-option">
                        <t-icon :name="roleIcon(opt.value)" class="role-option-icon" />
                        <span>{{ opt.label }}</span>
                      </span>
                    </t-option>
                  </t-select>
                  <span v-else class="role-pill" :class="`role-pill--${row.role}`">
                    {{ $t('tenantMember.role.' + row.role) }}
                  </span>
                </div>
              </template>
              <template #joined_at="{ row }">
                <span class="member-date">{{ formatDay(row.joined_at) }}</span>
              </template>
              <template #actions="{ row }">
                <t-popconfirm
                  v-if="canManage && row.user_id !== currentUserId"
                  :content="$t('tenantMember.remove.confirmBody', { name: row.username || row.email })"
                  :confirm-btn="{ content: $t('tenantMember.remove.confirm'), theme: 'danger' }"
                  :cancel-btn="{ content: $t('common.cancel') }"
                  placement="left"
                  @confirm="removeRow(row)">
                  <t-tooltip :content="$t('tenantMember.remove.button')" placement="top">
                    <t-button class="member-action-btn" theme="default" shape="square" variant="text" size="small"
                      @click.stop>
                      <template #icon><t-icon name="user-clear" /></template>
                    </t-button>
                  </t-tooltip>
                </t-popconfirm>
              </template>
            </t-table>
          </div>
          <div v-if="membersTotal > 0" class="data-table-shell__pager">
            <span class="members-pager-info">
              {{ $t('tenantMember.pageIndicator', { page: membersPage, total: membersPageCount }) }}
            </span>
            <t-pagination v-model="membersPage" v-model:page-size="membersPageSize" :total="membersTotal" size="small"
              :total-content="false" show-page-number show-page-size :page-size-options="MEMBERS_PAGE_SIZE_OPTIONS"
              @change="onMembersPageChange" />
          </div>
        </div>
      </div>

    </div>

    <!-- Audit log drawer. Only rendered for Admin+ because the backend
         route is g.Admin()-gated; rendering it for lower roles would
         just produce an unhelpful 403. Lazy-loaded on first open. -->
    <SettingDrawer v-if="canViewAudit" v-model:visible="auditDrawerVisible"
      :title="$t('tenantMember.audit.tabLabel')" width="1120px" :min-width="720" :max-width="1600"
      storage-key="setting-drawer:width:tenant-members-audit" :hide-footer="true"
      drawer-class-name="tenant-members-audit-drawer">
      <div class="audit-drawer-inner audit-panel audit-panel--drawer">
        <div class="audit-header">
          <span class="audit-desc">{{ $t('tenantMember.audit.description') }}</span>
          <t-button variant="text" size="small" class="audit-refresh-btn"
            :loading="auditLoading" :disabled="auditLoading" @click="reloadAuditLog">
            <template #icon><t-icon name="refresh" /></template>
            {{ $t('tenantMember.audit.refresh') }}
          </t-button>
        </div>

        <div class="audit-drawer-fill">
          <div v-if="auditError" class="audit-drawer-branch audit-drawer-branch--error">
            <div class="error-inline">
              <t-alert theme="error" :message="auditError">
                <template #operation>
                  <t-button size="small" @click="reloadAuditLog">
                    {{ $t('tenantMember.retry') }}
                  </t-button>
                </template>
              </t-alert>
            </div>
          </div>

          <div v-else-if="!auditLoading && auditEntries.length === 0"
            class="audit-drawer-branch audit-drawer-branch--empty empty-state empty-state--audit">
            <t-empty :description="$t('tenantMember.audit.empty')" />
          </div>

          <div v-else class="audit-scroll-area narrow-scrollbar audit-drawer-branch" ref="auditScrollRoot">
            <div class="data-table-shell audit-table-shell">
              <t-table
                row-key="id"
                :data="auditEntries"
                :columns="auditColumns"
                size="medium"
                hover
                expand-on-row-click
                :expanded-row-keys="auditExpandedRowKeys"
                @expand-change="onAuditExpandChange"
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
                      {{ row.actor_user_id ? actorDisplayName(row.actor_user_id) :
                        $t('tenantMember.audit.systemActor') }}
                    </span>
                    <span v-if="row.actor_role" class="audit-actor-role">
                      {{ $t('tenantMember.role.' + row.actor_role) }}
                    </span>
                  </div>
                </template>
                <template #action="{ row }">
                  <t-tag :theme="auditActionTheme(row.action)" size="small" variant="light-outline">
                    {{ formatAuditAction(row.action) }}
                  </t-tag>
                </template>
                <template #target="{ row }">
                  <div class="audit-target">
                    <span v-if="auditTargetSubject(row)" class="audit-target-key">{{ auditTargetSubject(row) }}</span>
                    <span v-if="auditTargetDiff(row)" class="audit-target-diff">{{ auditTargetDiff(row) }}</span>
                    <span v-else-if="!auditTargetSubject(row)" class="audit-target-empty">—</span>
                  </div>
                </template>
                <template #request_path="{ row }">
                  <span v-if="row.request_path" class="audit-path">
                    <span v-if="row.request_method" class="audit-method">{{ row.request_method }}</span>
                    {{ row.request_path }}
                  </span>
                  <span v-else class="audit-target-empty">—</span>
                </template>
                <template #outcome="{ row }">
                  <t-tag :theme="auditOutcomeTheme(row.outcome)" size="small" variant="light">
                    {{ $t('tenantMember.audit.outcome.' + row.outcome) }}
                  </t-tag>
                </template>
                <template #expandedRow="{ row }">
                  <div class="audit-expanded">
                    <div class="audit-expanded-grid">
                      <div class="audit-expanded-cell">
                        <span class="audit-expanded-label">{{ $t('tenantMember.audit.expanded.actorId') }}</span>
                        <span class="audit-expanded-value mono">{{ row.actor_user_id || '—' }}</span>
                      </div>
                      <div v-if="row.target_user_id" class="audit-expanded-cell">
                        <span class="audit-expanded-label">{{ $t('tenantMember.audit.expanded.targetUserId') }}</span>
                        <span class="audit-expanded-value mono">{{ row.target_user_id }}</span>
                      </div>
                      <div v-if="row.target_type" class="audit-expanded-cell">
                        <span class="audit-expanded-label">{{ $t('tenantMember.audit.expanded.targetType') }}</span>
                        <span class="audit-expanded-value mono">{{ row.target_type }}</span>
                      </div>
                      <div v-if="row.target_id" class="audit-expanded-cell">
                        <span class="audit-expanded-label">{{ $t('tenantMember.audit.expanded.targetId') }}</span>
                        <span class="audit-expanded-value mono">{{ row.target_id }}</span>
                      </div>
                    </div>
                    <div class="audit-expanded-details">
                      <span class="audit-expanded-label">{{ $t('tenantMember.audit.expanded.details') }}</span>
                      <pre class="audit-expanded-json mono">{{ auditDetailsJSON(row) }}</pre>
                    </div>
                  </div>
                </template>
              </t-table>
            </div>

            <!-- Alt sınıra ulaşma sentinel'i: IntersectionObserver root, audit-scroll-area'yı işaret eder -->
            <div ref="auditLoadSentinelEl" class="audit-load-sentinel" aria-hidden="true" />

            <div v-if="auditLoading && auditEntries.length > 0" class="audit-loading-more">
              <t-loading size="small" />
              <span>{{ $t('tenantMember.loading') }}</span>
            </div>

            <p v-if="!auditHasMore && auditEntries.length > 0 && !auditLoading" class="audit-end-hint">
              {{ $t('tenantMember.audit.end') }}
            </p>
          </div>
        </div>
      </div>
    </SettingDrawer>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onUnmounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { MessagePlugin } from 'tdesign-vue-next'
import SettingDrawer from '@/components/settings/SettingDrawer.vue'
import TenantRolePermissionsPopover from '@/components/settings/TenantRolePermissionsPopover.vue'
import { copyWithToast } from '@/utils/clipboard'
import { useAuthStore } from '@/stores/auth'
import { AUDIT_ACTION_I18N_ROOTS } from '@/i18n/auditActionRegistry'
import { auditActionLabel } from '@/i18n/auditActionLabel'
import {
  listMembers,
  updateMemberRole,
  removeMember,
  type TenantMember,
  type TenantRole,
} from '@/api/tenant/members'
import {
  listTenantInvitations,
  createInvitation,
  createInviteLink,
  revokeInvitation,
  type TenantInvitation,
} from '@/api/tenant/invitations'
import {
  listAuditLog,
  type AuditLog,
  type AuditAction,
  type AuditOutcome,
} from '@/api/tenant/audit-log'

const { t, tm, locale } = useI18n()
const authStore = useAuthStore()

// State
const members = ref<TenantMember[]>([])
const loading = ref(false)
const error = ref('')
const adding = ref(false)
/** Davet akışı: liste başlığındaki 「+」 düğmesinin yanına sabitlenmiş bir açılır katman (ortalanmış modal değil). */
const invitePopupVisible = ref(false)
// share-link generator state (separate popup next to the email
// invite). shareLinkResult is non-null after a successful create —
// the popup then switches into "here's your link, copy it" mode.
const shareLinkPopupVisible = ref(false)
const shareLinkForm = reactive<{ role: TenantRole }>({ role: 'contributor' })
const creatingShareLink = ref(false)
const shareLinkResult = ref<TenantInvitation | null>(null)
// Two-step invite inside the popup: 'form' renders the email/role inputs;
// 'confirm' swaps the body for an in-place summary; primary CTA toggles label.
const addDialogStep = ref<'form' | 'confirm'>('form')
const addFormRef = ref<any>(null)
const searchQuery = ref('')
/** Sunucu tarafı filtreye uygulanmış arama terimi (giriş kutusu debounce'una göre) */
const memberSearchQ = ref('')
let memberSearchDebounceTimer: number | undefined

const membersTotal = ref(0)
const membersPage = ref(1)
const membersPageSize = ref(20)

const invitationsTotal = ref(0)
const invitationsPage = ref(1)
const invitationsPageSize = ref(20)

/** Önceki sayfalama yüklerinde görülen üye görüntüleme alanları, denetim tablosunda mevcut sayfada olmayan user id'leri tamamlar */
const memberDisplayByUserId = reactive<Record<string, { username?: string; email?: string }>>({})

// Pending invitations live alongside members but in a distinct section
// at the top of the Members tab — they're "people we've asked to
// join but haven't yet accepted", and conflating them with the
// authoritative roster would mislead an Owner trying to see who
// actually has access. The load happens on the same trigger as the
// members fetch so the screen renders both at once.
const invitations = ref<TenantInvitation[]>([])
const invitationsLoading = ref(false)
const invitationsError = ref('')
// Invitation TTL is mirrored from the backend constant
// (defaultInvitationTTL in tenant_invitation.go). Kept as a UI string
// for the section description; the authoritative number comes from
// the server's expires_at on each row.
const INVITATION_TTL_DAYS = 7

const MEMBERS_PAGE_SIZE_OPTIONS = [10, 20, 50, 100]
const INVITATIONS_PAGE_SIZE_OPTIONS = [10, 20, 50, 100]

// Audit log moved out of t-tabs into a right-side drawer; this flag
// controls its visibility. Default closed — most operators come here
// to manage members, not to audit. Lazy-load logic is wired through
// openAuditDrawer() rather than a watch() on the visibility flag so a
// re-open of the drawer doesn't re-trigger a fetch the user didn't ask
// for (they have an explicit "refresh" button inside the drawer).
const auditDrawerVisible = ref(false)

// Audit-log state. Backend cursor-paged by descending id (`after_id`);
// frontend appends rows when the sentinel scrolls into view. When
// `next_cursor` is 0, `auditHasMore` becomes false and loading stops.
const auditEntries = ref<AuditLog[]>([])
const auditLoading = ref(false)
const auditError = ref('')
const auditCursor = ref<number>(0) // 0 = "from the top"
const auditHasMore = ref(true)
const auditLoadedOnce = ref(false)
const AUDIT_PAGE_SIZE = 50

/** Çekmece içindeki kaydırma kökü ve alt sınıra ulaşma sentinel'i; imleç tabanlı sayfalama ile sonraki sayfayı otomatik yüklemek için kullanılır (bkz. attachAuditInfiniteScroll) */
const auditScrollRoot = ref<HTMLElement | null>(null)
const auditLoadSentinelEl = ref<HTMLElement | null>(null)
let auditScrollObserver: IntersectionObserver | null = null

// Add dialog model — reset on each open. Default role is contributor:
// inviting a fresh member with viewer is too restrictive for the
// expected "let them collaborate on KBs" use case, and admin/owner
// should be a deliberate promote step after the user accepts.
const addForm = reactive<{ email: string; role: TenantRole }>({
  email: '',
  role: 'contributor',
})

// Role-aware gates. The server enforces every mutation; UI gates here
// are presentational only, matching the security note in stores/auth.ts.
const currentRole = computed<TenantRole | ''>(() => (authStore.currentTenantRole || '') as TenantRole | '')
// Cross-tenant superusers (org-level operators) bypass the Owner gate
// on the server (see middleware/rbac.go RequireRole). The UI must
// mirror that or the buttons would be invisible to the exact admins
// who actually need them. Local Owners of their own tenant come in via
// the role branch.
const canManage = computed(
  () => currentRole.value === 'owner' || authStore.canAccessAllTenants === true,
)
// Admin+ (and cross-tenant superusers) can view the audit log. Mirrors
// the server's g.Admin() guard on /tenants/:id/audit-log so we don't
// render a tab that would just 403.
const canViewAudit = computed(
  () =>
    currentRole.value === 'owner' ||
    currentRole.value === 'admin' ||
    authStore.canAccessAllTenants === true,
)
const currentUserId = computed(() => authStore.user?.id ?? '')

// Use the active tenant id from the auth store; the route only allows
// :id == active tenant (auth middleware enforces membership), so we
// don't expose a tenant picker here.
const activeTenantId = computed(() => Number(authStore.currentTenantId ?? 0))

const roleOptions = computed(() => [
  { label: t('tenantMember.role.owner'), value: 'owner' },
  { label: t('tenantMember.role.admin'), value: 'admin' },
  { label: t('tenantMember.role.contributor'), value: 'contributor' },
  { label: t('tenantMember.role.viewer'), value: 'viewer' },
])

/** Açılır katman, davet açılır katmanından (3050) ve kuruluş ayarları tam ekran maskesinden daha yukarıda olmalıdır; aksi hâlde altında kalır */
const roleSelectPopupProps = {
  zIndex: 6200,
  overlayClassName: 'tenant-members-role-select-popup',
}

function roleMatrixIcon(role: TenantRole): string {
  switch (role) {
    case 'owner':
      return 'user-vip-filled'
    case 'admin':
      return 'user-safety'
    case 'contributor':
      return 'edit'
    default:
      return 'browse'
  }
}

const columns = computed(() => [
  { colKey: 'member', title: t('tenantMember.columns.name'), ellipsis: true, minWidth: 200 },
  { colKey: 'email', title: t('tenantMember.columns.email'), ellipsis: true, minWidth: 200 },
  { colKey: 'role', title: t('tenantMember.columns.role'), width: 176 },
  { colKey: 'joined_at', title: t('tenantMember.columns.joinedAt'), width: 132 },
  { colKey: 'actions', title: t('tenantMember.columns.operations'), width: 88, align: 'center' },
])

const membersPageCount = computed(() =>
  Math.max(1, Math.ceil(membersTotal.value / (membersPageSize.value || 1))),
)

function memberInitials(row: { username?: string; email?: string }) {
  const source = row.username?.trim() || row.email?.split('@')[0]?.trim() || ''
  if (!source) return '?'
  const parts = source.split(/[\s._-]+/).filter(Boolean)
  const letters = parts.length > 1 ? parts[0][0] + parts[1][0] : source.slice(0, 2)
  return letters.toLocaleUpperCase(locale.value || 'tr-TR')
}

function memberPrimary(row: { username?: string; email?: string }) {
  return row.username?.trim() || row.email?.trim() || '—'
}

const addFormRules = {
  email: [
    { required: true, message: t('tenantMember.errors.emailRequired'), trigger: 'blur' },
    { email: true, message: t('tenantMember.errors.emailFormat'), trigger: 'blur' },
  ],
  role: [{ required: true, message: t('tenantMember.errors.roleRequired'), trigger: 'change' }],
}

/** Üye tablosu/açılır liste ve izin matrisi simgeleri ortak kullanır (crown, tdesign-icons-vue-next içinde yoktur). */
function roleIcon(role: TenantRole | string): string {
  if (role === 'owner' || role === 'admin' || role === 'contributor' || role === 'viewer') {
    return roleMatrixIcon(role as TenantRole)
  }
  return 'user'
}

function formatDate(s: string | undefined): string {
  if (!s) return '-'
  try {
    const d = new Date(s)
    return new Intl.DateTimeFormat(locale.value || 'tr-TR', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    }).format(d)
  } catch {
    return s
  }
}

function formatDay(s: string | undefined): string {
  if (!s) return '—'
  const d = new Date(s)
  if (Number.isNaN(d.getTime())) return s
  return new Intl.DateTimeFormat(locale.value || 'tr-TR', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(d)
}

function rememberMembersForAudit(rows: TenantMember[]) {
  for (const m of rows) {
    memberDisplayByUserId[m.user_id] = { username: m.username, email: m.email }
  }
}

async function loadMembers() {
  if (!activeTenantId.value) {
    return
  }
  loading.value = true
  error.value = ''
  try {
    const resp = await listMembers(activeTenantId.value, {
      page: membersPage.value,
      page_size: membersPageSize.value,
      q: memberSearchQ.value || undefined,
    })
    if (resp.success && resp.data) {
      const total = resp.data.total ?? 0
      const ps = resp.data.page_size ?? membersPageSize.value
      const safePs = Math.max(1, ps)
      const maxPage = Math.max(1, Math.ceil(total / safePs))
      if (membersPage.value > maxPage) {
        membersPage.value = maxPage
        loading.value = false
        await loadMembers()
        return
      }
      members.value = resp.data.members ?? []
      membersTotal.value = total
      if (typeof resp.data.page === 'number' && resp.data.page > 0) {
        membersPage.value = resp.data.page
      }
      if (typeof resp.data.page_size === 'number' && resp.data.page_size > 0) {
        membersPageSize.value = resp.data.page_size
      }
      rememberMembersForAudit(members.value)
    } else {
      error.value = resp.message || t('tenantMember.errors.generic')
    }
  } catch (err: any) {
    error.value = err?.message || t('tenantMember.errors.generic')
  } finally {
    loading.value = false
  }
}

function onMembersPageChange() {
  void loadMembers()
}

watch(searchQuery, () => {
  if (!activeTenantId.value) return
  window.clearTimeout(memberSearchDebounceTimer)
  memberSearchDebounceTimer = window.setTimeout(() => {
    memberSearchQ.value = searchQuery.value.trim()
    membersPage.value = 1
    loadMembers()
  }, 320)
})

// ---- Pending invitations ------------------------------------------------

// Same shell/column rhythm as the members table. Pill cells (role,
// status) get fixed widths wide enough for the longest locale label so
// they never clip; text cells ellipsize with a title tooltip.
const invitationColumns = computed(() => [
  { colKey: 'invitee', title: t('tenantInvitation.columns.invitee'), minWidth: 240 },
  { colKey: 'role', title: t('tenantInvitation.columns.role'), width: 160 },
  { colKey: 'inviter', title: t('tenantInvitation.columns.inviter'), minWidth: 160 },
  { colKey: 'expires_at', title: t('tenantInvitation.columns.expiresAt'), width: 170 },
  { colKey: 'status', title: t('tenantInvitation.columns.status'), width: 140 },
  ...(canManage.value
    ? [{ colKey: 'actions', title: t('tenantInvitation.columns.operations'), width: 104, align: 'center' as const }]
    : []),
])

const invitationsPageCount = computed(() =>
  Math.max(1, Math.ceil(invitationsTotal.value / (invitationsPageSize.value || 1))),
)

function inviteePrimary(row: TenantInvitation): string {
  return row.invitee_name?.trim() || row.invitee_email?.trim() || row.invitee_user_id
}

function inviteeInitials(row: TenantInvitation): string {
  return memberInitials({ username: row.invitee_name, email: row.invitee_email })
}

function inviterPrimary(row: TenantInvitation): string {
  return row.inviter_name?.trim() || row.inviter_email?.trim() || row.invited_by || '—'
}

// loadInvitations is called from the same trigger as loadMembers so
// the Members tab can render the pending section without an extra
// round-trip latency budget. canManage gates the fetch — viewers /
// contributors / admins-without-management see no pending section at
// all (the route returns 200 but listing pending invites to those
// roles would be UX noise; the route layer is Viewer+ for the read
// itself).
async function loadInvitations() {
  if (!activeTenantId.value || !canManage.value) {
    invitations.value = []
    invitationsTotal.value = 0
    return
  }
  invitationsLoading.value = true
  invitationsError.value = ''
  try {
    const resp = await listTenantInvitations(activeTenantId.value, {
      page: invitationsPage.value,
      page_size: invitationsPageSize.value,
    })
    if (resp.success && resp.data) {
      const total = resp.data.total ?? 0
      const ps = resp.data.page_size ?? invitationsPageSize.value
      const safePs = Math.max(1, ps)
      const maxPage = Math.max(1, Math.ceil(total / safePs))
      if (invitationsPage.value > maxPage) {
        invitationsPage.value = maxPage
        invitationsLoading.value = false
        await loadInvitations()
        return
      }
      invitations.value = resp.data.invitations ?? []
      invitationsTotal.value = total
      if (typeof resp.data.page === 'number' && resp.data.page > 0) {
        invitationsPage.value = resp.data.page
      }
      if (typeof resp.data.page_size === 'number' && resp.data.page_size > 0) {
        invitationsPageSize.value = resp.data.page_size
      }
    } else {
      invitationsError.value = resp.message || t('tenantInvitation.errors.generic')
    }
  } catch (err: any) {
    invitationsError.value = err?.message || t('tenantInvitation.errors.generic')
  } finally {
    invitationsLoading.value = false
  }
}

function onInvitationsPageChange() {
  void loadInvitations()
}

// doRevokeInvitation is wired to the t-popconfirm @confirm event in
// the table template. The popconfirm itself owns the yes/no surface,
// so this function is the post-confirmation action only — no nested
// modal. Errors surface as toasts and the row stays in place for retry.
async function doRevokeInvitation(row: TenantInvitation) {
  try {
    const resp = await revokeInvitation(activeTenantId.value, row.id)
    if (resp.success) {
      await loadInvitations()
      MessagePlugin.success(t('tenantInvitation.revoke.success'))
    } else {
      MessagePlugin.error(resp.message || t('tenantInvitation.errors.generic'))
    }
  } catch (err: any) {
    const status = err?.status
    if (status === 404) {
      MessagePlugin.error(t('tenantInvitation.errors.notFound'))
    } else if (status === 409) {
      MessagePlugin.error(err?.message || t('tenantInvitation.errors.notPending'))
    } else {
      MessagePlugin.error(err?.message || t('tenantInvitation.errors.generic'))
    }
  }
}

// ---- Audit-log helpers --------------------------------------------------

// Stacked "date / time" cell — mirrors SystemSettings audit table. When
// 50 events fall in the same minute, ellipsing a flat string makes them
// indistinguishable; splitting on two lines keeps the seconds visible
// without eating horizontal budget the diff column needs.

const auditColumns = computed(() => [
  { colKey: 'created_at', title: t('tenantMember.audit.columns.time'), width: 120 },
  { colKey: 'actor', title: t('tenantMember.audit.columns.actor'), width: 180 },
  { colKey: 'action', title: t('tenantMember.audit.columns.action'), width: 210 },
  {
    colKey: 'target',
    title: t('tenantMember.audit.columns.target'),
    // No fixed width / no ellipsis: this is where the role-diff and
    // denied-action context live. Wrap rather than clip — losing the
    // "Owner → Admin" half of a role change defeats the point.
    minWidth: 200,
  },
  {
    colKey: 'request_path',
    title: t('tenantMember.audit.columns.path'),
    minWidth: 160,
  },
  { colKey: 'outcome', title: t('tenantMember.audit.columns.outcome'), width: 80, align: 'center' as const },
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

// Action chip colour: rejection events are loud (danger) so an
// operator can scan a chronological feed and immediately spot abuse;
// member adds are reassuring green; removals/role changes warning
// orange because they're worth a second look but aren't intrinsically
// suspicious.
function auditActionTheme(
  action: AuditAction,
): 'success' | 'warning' | 'danger' | 'primary' | 'default' {
  switch (action) {
    case 'rbac.access_denied':
      return 'danger'
    case 'rbac.member_added':
      return 'success'
    case 'rbac.member_removed':
    case 'rbac.member_left':
    case 'rbac.member_role_changed':
      return 'warning'
    default:
      return 'default'
  }
}

function auditOutcomeTheme(o: AuditOutcome): 'success' | 'danger' | 'default' {
  if (o === 'denied') return 'danger'
  if (o === 'success') return 'success'
  return 'default'
}

// i18n anahtar adı nokta içerir (rbac.member_added). t(path) kullanımı bunu yol olarak bölüp ayrıştırır,
// tenantMember.audit.action['rbac.*'] eşleşemez — tm + değişmez anahtar kullanılmalıdır.
function formatAuditAction(action: AuditAction): string {
  return auditActionLabel({ tm }, AUDIT_ACTION_I18N_ROOTS.tenantMember, action)
}

// Resolve a user id to a display label: prefer current sayfadaki members,
// ardından önceki sayfalardan biriken memberDisplayByUserId'ye dön, son olarak ham id'yi kullan.
function actorDisplayName(userId: string): string {
  const cur = members.value.find((x) => x.user_id === userId)
  if (cur?.username?.trim()) return cur.username.trim()
  if (cur?.email?.trim()) return cur.email.trim()
  const memo = memberDisplayByUserId[userId]
  if (memo?.username?.trim()) return memo.username!.trim()
  if (memo?.email?.trim()) return memo.email!.trim()
  return userId
}

// Split target rendering into a "subject" (who was acted on) and a
// "diff" (what changed). The cell template stacks them with the
// subject on the first line, diff on the second. Returns '' when a
// piece is unavailable so the v-if branches drop the wrapper cleanly.

function auditDetailsObject(row: AuditLog): Record<string, unknown> | null {
  if (row.details && typeof row.details === 'object') {
    return row.details as Record<string, unknown>
  }
  return null
}

function auditTargetSubject(row: AuditLog): string {
  if (row.target_user_id) return actorDisplayName(row.target_user_id)
  if (row.target_id) {
    return row.target_type ? `${row.target_type}:${row.target_id}` : row.target_id
  }
  return ''
}

function auditTargetDiff(row: AuditLog): string {
  const d = auditDetailsObject(row)
  if (!d) return ''
  if (row.action === 'rbac.member_role_changed') {
    if (d.old_role && d.new_role) return `${d.old_role} → ${d.new_role}`
  }
  if (row.action === 'rbac.access_denied') {
    if (typeof d.required_role === 'string') {
      return t('tenantMember.audit.requiredRole', { role: d.required_role })
    }
  }
  if (row.action === 'rbac.invitation_sent' || row.action === 'rbac.invitation_revoked') {
    if (typeof d.role === 'string') return String(d.role)
  }
  return ''
}

// Expanded row state — local set of ids the user has opened. We keep
// it ephemeral (not persisted) so reopening the drawer always starts
// in the collapsed view.
const auditExpandedRowKeys = ref<number[]>([])

function onAuditExpandChange(value: (string | number)[]) {
  auditExpandedRowKeys.value = value
    .map((v) => (typeof v === 'number' ? v : Number(v)))
    .filter((v) => Number.isFinite(v))
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

// loadAuditLog fetches a page. `reset=true` discards the current
// list and starts from cursor=0. Used by the refresh button and the
// initial tab-switch trigger.
async function loadAuditLog(reset: boolean) {
  if (!activeTenantId.value || !canViewAudit.value) return
  if (auditLoading.value) return
  if (!reset && !auditHasMore.value) return

  auditLoading.value = true
  auditError.value = ''
  try {
    const resp = await listAuditLog(activeTenantId.value, {
      after_id: reset ? undefined : auditCursor.value || undefined,
      limit: AUDIT_PAGE_SIZE,
    })
    if (resp.success) {
      const rows = resp.data || []
      if (reset) {
        auditEntries.value = rows
      } else {
        auditEntries.value = [...auditEntries.value, ...rows]
      }
      auditCursor.value = resp.next_cursor || 0
      // The server returns next_cursor=0 when the page is empty OR
      // when the last row is the smallest possible id. Both mean
      // "stop paginating".
      auditHasMore.value = !!resp.next_cursor && rows.length > 0
      auditLoadedOnce.value = true
    } else {
      auditError.value = resp.message || t('tenantMember.errors.generic')
    }
  } catch (err: any) {
    const status = err?.status
    if (status === 403) {
      auditError.value = t('tenantMember.audit.forbidden')
    } else {
      auditError.value = err?.message || t('tenantMember.errors.generic')
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
  loadAuditLog(true)
}

// Lazy-load the audit log the first time the drawer is opened. We
// avoid watching `auditDrawerVisible` so closing+reopening the drawer
// doesn't re-fetch behind the user's back — refresh is an explicit
// action via the drawer's "Refresh" button.
function openAuditDrawer() {
  auditDrawerVisible.value = true
  if (!auditLoadedOnce.value) {
    loadAuditLog(true)
  }
}

watch(
  auditDrawerVisible,
  async (open) => {
    if (!open) {
      detachAuditInfiniteScroll()
      return
    }
    await nextTick()
    attachAuditInfiniteScroll()
  },
  { flush: 'post' },
)

watch(
  () => auditError.value,
  async () => {
    if (!auditDrawerVisible.value) return
    await nextTick()
    if (!auditError.value) {
      attachAuditInfiniteScroll()
      return
    }
    detachAuditInfiniteScroll()
  },
  { flush: 'post' },
)

onUnmounted(() => detachAuditInfiniteScroll())

watch(invitePopupVisible, (open) => {
  if (!open) return
  addForm.email = ''
  addForm.role = 'contributor'
  addDialogStep.value = 'form'
})

// Share-link popup: re-init on every open so the operator never sees
// the previous result on a fresh click.
watch(shareLinkPopupVisible, (open) => {
  if (!open) return
  shareLinkForm.role = 'contributor'
  shareLinkResult.value = null
})

// absoluteInviteURL turns the backend's potentially-host-relative
// invite_url into a copy-friendly absolute URL. The backend returns
// "/register?token=…" when FRONTEND_BASE_URL is unset (the typical
// case); the SPA is best-positioned to know its own origin.
function absoluteInviteURL(raw: string): string {
  if (!raw) return ''
  if (/^https?:\/\//i.test(raw)) return raw
  const origin = (typeof window !== 'undefined' && window.location && window.location.origin) || ''
  return raw.startsWith('/') ? origin + raw : origin + '/' + raw
}

async function copyText(text: string) {
  await copyWithToast(text, 'tenantInvitation.copied', 'tenantInvitation.copyFailed')
}

async function submitShareLink() {
  creatingShareLink.value = true
  try {
    const resp = await createInviteLink(activeTenantId.value, { role: shareLinkForm.role })
    if (!resp.success || !resp.data) {
      MessagePlugin.error(resp.message || t('tenantInvitation.errors.generic'))
      return
    }
    shareLinkResult.value = resp.data
    invitationsPage.value = 1
    await loadInvitations()
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('tenantInvitation.errors.generic'))
  } finally {
    creatingShareLink.value = false
  }
}

// Live display strings for the in-place confirm step. Recomputed
// every time the user goes Back, tweaks the form, and re-advances —
// the summary always mirrors the current form state.
const addConfirmEmail = computed(() => addForm.email.trim())
const addConfirmRoleLabel = computed(() => t('tenantMember.role.' + addForm.role))

// submitAdd is wired to the popup footer primary CTA. On step='form' it
// validates and swaps to summary; on step='confirm' it fires the API.
// With auto-accept the action is a direct add, so we skip the invitation
// confirm step entirely and fire the API right after validation.
async function submitAdd() {
  if (addDialogStep.value === 'form') {
    const valid = await addFormRef.value?.validate?.()
    if (valid !== true) return
    if (authStore.autoAcceptInvitation) {
      await sendInvitation(addForm.email.trim(), addForm.role)
      return
    }
    addDialogStep.value = 'confirm'
    return
  }
  await sendInvitation(addForm.email.trim(), addForm.role)
}

// goBackToForm un-advances from confirm to form inside the popup.
function goBackToForm() {
  addDialogStep.value = 'form'
}

// dialogConfirmLabel: "Send" on the confirm step, or when auto-accept makes
// the form step a direct add; otherwise "Send invitation".
const dialogConfirmLabel = computed(() =>
  addDialogStep.value === 'confirm' || authStore.autoAcceptInvitation
    ? t('tenantInvitation.confirmSend')
    : t('tenantInvitation.inviteSubmit'),
)

// sendInvitation actually fires the create-invitation API call.
async function sendInvitation(email: string, role: TenantRole) {
  adding.value = true
  try {
    const resp = await createInvitation(activeTenantId.value, { email, role })
    if (resp.success) {
      // auto-accept returns a member (user_id) instead of an invitation (id)
      const autoJoined = !!resp.data && 'user_id' in resp.data
      invitePopupVisible.value = false
      if (autoJoined) {
        // The invitee is already a member — refresh the roster so they
        // appear immediately. No toast: the new row is the feedback.
        await loadMembers()
      } else {
        invitationsPage.value = 1
        await loadInvitations()
        MessagePlugin.success(t('tenantInvitation.inviteSuccess'))
      }
    } else {
      MessagePlugin.error(resp.message || t('tenantInvitation.errors.generic'))
    }
  } catch (err: any) {
    const status = err?.status
    if (status === 404) {
      MessagePlugin.error(t('tenantMember.errors.userNotFound'))
    } else if (status === 409) {
      // Server returns the same 409 for both "already a member" and
      // "already a pending invite". The message body discriminates,
      // but for the toast we show both possibilities folded into one
      // helpful line.
      MessagePlugin.error(
        err?.message ||
        `${t('tenantInvitation.errors.alreadyMember')} / ${t(
          'tenantInvitation.errors.pendingExists',
        )}`,
      )
    } else if (status === 400) {
      MessagePlugin.error(err?.message || t('tenantMember.errors.invalidRole'))
    } else {
      MessagePlugin.error(err?.message || t('tenantInvitation.errors.generic'))
    }
  } finally {
    adding.value = false
  }
}

async function onRoleChange(row: TenantMember, newRole: string) {
  const prev = row.role
  const next = newRole as TenantRole
  if (prev === next) return

  try {
    const resp = await updateMemberRole(activeTenantId.value, row.user_id, next)
    if (resp.success) {
      // Mutate the row by replacing it in `members.value` instead of
      // assigning `row.role = next` in place. The `row` argument here
      // is the row object handed in by t-table's slot scope, which in
      // some TDesign versions is a shallow copy that doesn't share
      // reactivity with the `members` array — assigning `row.role`
      // updates the local handle but not the rendered cell, so the
      // select keeps showing the previous value until a refresh.
      // Splicing a fresh object into the source array guarantees the
      // table re-renders.
      const idx = members.value.findIndex((m) => m.user_id === row.user_id)
      if (idx >= 0) {
        const merged = { ...members.value[idx], role: next }
        members.value.splice(idx, 1, merged)
        rememberMembersForAudit([merged])
      } else {
        row.role = next
      }
      MessagePlugin.success(t('tenantMember.roleChange.success'))
      return
    }
    MessagePlugin.error(resp.message || t('tenantMember.errors.generic'))
  } catch (err: any) {
    const status = err?.status
    if (status === 409) {
      MessagePlugin.error(t('tenantMember.errors.lastOwner'))
    } else if (status === 404) {
      MessagePlugin.error(t('tenantMember.errors.notFound'))
    } else {
      MessagePlugin.error(err?.message || t('tenantMember.errors.generic'))
    }
    // The t-select is bound via :model-value (one-way), so its rendered
    // value stays at `prev` automatically — no DOM hack needed.
  }
}

// Yerinde popconfirm, DialogPlugin modal onayının yerine geçer: "paylaşılan kaynak silme" gibi diğer liste içi
// silme girişleriyle stil birliği sağlar ve basit bir ikinci onayın üye yönetim tablosundaki gezinme akışını kesmesini önler.
// Hata dalları eski uygulamayla tutarlı kalır (409 last-owner / 404 not-found / varsayılan).
async function removeRow(row: TenantMember) {
  try {
    const resp = await removeMember(activeTenantId.value, row.user_id)
    if (resp.success) {
      await loadMembers()
      MessagePlugin.success(t('tenantMember.remove.success'))
    } else {
      MessagePlugin.error(resp.message || t('tenantMember.errors.generic'))
    }
  } catch (err: any) {
    const status = err?.status
    if (status === 409) {
      MessagePlugin.error(t('tenantMember.errors.lastOwner'))
    } else if (status === 404) {
      MessagePlugin.error(t('tenantMember.errors.notFound'))
    } else {
      MessagePlugin.error(err?.message || t('tenantMember.errors.generic'))
    }
  }
}

// Re-load whenever the active tenant resolves (or changes via the
// tenant switcher). onMounted alone would race with auth-store
// hydration — currentTenantId is often 0 at the moment this component
// mounts on a cold reload.
watch(
  activeTenantId,
  (id) => {
    if (id) {
      searchQuery.value = ''
      memberSearchQ.value = ''
      window.clearTimeout(memberSearchDebounceTimer)
      membersPage.value = 1
      invitationsPage.value = 1
      membersPageSize.value = 20
      invitationsPageSize.value = 20
      membersTotal.value = 0
      invitationsTotal.value = 0
      loadMembers()
      loadInvitations()
    }
  },
  { immediate: true },
)
</script>

<style lang="less" scoped>
@import (reference) '@/components/css/settings-section.less';

.tenant-members {
  width: 100%;
}

.member-cell {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;

  .member-avatar {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    width: 36px;
    height: 36px;
    overflow: hidden;
    border-radius: 11px;
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-text-color-primary);
    font-size: 11px;
    font-weight: 500;
    line-height: 1;

    img {
      width: 100%;
      height: 100%;
      object-fit: cover;
    }
  }

  .member-name {
    min-width: 0;
    font-weight: 500;
    font-size: 14px;
    color: var(--td-text-color-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .member-avatar--link {
    background: color-mix(in srgb, var(--td-brand-color) 10%, transparent);
    color: var(--td-brand-color);
  }

  .member-you {
    flex-shrink: 0;
    font-size: 12px;
    color: var(--td-text-color-secondary);
  }

  /* Name + secondary line (invitations table). */
  .member-stack {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;

    .member-name {
      display: block;
    }
  }

  .member-sub {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 12px;
    color: var(--td-text-color-secondary);
  }
}

.member-email {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--td-text-color-primary);
}

.member-date {
  color: var(--td-text-color-secondary);
  white-space: nowrap;
}

/* Members table: card-style shell aligned with the rag-platform data table. */
.members-table-shell {
  --members-border: color-mix(in srgb, var(--td-component-stroke) 70%, transparent);
  --members-muted: var(--td-bg-color-secondarycontainer);

  border-radius: var(--app-radius-2xl, 16px);
  border: 1px solid var(--members-border);
  background-color: var(--td-bg-color-container);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
  overflow: hidden;

  :deep(.t-table) {
    background: transparent;
    font-size: 13px;
  }

  :deep(.t-table thead th),
  :deep(.t-table .t-table__header th) {
    height: 44px !important;
    padding: 0 16px !important;
    font-size: 12px !important;
    font-weight: 500 !important;
    color: var(--td-text-color-secondary) !important;
    background-color: color-mix(in srgb, var(--members-muted) 60%, transparent) !important;
    border-bottom: 1px solid color-mix(in srgb, var(--td-component-stroke) 60%, transparent) !important;
    border-right: 1px solid color-mix(in srgb, var(--td-component-stroke) 40%, transparent);
  }

  :deep(.t-table tbody td) {
    padding: 12px 16px !important;
    font-size: 13px;
    border-bottom: 1px solid color-mix(in srgb, var(--td-component-stroke) 30%, transparent) !important;
    border-right: 1px solid color-mix(in srgb, var(--td-component-stroke) 20%, transparent);
  }

  :deep(.t-table thead th:last-child),
  :deep(.t-table tbody td:last-child) {
    border-right: none;
  }

  :deep(.t-table tbody tr:last-child td) {
    border-bottom: none !important;
  }

  :deep(.t-table tbody tr:nth-child(even)) {
    background-color: color-mix(in srgb, var(--members-muted) 20%, transparent);
  }

  :deep(.t-table.t-table--hoverable tbody tr:hover),
  :deep(.t-table tbody tr:hover) {
    background-color: color-mix(in srgb, var(--td-brand-color) 3%, transparent) !important;
  }

  :deep(.role-cell) {
    display: flex;
    align-items: center;
  }

  :deep(.member-action-btn.t-button) {
    color: var(--td-text-color-secondary);
    border-radius: var(--app-radius-md, 8px);

    &:hover {
      color: var(--td-error-color);
      background-color: color-mix(in srgb, var(--td-error-color) 8%, transparent);
    }
  }

  :deep(.member-action-btn.member-action-btn--neutral.t-button:hover) {
    color: var(--td-text-color-primary);
    background-color: var(--members-muted);
  }

  &.data-table-shell.data-table-shell--with-footer > .data-table-shell__pager {
    justify-content: space-between;
    padding: 12px 16px;
    border-top: 1px solid color-mix(in srgb, var(--td-component-stroke) 60%, transparent);
    background-color: color-mix(in srgb, var(--members-muted) 12%, transparent);
    font-size: 12px;
    color: var(--td-text-color-secondary);

    :deep(.t-pagination) {
      width: auto;
      gap: 8px;
    }

    :deep(.t-pagination__pager .t-pagination__number),
    :deep(.t-pagination__btn) {
      min-width: 28px;
      height: 28px;
      border: 1px solid transparent !important;
      border-radius: var(--app-radius-md, 8px) !important;
      background: transparent;
      font-size: 12px;
    }

    :deep(.t-pagination__pager .t-pagination__number:hover),
    :deep(.t-pagination__btn:not(.t-is-disabled):hover) {
      background-color: var(--members-muted) !important;
    }

    :deep(.t-pagination__pager .t-pagination__number.t-is-current) {
      border-color: var(--td-component-stroke) !important;
      background-color: var(--td-bg-color-container) !important;
      color: var(--td-text-color-primary) !important;
      box-shadow: none;
      font-weight: 500;
    }

    /* "Sayfa başına 100" etiketinin kesilmemesi için sayfa boyutu seçimi genişletilir. */
    :deep(.t-pagination__select),
    :deep(.t-pagination__select .t-select__wrap),
    :deep(.t-pagination__select .t-select) {
      width: 168px;
    }

    :deep(.t-pagination__select .t-input) {
      height: 28px;
      border-radius: var(--app-radius-md, 8px);
      font-size: 12px;
    }
  }
}

.members-pager-info {
  white-space: nowrap;
}

.members-table-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 48px 16px;
  text-align: center;

  &__icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    border-radius: 10px;
    background-color: var(--td-bg-color-secondarycontainer);
    color: var(--td-text-color-secondary);
    font-size: 18px;
  }

  &__text {
    font-size: 13px;
    color: var(--td-text-color-secondary);
  }
}

.members-table-skeleton {
  display: flex;
  flex-direction: column;
  gap: 8px;

  &__bar {
    display: block;
    height: 48px;
    border-radius: 12px;
    background-color: var(--td-bg-color-secondarycontainer);
    animation: members-skeleton-pulse 1.6s ease-in-out infinite;
  }
}

@keyframes members-skeleton-pulse {
  0%,
  100% {
    opacity: 1;
  }

  50% {
    opacity: 0.5;
  }
}

/* Role pill: shared by the read-only badge and the editable select trigger. */
.role-pill {
  display: inline-flex;
  align-items: center;
  height: 28px;
  padding: 0 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 999px;
  background-color: color-mix(in srgb, var(--td-bg-color-secondarycontainer) 60%, transparent);
  color: var(--td-text-color-primary);
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
}

.role-pill--owner {
  --role-pill-border: rgba(245, 158, 11, 0.25);
  --role-pill-bg: rgba(245, 158, 11, 0.1);
  --role-pill-fg: #92400e;
}

.role-pill--admin {
  --role-pill-border: rgba(14, 165, 233, 0.25);
  --role-pill-bg: rgba(14, 165, 233, 0.1);
  --role-pill-fg: #075985;
}

.role-pill--contributor {
  --role-pill-border: rgba(16, 185, 129, 0.25);
  --role-pill-bg: rgba(16, 185, 129, 0.1);
  --role-pill-fg: #065f46;
}

/* Invitation status pills reuse the role-pill shape. */
.status-pill--pending {
  --role-pill-border: rgba(16, 185, 129, 0.25);
  --role-pill-bg: rgba(16, 185, 129, 0.1);
  --role-pill-fg: #065f46;
}

.status-pill--accepted {
  --role-pill-border: rgba(14, 165, 233, 0.25);
  --role-pill-bg: rgba(14, 165, 233, 0.1);
  --role-pill-fg: #075985;
}

.status-pill--declined,
.status-pill--revoked {
  --role-pill-border: rgba(245, 158, 11, 0.25);
  --role-pill-bg: rgba(245, 158, 11, 0.1);
  --role-pill-fg: #92400e;
}

.status-pill--expired {
  --role-pill-border: rgba(239, 68, 68, 0.25);
  --role-pill-bg: rgba(239, 68, 68, 0.1);
  --role-pill-fg: #991b1b;
}

.role-pill.status-pill--pending,
.role-pill.status-pill--accepted,
.role-pill.status-pill--declined,
.role-pill.status-pill--revoked,
.role-pill.status-pill--expired,
.role-pill.role-pill--owner,
.role-pill.role-pill--admin,
.role-pill.role-pill--contributor {
  border-color: var(--role-pill-border);
  background-color: var(--role-pill-bg);
  color: var(--role-pill-fg);
}

.members-table-shell :deep(.member-role-select.t-select) {
  width: auto;
  min-width: 128px;
  max-width: 100%;

  .t-input {
    height: 28px;
    padding: 0 10px;
    border: 1px solid var(--td-component-stroke);
    border-radius: 999px;
    background-color: color-mix(in srgb, var(--td-bg-color-secondarycontainer) 60%, transparent);
    box-shadow: none;
    font-size: 12px;
    font-weight: 500;
  }

  .t-input__inner {
    font-size: 12px;
    font-weight: 500;
    color: inherit;
  }
}

.members-table-shell :deep(.member-role-select.role-pill--owner),
.members-table-shell :deep(.member-role-select.role-pill--admin),
.members-table-shell :deep(.member-role-select.role-pill--contributor) {
  .t-input {
    border-color: var(--role-pill-border);
    background-color: var(--role-pill-bg);
    color: var(--role-pill-fg);
  }

  .t-input__inner,
  .t-input__suffix,
  .t-fake-arrow {
    color: var(--role-pill-fg);
  }
}

.section-header {
  .settings-section-header();
}

.section-header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.section-header-titlewrap {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  min-width: 0;

  h2 {
    margin: 0;
    line-height: 1.25;
  }
}

/* Sağ üstteki 「Denetim günlükleri」 girişi. t-button variant="text" kendi renk
   etkileşimine sahiptir; burada yalnızca flex davranışı ayarlanır, wrap sırasında sıkışması önlenir. */
.header-audit-btn {
  flex-shrink: 0;
}

.members-tab-layout {
  display: flex;
  flex-direction: column;
}

/* Şerit sayfalı kart: yatay kaydırma yalnızca tablo gövdesinde olur, alt bilgi kaydırmaya katılmaz; böylece sayfalama çubuğunun kayması veya hizasının bozulması önlenir.
   Çift sınıf seçici, kökteki .data-table-shell üzerindeki overflow-x: auto değerini ezmek için kullanılır */
.data-table-shell.data-table-shell--with-footer {
  display: flex;
  flex-direction: column;
  overflow: hidden;

  >.data-table-shell__scroll {
    overflow-x: auto;
    min-width: 0;
  }

  >.data-table-shell__pager {
    flex-shrink: 0;
    display: flex;
    justify-content: flex-end;
    align-items: center;
    flex-wrap: wrap;
    gap: 8px 12px;
    padding: 10px 14px;
    border-top: 1px solid var(--td-component-stroke);
    background-color: var(--td-bg-color-container);

    :deep(.t-pagination) {
      flex-wrap: wrap;
      justify-content: flex-end;
      row-gap: 8px;
    }
  }
}

/* Listenin üzerindeki başlık satırı: solda “Alan üyeleri [N] · K filtrelendi”; sağda
   “Arama + davet düğmesi” grubudur. Görsel önceliği “Bekleyen davetler” ile aynıdır. */
.members-list-wrap {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.members-list-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 0 2px;
  flex-wrap: wrap;
}

.members-list-titlewrap {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.members-list-title {
  font-size: var(--app-text-base);
  font-weight: 500;
  color: var(--td-text-color-primary);
}

/* Sayının etrafına açık zeminli, yuvarlatılmış köşeli bir rozet ekleyin; böylece tek başına duran “Üyeler 1” ifadesi
   bir mizanpaj kalıntısı gibi okunmaz. Renk tonu td-tag default+light ile uyumludur. */
.members-list-count-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 22px;
  height: 20px;
  padding: 0 7px;
  border-radius: var(--app-radius-lg);
  background-color: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-primary);
  font-size: var(--app-text-sm);
  font-weight: 600;
  line-height: 1;
}

.members-list-filter-hint {
  font-size: var(--app-text-sm);
  color: var(--td-text-color-secondary);
}

.members-list-actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex: 0 1 auto;
  min-width: 0;
}

.members-list-search {
  /* Dış kapsayıcının boyutunu açık bir width ile sabitleyin; böylece inline-flex üst kapsayıcısındaki alt öğelerin
     içerik genişliği değişimini (TDesign t-input hover/focus durumunda clearable
     × simgesini gösterir ve kenarlık durumu değişir) izlemesi ve tüm satırın yatayda titremesi önlenir. */
  flex: 0 0 14rem;
  width: 14rem;
  min-width: 0;

  :deep(.t-input) {
    width: 100%;
  }
}

/* Liste başlığındaki davet düğmesi: outline + primary, kendi çerçevesine sahiptir ve sade bir icon
   düğmesinden daha belirgindir; shape="square" onu yine de kompakt bir simge düğmesi yapar. */
.members-list-add-btn {
  flex-shrink: 0;
}

@media (max-width: 560px) {
  .members-list-actions {
    width: 100%;
    justify-content: flex-start;
  }

  .members-list-search {
    flex: 1 1 auto;
    width: auto;
    max-width: none;
  }
}

.data-table-shell {
  overflow-x: auto;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--td-component-stroke);
  background-color: var(--td-bg-color-container);

  &:deep(thead th) {
    font-weight: 600;
    font-size: var(--app-text-md);
  }

  &:deep(.t-table td),
  &:deep(.t-table th) {
    padding-top: 12px;
    padding-bottom: 12px;
  }

  /* Rol sütunu: açılır menü içerik genişliğine daralır, artık tüm hücreyi kaplamaz. Önceki 100% değeri, dar rol
     adlarında (ör. "Owner") boş görünür ve diğer sütunlarla hizasız kalır. */
  &:deep(.role-cell) {
    display: flex;
    align-items: center;
    min-width: 0;
    box-sizing: border-box;
  }

  &:deep(.member-role-select.t-select) {
    width: 100%;
  }
}

/* Audit drawer's data-table-shell variant: only used inside the audit
   drawer, so members-list specific tweaks (role-cell, role-select)
   from the base block don't kick in. The selector is more specific
   than `.data-table-shell` alone so the per-cell overrides win. */
.audit-table-shell {
  &:deep(.t-table td),
  &:deep(.t-table th) {
    /* See SystemSettings audit table: middle keeps the row weight
       unified across single-line tag cells and multi-line diff cells. */
    vertical-align: middle;
    padding-top: 14px;
    padding-bottom: 14px;
  }

  /* Sticky thead so the column labels survive long scrolls. The drawer's
     `.audit-scroll-area` is the scroll container; top:0 pins the
     headers there. z-index sits above hover/expand backgrounds, and
     the inset shadow replaces the row separator that would otherwise
     scroll out of frame. */
  &:deep(thead th) {
    position: sticky;
    top: 0;
    z-index: 2;
    background-color: var(--td-bg-color-secondarycontainer) !important;
    box-shadow: inset 0 -1px 0 var(--td-component-stroke);
  }

  &:deep(.t-table tbody tr:hover > td) {
    background-color: var(--td-bg-color-container-hover);
  }

  &:deep(.t-table tbody tr.t-table__expanded-row > td) {
    padding: 0 !important;
    background-color: transparent;
  }

  &:deep(.t-table__expandable-icon-cell) {
    width: 36px;
  }
}

.loading-inline,
.error-inline {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 20px 0 8px;
}

/* Invite popup: confirm copy + footer actions (anchored beside +). */
.member-invite-popup-inner {
  max-width: 100%;
}

.member-invite-popup-title {
  font-size: var(--app-text-lg);
  font-weight: 500;
  color: var(--td-text-color-primary);
  margin: 0 0 12px;
  line-height: 1.35;
}

.member-invite-form {
  &:deep(.t-form__label) {
    display: flex;
    align-items: center;
    min-height: var(--td-comp-size-m);
  }

  &:deep(.t-input__extra) {
    position: static;
  }

  &:deep(.t-form__item) {
    margin-bottom: 14px;

    &:last-child {
      margin-bottom: 4px;
    }
  }
}

.invite-confirm-body {
  padding: 4px 0 8px;
  color: var(--td-text-color-primary);
  font-size: var(--app-text-base);
  line-height: 1.6;
}

.invite-popup-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 16px;
}

/* Share-link result panel — single-row layout: input field stretches,
 * copy button stays fixed-width. Mirrors the rounded card style of
 * the surrounding popup. */
.share-link-result {
  padding: 4px 0 0;
}

.share-link-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
}

.share-link-row__input {
  flex: 1 1 auto;
  min-width: 0;
  padding: 7px 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-sm);
  background: var(--td-bg-color-page);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: var(--app-text-sm);
  color: var(--td-text-color-primary);
  outline: none;
}

.share-link-row__input:focus {
  border-color: var(--td-brand-color);
}

/* Inline tag for share-link rows in the pending invitations list,
 * so they read as "this row is a share link" instead of looking like
 * a malformed per-user invitation with a missing email. */
.member-cell .member-stack .share-link-title {
  color: var(--td-brand-color);
}

.pending-invitations-section {
  margin-bottom: 24px;
  display: flex;
  flex-direction: column;
  gap: 10px;

  .pending-invitations-header {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .pending-invitations-titlewrap {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .pending-invitations-title {
    font-size: var(--app-text-base);
    font-weight: 500;
    color: var(--td-text-color-primary);
  }

  .pending-invitations-desc {
    font-size: var(--app-text-sm);
    color: var(--td-text-color-secondary);
  }

  .pending-invitations-empty {
    padding: 10px 12px;
    border: 1px dashed var(--td-component-stroke);
    border-radius: var(--app-radius-md);
    color: var(--td-text-color-secondary);
    font-size: var(--app-text-md);
    background: var(--td-bg-color-container);
  }
}

.empty-state {
  padding: 40px 0 16px;
  display: flex;
  justify-content: center;
}

.audit-panel {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding-top: 8px;
}

/* Inside the drawer the t-drawer body already gives us padding, so
   drop the top offset; otherwise the audit-header floats away from
   the drawer title with no visual anchor. */
.audit-panel--drawer {
  padding-top: 0;
}

.audit-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: var(--td-bg-color-secondarycontainer);
  padding: 12px 16px;
  border-radius: var(--app-radius-md);
  gap: 12px;

  .audit-desc {
    flex: 1;
    min-width: 0;
    font-size: var(--app-text-md);
    color: var(--td-text-color-secondary);
  }

  .audit-refresh-btn {
    flex-shrink: 0;
  }
}

.audit-drawer-inner {
  display: flex;
  flex-direction: column;
  flex: 1 1 auto;
  gap: 14px;
  min-height: 0;
  width: 100%;
  box-sizing: border-box;
}

.audit-drawer-fill {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.audit-drawer-branch {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.audit-drawer-branch--error {
  justify-content: center;

  .error-inline {
    width: 100%;
  }
}

.audit-drawer-branch--empty.empty-state--audit {
  flex: 1 1 auto;
  justify-content: center;
  align-items: center;
  padding: 24px 12px;
  min-height: 0;
}

.audit-scroll-area {
  flex: 1 1 auto;
  min-height: 0;
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

  .audit-target-key {
    font-size: var(--app-text-md);
    color: var(--td-text-color-primary);
    word-break: break-all;
  }

  .audit-target-diff {
    font-size: var(--app-text-sm);
    color: var(--td-text-color-secondary);
    font-family: var(--td-font-family-mono);
    word-break: break-all;
    line-height: 1.4;
  }

  .audit-target-empty {
    color: var(--td-text-color-placeholder);
  }
}

.audit-path {
  font-family: var(--td-font-family-mono);
  font-size: var(--app-text-sm);
  color: var(--td-text-color-secondary);
  word-break: break-all;

  .audit-method {
    display: inline-block;
    font-weight: 600;
    color: var(--td-text-color-primary);
    margin-right: 4px;
  }
}

/* Expandable row body. Background steps off-card so the nested
   context is clearly distinct from the row strip above it. Mirrors
   the platform audit drawer in SystemSettings.vue. */
.audit-expanded {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 12px 16px;
  background: var(--td-bg-color-container-hover);
}

.audit-expanded-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 10px 18px;
}

.audit-expanded-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.audit-expanded-label {
  font-size: var(--app-text-xs);
  font-weight: 500;
  color: var(--td-text-color-secondary);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.audit-expanded-value {
  font-size: var(--app-text-sm);
  color: var(--td-text-color-primary);
  word-break: break-all;
}

.audit-expanded-details {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.audit-expanded-json {
  margin: 0;
  padding: 10px 12px;
  font-size: var(--app-text-sm);
  line-height: 1.55;
  color: var(--td-text-color-primary);
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-sm);
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 280px;
  overflow: auto;
}

.mono {
  font-family: var(--td-font-family-mono);
}
</style>

<style lang="less">
/* Members table role pills: brighter text on dark backgrounds. */
:root[theme-mode='dark'] .members-table-shell {
  .role-pill--owner {
    --role-pill-fg: #fcd34d;
  }

  .role-pill--admin {
    --role-pill-fg: #7dd3fc;
  }

  .role-pill--contributor,
  .status-pill--pending {
    --role-pill-fg: #6ee7b7;
  }

  .status-pill--accepted {
    --role-pill-fg: #7dd3fc;
  }

  .status-pill--declined,
  .status-pill--revoked {
    --role-pill-fg: #fcd34d;
  }

  .status-pill--expired {
    --role-pill-fg: #fca5a5;
  }
}

/* t-popup body'ye eklenir, genel stil gerekir; z-index ayarlar tam ekran örtüsünden (2000) yüksek olmalıdır. */
.member-invite-popup-overlay {
  z-index: 3050 !important;

}

:root[theme-mode='dark'] .member-invite-popup-overlay .t-popup__content {
  background: rgba(36, 36, 36, 0.92) !important;
  border-color: rgba(255, 255, 255, 0.08) !important;
  box-shadow:
    0 0 0 0.5px rgba(255, 255, 255, 0.05),
    0 2px 4px rgba(0, 0, 0, 0.12),
    0 8px 32px rgba(0, 0, 0, 0.28) !important;
}

/* Rol açılır menüsü body'ye eklendiğinde davet Popup'ı / ayar örtüsü tarafından kapatılabilir; sınıf adı t-popup kök düğümüne eklenir */
.tenant-members-role-select-popup {
  z-index: 6200 !important;

  .role-option {
    display: inline-flex;
    align-items: center;
    gap: 8px;
  }
  .role-option-icon {
    font-size: var(--app-text-base);
    color: var(--td-text-color-secondary);
  }
}

/* Üye sayfası denetim çekmecesi body'ye teleport edilir; iç kaydırma için body yükseklik zincirini tam doldurmak üzere genel stil gerekir */
.t-drawer.tenant-members-audit-drawer.t-drawer--right .t-drawer__content-wrapper--right {
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  max-width: 100vw;
  max-height: 100vh;
  height: 100%;
}

.t-drawer.tenant-members-audit-drawer .t-drawer__body {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
  overflow: hidden !important;
}

.t-drawer.tenant-members-audit-drawer .setting-drawer__body {
  flex: 1 1 auto;
  min-height: 0;
}
</style>
