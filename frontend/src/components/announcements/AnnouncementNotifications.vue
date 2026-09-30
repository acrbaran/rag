<template>
  <component :is="inSettings ? 'section' : 'div'" v-if="authenticated" class="announcement-notifications"
    :class="{ 'announcement-notifications--settings': inSettings }"
    :aria-label="inSettings ? t('announcements.settingsTitle') : undefined" :tabindex="inSettings ? -1 : undefined">
    <div class="announcement-notifications__header">
      <h2 class="announcement-notifications__title">
        {{ t(inSettings ? 'announcements.settingsTitle' : 'announcements.notificationsTitle') }}
      </h2>
      <div class="announcement-notifications__header-actions">
        <button type="button" class="ann-btn ann-btn--ghost ann-btn--icon-sm" :title="t('announcements.markAllRead')"
          :aria-label="t('announcements.markAllRead')" :disabled="state.notificationsBusy || unread.length === 0"
          @click="updateNotifications('read', unread.map((item) => item.ref))">
          <LucideIcon name="check-check" />
        </button>
        <button type="button" class="ann-btn ann-btn--ghost ann-btn--icon-sm ann-btn--danger-hover"
          :title="t('announcements.deleteAllNotifications')" :aria-label="t('announcements.deleteAllNotifications')"
          :disabled="state.notificationsBusy || notifications.length === 0"
          @click="updateNotifications('delete', notifications.map((item) => item.ref))">
          <LucideIcon name="trash-2" />
        </button>
        <button v-if="!inSettings" type="button" class="ann-btn ann-btn--ghost ann-btn--icon-sm"
          :aria-label="t('announcements.dismiss')" @click="emit('close')">
          <LucideIcon name="x" />
        </button>
      </div>
    </div>
    <p v-if="inSettings" class="announcement-notifications__note">{{ t('announcements.historyDescription') }}</p>
    <p v-if="state.notificationsError" role="alert" class="announcement-notifications__note is-error">
      {{ state.notificationsError }}
    </p>
    <div class="announcement-notifications__scroll">
      <div v-if="localPreview" class="announcement-notifications__preview" role="status">
        <p class="announcement-notifications__preview-label">
          {{ t('announcements.previewLabel') }} · {{ announcementTypeLabel(localPreview.type, t) }}
        </p>
        <p class="announcement-notifications__preview-title">{{ localPreview.title }}</p>
        <AnnouncementBody :title="localPreview.title" :body="localPreview.body"
          :open-details="() => openDetails(localPreview!.title, localPreview!.body)" />
      </div>
      <ul v-if="notifications.length > 0" class="announcement-notifications__list">
        <li v-for="item in visibleNotifications" :key="item.key" :aria-label="item.title || undefined"
          class="announcement-notifications__item" :class="{ 'is-unread': !item.state?.read }">
          <LucideIcon name="bell" class="announcement-notifications__item-icon" />
          <div class="announcement-notifications__item-main">
            <p class="announcement-notifications__item-type">
              <span v-if="!item.state?.read" class="announcement-notifications__unread-dot"
                :aria-label="t('announcements.unread')" />
              <span>{{ announcementTypeLabel(item.type, t) }}</span>
            </p>
            <p :data-announcement-id="item.ref.announcementId" :data-announcement-phase="item.ref.phase"
              class="announcement-notifications__item-title" :class="{ 'is-read': item.state?.read }">
              {{ item.title }}
            </p>
            <AnnouncementBody :title="item.title" :body="item.body"
              :on-read-more="!item.state?.read ? () => updateNotifications('read', [item.ref]) : undefined"
              :open-details="!inSettings ? () => openItemDetails(item) : undefined" />
            <time v-if="item.time != null" class="announcement-notifications__time"
              :datetime="new Date(item.time).toISOString()">
              {{ formatTime(item.time) }}
            </time>
          </div>
          <div class="announcement-notifications__item-actions">
            <button type="button" class="ann-btn ann-btn--ghost ann-btn--icon-sm" :disabled="state.notificationsBusy"
              :title="t(item.state?.read ? 'announcements.markUnread' : 'announcements.markRead')"
              :aria-label="`${t(item.state?.read ? 'announcements.markUnread' : 'announcements.markRead')}: ${item.title}`"
              @click="updateNotifications(item.state?.read ? 'unread' : 'read', [item.ref])">
              <LucideIcon :name="item.state?.read ? 'mail' : 'mail-check'" />
            </button>
            <button type="button" class="ann-btn ann-btn--ghost ann-btn--icon-sm ann-btn--danger-hover"
              :disabled="state.notificationsBusy" :title="t('announcements.deleteNotification')"
              :aria-label="`${t('announcements.deleteNotification')}: ${item.title}`"
              @click="updateNotifications('delete', [item.ref])">
              <LucideIcon name="trash-2" />
            </button>
          </div>
        </li>
      </ul>
      <p v-else-if="!localPreview" class="announcement-notifications__empty">
        {{ t(state.data ? 'announcements.notificationsEmpty' : state.unavailable ? 'announcements.admin.loadFailed' :
          'announcements.admin.loading') }}
      </p>
    </div>
    <div v-if="inSettings && notifications.length > 0" class="announcement-notifications__pagination">
      <NumericPagination :page="currentSettingsPage" :page-count="settingsPageCount"
        :label="t('announcements.settingsTitle')" :disabled="state.notificationsBusy"
        @page="settingsPage = $event" />
    </div>
    <p v-if="state.unavailable && state.data" role="status" class="announcement-notifications__note is-footer">
      {{ t('announcements.connectionLost') }}
    </p>
    <div v-if="!inSettings" class="announcement-notifications__footer">
      <button type="button" class="ann-btn ann-btn--ghost ann-btn--block announcement-notifications__all"
        :aria-label="t('announcements.allNotifications')" @click="emit('open-all')">
        {{ t('announcements.allNotifications') }}
        <LucideIcon name="arrow-right" />
      </button>
    </div>
  </component>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { announcementTypeLabel, type AnnouncementType, type NotificationRef } from '@/api/announcements'
import { useAnnouncements } from '@/composables/useAnnouncements'
import AnnouncementBody from './AnnouncementBody.vue'
import LucideIcon from './LucideIcon.vue'
import NumericPagination from './NumericPagination.vue'

const props = withDefaults(defineProps<{ inSettings?: boolean }>(), { inSettings: false })
const emit = defineEmits<{
  (e: 'close'): void
  (e: 'open-all'): void
  (e: 'details', value: { title: string; body: string }): void
}>()

const { t, locale } = useI18n()
const { state, authenticated, updateNotifications } = useAnnouncements()
const settingsPage = ref(1)

interface NotificationItem {
  key: string
  ref: NotificationRef
  state?: { read: boolean; deleted: boolean }
  title: string
  body: string
  type?: AnnouncementType
  time: number | null
}

const localPreview = computed(() =>
  !props.inSettings && state.preview && !state.preview.showBanner ? state.preview : null,
)

const notifications = computed<NotificationItem[]>(() =>
  (state.data?.items ?? [])
    .map((item) => ({
      key: `${item.id}:published`,
      ref: { announcementId: item.id, phase: 'published' as const },
      state: item.notificationState?.published,
      title: item.title,
      body: item.body,
      type: item.type,
      time: item.publishedAt,
    }))
    .filter((item) => !item.state?.deleted)
    .sort((a, b) => (b.time ?? 0) - (a.time ?? 0)),
)

const unread = computed(() => notifications.value.filter((item) => !item.state?.read))
const settingsPageCount = computed(() => Math.max(1, Math.ceil(notifications.value.length / 10)))
const currentSettingsPage = computed(() => Math.min(settingsPage.value, settingsPageCount.value))
const visibleNotifications = computed(() =>
  props.inSettings
    ? notifications.value.slice((currentSettingsPage.value - 1) * 10, currentSettingsPage.value * 10)
    : notifications.value,
)

function formatTime(time: number) {
  return new Intl.DateTimeFormat(locale.value, { dateStyle: 'medium', timeStyle: 'short' }).format(time)
}

function openDetails(title: string, body: string) {
  emit('details', { title, body })
}

function openItemDetails(item: NotificationItem) {
  if (!item.state?.read) void updateNotifications('read', [item.ref])
  openDetails(item.title, item.body)
}
</script>

<style lang="less" scoped>
.announcement-notifications {
  color: var(--td-text-color-primary);
  font-size: var(--app-text-base);
  text-align: left;

  p {
    margin: 0;
  }

  &--settings {
    margin-bottom: 16px;
    overflow: hidden;
    border: 1px solid var(--td-component-stroke);
    border-radius: var(--app-radius-token-2xl);
    outline: none;
  }
}

.announcement-notifications__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.announcement-notifications__title {
  margin: 0;
  font-size: var(--app-text-base);
  font-weight: 600;
  line-height: 1.5;
}

.announcement-notifications__header-actions {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  gap: 4px;
}

.announcement-notifications__note {
  padding: 12px 16px;
  border-bottom: 1px solid var(--td-component-stroke);
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);

  &.is-error {
    color: var(--td-error-color);
  }

  &.is-footer {
    border-top: 1px solid var(--td-component-stroke);
    border-bottom: 0;
  }
}

.announcement-notifications__scroll {
  max-height: min(28rem, 60dvh);
  overflow-y: auto;
  overscroll-behavior: contain;
}

.announcement-notifications__preview {
  padding: 16px;
  border-bottom: 1px solid var(--td-component-stroke);
  background: color-mix(in srgb, var(--app-surface-muted) 40%, transparent);
}

.announcement-notifications__preview-label {
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);
  font-weight: 500;
}

.announcement-notifications__preview-title {
  margin-top: 4px;
  font-weight: 500;
  word-break: break-word;
}

.announcement-notifications__list {
  margin: 0;
  padding: 0;
  list-style: none;

  > li + li {
    border-top: 1px solid var(--td-component-stroke);
  }
}

.announcement-notifications__item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px;

  &.is-unread {
    background: color-mix(in srgb, var(--td-brand-color) 5%, transparent);
  }
}

.announcement-notifications__item-icon {
  margin-top: 2px;
  color: var(--td-text-color-secondary);
}

.announcement-notifications__item-main {
  flex: 1;
  min-width: 0;
}

.announcement-notifications__item-type {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);
}

.announcement-notifications__unread-dot {
  width: 6px;
  height: 6px;
  flex-shrink: 0;
  border-radius: 50%;
  background: var(--td-brand-color);
}

.announcement-notifications__item-title {
  margin-top: 4px;
  font-weight: 600;
  word-break: break-word;

  &.is-read {
    color: var(--td-text-color-secondary);
    font-weight: 400;
  }
}

.announcement-notifications__time {
  display: block;
  margin-top: 8px;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);
}

.announcement-notifications__item-actions {
  display: flex;
  flex-shrink: 0;
  flex-direction: column;
  gap: 4px;
}

.announcement-notifications__empty {
  padding: 32px 16px;
  color: var(--td-text-color-secondary);
  text-align: center;
}

.announcement-notifications__pagination {
  display: flex;
  justify-content: flex-end;
  padding: 8px;
  border-top: 1px solid var(--td-component-stroke);
}

.announcement-notifications__footer {
  padding: 8px;
  border-top: 1px solid var(--td-component-stroke);
}

.announcement-notifications__all {
  justify-content: space-between;
}
</style>
