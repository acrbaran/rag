<template>
  <section v-if="authenticated && current" class="announcement-banner" :aria-label="t('announcements.admin.tab')">
    <AnnouncementBannerCard :key="current.key" :title="current.announcement.title" :body="current.announcement.body"
      :type="current.announcement.type" :banner-color="current.announcement.bannerColor"
      :announcement-id="current.preview ? undefined : current.announcement.id"
      :unavailable="state.unavailable && !current.preview" :position="index + 1" :total="items.length"
      :has-previous="items.length > 1" :has-next="items.length > 1" dismissible
      :dismiss-disabled="!current.preview && state.notificationsBusy" @previous="move(-1)" @next="move(1)"
      @dismiss="dismiss" />
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Announcement } from '@/api/announcements'
import { useAnnouncements } from '@/composables/useAnnouncements'
import AnnouncementBannerCard from './AnnouncementBannerCard.vue'

type BannerItem = {
  key: string
  announcement: Pick<Announcement, 'id' | 'version' | 'title' | 'body' | 'type' | 'bannerColor'>
  preview: boolean
}

const { t } = useI18n()
const { state, authenticated, clearPreview, updateNotifications } = useAnnouncements()
const viewIndex = ref(0)
const dismissed = ref<string[]>([])

const items = computed<BannerItem[]>(() => {
  const data = state.data
  const fallback = data?.announcement?.status === 'maintenance' ? [data.announcement] : []
  const live = (data?.announcements ?? fallback).filter(
    (item) =>
      item.status === 'maintenance' &&
      item.showBanner &&
      item.notificationState?.published?.dismissedVersion !== item.version,
  )
  const preview = state.preview
  return [
    ...(preview?.showBanner
      ? [{
          key: `preview:${preview.title}:${preview.body}:${preview.bannerColor}`,
          announcement: { id: '', version: 0, ...preview },
          preview: true,
        }]
      : []),
    ...live.map((announcement) => ({ key: `${announcement.id}:${announcement.version}`, announcement, preview: false })),
  ].filter((item) => !dismissed.value.includes(item.key))
})

const index = computed(() => Math.min(viewIndex.value, Math.max(0, items.value.length - 1)))
const current = computed(() => items.value[index.value] ?? null)

function move(step: number) {
  const total = items.value.length
  if (!total) return
  viewIndex.value = (index.value + step + total) % total
}

function dismiss() {
  const item = current.value
  if (!item) return
  dismissed.value = [...dismissed.value, item.key]
  if (item.preview) {
    clearPreview()
  } else {
    void updateNotifications('dismiss', [{ announcementId: item.announcement.id, phase: 'published' }])
  }
}
</script>

<style lang="less" scoped>
.announcement-banner {
  position: fixed;
  top: 12px;
  right: 12px;
  left: 12px;
  z-index: 100;
  max-width: 768px;
  margin: 0 auto;
  pointer-events: none;
}
</style>
