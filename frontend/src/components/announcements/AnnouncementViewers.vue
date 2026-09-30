<template>
  <button type="button" class="ann-btn ann-btn--outline ann-btn--xs"
    :aria-label="`${t('announcements.admin.viewers')}: ${announcement.title}`" @click="openDialog">
    <LucideIcon name="eye" :size="14" />{{ t('announcements.admin.viewers') }}
  </button>
  <t-dialog v-model:visible="open" :footer="false" width="576px" attach="body" destroy-on-close
    dialog-class-name="announcement-viewers-dialog">
    <template #header>
      <div class="announcement-viewers-dialog__header">
        <span class="announcement-viewers-dialog__title">
          {{ t('announcements.admin.viewers') }} · {{ announcement.title }}
        </span>
        <span class="announcement-viewers-dialog__description">{{ t('announcements.admin.viewersHint') }}</span>
      </div>
    </template>
    <div class="announcement-viewers-dialog__content">
      <div class="announcement-viewers-dialog__buttons">
        <button type="button" class="ann-btn ann-btn--default ann-btn--sm" aria-pressed="true">
          {{ t('announcements.admin.phasePublished') }}
        </button>
        <button type="button" class="ann-btn ann-btn--ghost ann-btn--sm" :disabled="!current" @click="revision++">
          {{ t('announcements.admin.refresh') }}
        </button>
      </div>
      <p v-if="!current" role="status">{{ t('announcements.admin.loading') }}</p>
      <div v-else-if="current.error" role="alert" class="announcement-viewers-dialog__error">
        <p>{{ t('announcements.admin.viewersFailed') }}</p>
        <button type="button" class="ann-btn ann-btn--outline" @click="revision++">
          {{ t('announcements.admin.retry') }}
        </button>
      </div>
      <template v-else-if="current.data">
        <p role="status" class="announcement-viewers-dialog__count">
          {{ t('announcements.admin.viewersCount', { count: current.data.total }) }}
        </p>
        <ul v-if="current.data.items.length" class="announcement-viewers-dialog__list">
          <li v-for="viewer in current.data.items" :key="viewer.userId" class="announcement-viewers-dialog__row">
            <div class="announcement-viewers-dialog__person">
              <p class="announcement-viewers-dialog__name">{{ viewer.name }}</p>
              <p class="announcement-viewers-dialog__email">{{ viewer.email }}</p>
            </div>
            <time class="announcement-viewers-dialog__time" :datetime="new Date(viewer.seenAt).toISOString()">
              {{ formatTime(viewer.seenAt) }}
            </time>
          </li>
        </ul>
        <p v-else>{{ t('announcements.admin.viewersEmpty') }}</p>
        <NumericPagination :page="page" :page-count="Math.max(1, Math.ceil(current.data.total / 50))"
          :label="t('announcements.admin.viewers')" @page="page = $event" />
      </template>
    </div>
  </t-dialog>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { announcementsApi, type Announcement, type AnnouncementViewers } from '@/api/announcements'
import LucideIcon from './LucideIcon.vue'
import NumericPagination from './NumericPagination.vue'

const props = defineProps<{ announcement: Announcement }>()
const { t, locale } = useI18n()

const phase = 'published' as const
const open = ref(false)
const page = ref(1)
const revision = ref(0)
const result = ref<{ key: string; data?: AnnouncementViewers; error?: boolean } | null>(null)
const key = computed(() => `${props.announcement.id}:${phase}:${page.value}:${revision.value}`)
const current = computed(() => (result.value?.key === key.value ? result.value : null))

let controller: AbortController | null = null

function openDialog() {
  page.value = 1
  revision.value++
  open.value = true
}

watch([open, key], () => {
  controller?.abort()
  controller = null
  if (!open.value) return
  const requestKey = key.value
  const current = new AbortController()
  controller = current
  announcementsApi.viewers(props.announcement.id, phase, page.value, current.signal).then(
    (data) => { if (!current.signal.aborted) result.value = { key: requestKey, data } },
    () => { if (!current.signal.aborted) result.value = { key: requestKey, error: true } },
  )
})

onBeforeUnmount(() => controller?.abort())

function formatTime(time: number) {
  return new Intl.DateTimeFormat(locale.value, { dateStyle: 'medium', timeStyle: 'short' }).format(time)
}
</script>

<style lang="less">
.announcement-viewers-dialog {
  .t-dialog__body {
    padding-top: 16px;
  }
}

.announcement-viewers-dialog__header {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}

.announcement-viewers-dialog__title {
  overflow-wrap: anywhere;
}

.announcement-viewers-dialog__description {
  color: var(--td-text-color-secondary);
  font-family: var(--app-font-family);
  font-size: var(--app-text-base);
  font-weight: 400;
  line-height: 1.5;
}

.announcement-viewers-dialog__content {
  display: flex;
  flex-direction: column;
  gap: 16px;
  color: var(--td-text-color-primary);

  p {
    margin: 0;
  }
}

.announcement-viewers-dialog__buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.announcement-viewers-dialog__error {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
}

.announcement-viewers-dialog__count {
  color: var(--td-text-color-secondary);
}

.announcement-viewers-dialog__list {
  max-height: 320px;
  margin: 0;
  padding: 0;
  overflow-y: auto;
  list-style: none;

  > li + li {
    border-top: 1px solid var(--td-component-stroke);
  }
}

.announcement-viewers-dialog__row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 12px 0;
}

.announcement-viewers-dialog__person {
  min-width: 0;
}

.announcement-viewers-dialog__name {
  font-weight: 500;
  word-break: break-word;
}

.announcement-viewers-dialog__email,
.announcement-viewers-dialog__time {
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);
  word-break: break-all;
}
</style>
