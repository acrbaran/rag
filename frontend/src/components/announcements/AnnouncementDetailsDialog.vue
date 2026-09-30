<template>
  <t-dialog :visible="visible" :footer="false" width="672px" attach="body" destroy-on-close
    dialog-class-name="announcement-details-dialog" @update:visible="emit('update:visible', $event)"
    @close="emit('update:visible', false)">
    <template #header>
      <span class="announcement-details-dialog__title">{{ title }}</span>
    </template>
    <div class="announcement-details-dialog__body" :class="{ 'has-button': hasButton }">
      <AnnouncementRichText :body="content.body" />
    </div>
    <div v-if="hasButton" class="announcement-details-dialog__action">
      <a class="announcement-button" :href="content.buttonUrl" target="_blank" rel="noopener noreferrer">
        {{ content.buttonLabel }}
      </a>
    </div>
  </t-dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import AnnouncementRichText from './AnnouncementRichText.vue'
import { splitAnnouncementBody } from './announcementBody'

const props = defineProps<{ visible: boolean; title?: string | null; body: string }>()
const emit = defineEmits<{ (e: 'update:visible', value: boolean): void }>()

const content = computed(() => splitAnnouncementBody(props.body))
const hasButton = computed(() => Boolean(content.value.buttonLabel && content.value.buttonUrl))
</script>

<style lang="less">
.announcement-details-dialog {
  position: relative;
  display: flex;
  flex-direction: column;
  max-height: calc(100dvh - 64px);
  overflow: hidden;

  .t-dialog__header {
    flex-shrink: 0;
    padding-right: 32px;
  }

  .t-dialog__body {
    display: flex;
    min-height: 0;
    flex-direction: column;
    padding-top: 16px;
    overflow: hidden;
  }
}

.announcement-details-dialog__title {
  line-height: 1.375;
  overflow-wrap: anywhere;
}

.announcement-details-dialog__body {
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
  color: var(--td-text-color-secondary);
  line-height: 1.625;

  &.has-button {
    padding-bottom: 48px;
  }
}

.announcement-details-dialog__action {
  position: absolute;
  right: 28px;
  bottom: 28px;
}
</style>
