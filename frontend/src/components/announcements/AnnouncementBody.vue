<template>
  <div class="announcement-body">
    <AnnouncementRichText :body="shortened ? `${excerpt.trimEnd()}…` : text" />
  </div>
  <button v-if="shortened" type="button" class="ann-btn ann-btn--link ann-btn--sm announcement-body__more"
    :aria-label="`${t('announcements.readMore')}: ${title ?? ''}`" @pointerdown.stop @click.stop="onReadMoreClick">
    {{ t('announcements.readMore') }}
  </button>
  <AnnouncementDetailsDialog v-if="shortened && !openDetails" v-model:visible="detailsOpen" :title="title"
    :body="text" />
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AnnouncementRichText from './AnnouncementRichText.vue'
import AnnouncementDetailsDialog from './AnnouncementDetailsDialog.vue'
import { announcementExcerpt } from './announcementBody'

const props = defineProps<{
  title?: string | null
  body?: string | null
  onReadMore?: () => void
  openDetails?: () => void
}>()

const { t } = useI18n()
const detailsOpen = ref(false)
const text = computed(() => props.body ?? '')
const excerptInfo = computed(() => announcementExcerpt(text.value))
const excerpt = computed(() => excerptInfo.value.excerpt)
const shortened = computed(() => excerptInfo.value.shortened)

function onReadMoreClick() {
  if (props.openDetails) {
    props.openDetails()
    return
  }
  detailsOpen.value = true
  props.onReadMore?.()
}
</script>

<style lang="less" scoped>
.announcement-body {
  margin-top: 4px;
  color: var(--td-text-color-secondary);
  line-height: 1.625;
}

.announcement-body__more {
  height: auto;
  min-height: 36px;
  margin-top: 4px;
  padding: 0;
}
</style>
