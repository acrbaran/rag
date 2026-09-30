<template>
  <div role="alert" aria-live="polite" aria-atomic="true" class="ann-alert ann-alert--icon announcement-banner-card"
    :class="[`announcement-banner-card--${bannerColor}`, controls ? 'has-controls' : '']">
    <LucideIcon :name="icon" />
    <div class="ann-alert__title announcement-banner-card__title" :data-announcement-id="announcementId || undefined"
      :data-announcement-phase="announcementId ? 'published' : undefined">
      {{ title }}
    </div>
    <div class="ann-alert__description announcement-banner-card__description" :class="{ 'has-button': hasButton }">
      <AnnouncementRichText :body="content.body" />
      <template v-if="unavailable"> · {{ t('announcements.connectionLost') }}</template>
    </div>
    <div v-if="hasButton" class="announcement-banner-card__button-row">
      <a class="announcement-button" :href="content.buttonUrl" target="_blank" rel="noopener noreferrer">
        {{ content.buttonLabel }}
      </a>
    </div>
    <div class="announcement-banner-card__actions">
      <template v-if="controls">
        <button type="button" class="ann-btn ann-btn--ghost ann-btn--icon-sm" :aria-label="t('announcements.previous')"
          :disabled="!hasPrevious" @click="emit('previous')">
          <LucideIcon name="chevron-left" />
        </button>
        <span class="announcement-banner-card__position"
          :aria-label="t('announcements.position', { current: position, total })">
          {{ position }}/{{ total }}
        </span>
        <button type="button" class="ann-btn ann-btn--ghost ann-btn--icon-sm" :aria-label="t('announcements.next')"
          :disabled="!hasNext" @click="emit('next')">
          <LucideIcon name="chevron-right" />
        </button>
      </template>
      <button v-if="dismissible" type="button" class="ann-btn ann-btn--ghost ann-btn--icon-sm"
        :aria-label="t('announcements.dismiss')" :disabled="dismissDisabled" @click="emit('dismiss')">
        <LucideIcon name="x" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  announcementTone,
  type AnnouncementBannerColor,
  type AnnouncementTone,
  type AnnouncementType,
} from '@/api/announcements'
import AnnouncementRichText from './AnnouncementRichText.vue'
import LucideIcon from './LucideIcon.vue'
import { splitAnnouncementBody } from './announcementBody'

const props = withDefaults(defineProps<{
  title: string
  body: string
  type?: AnnouncementType
  bannerColor?: AnnouncementBannerColor
  announcementId?: string
  unavailable?: boolean
  position?: number
  total?: number
  hasPrevious?: boolean
  hasNext?: boolean
  dismissible?: boolean
  dismissDisabled?: boolean
}>(), {
  bannerColor: 'neutral',
  unavailable: false,
  position: 1,
  total: 1,
  hasPrevious: false,
  hasNext: false,
  dismissible: false,
  dismissDisabled: false,
})

const emit = defineEmits<{ (e: 'previous'): void; (e: 'next'): void; (e: 'dismiss'): void }>()
const { t } = useI18n()

const ICONS: Record<AnnouncementTone, string> = {
  maintenance: 'wrench',
  warning: 'triangle-alert',
  info: 'info',
  update: 'sparkles',
  custom: 'circle-alert',
}

const icon = computed(() => ICONS[announcementTone(props.type?.id)])
const controls = computed(() => props.total > 1)
const content = computed(() => splitAnnouncementBody(props.body))
const hasButton = computed(() => Boolean(content.value.buttonLabel && content.value.buttonUrl))
</script>

<style lang="less" scoped>
.announcement-banner-card {
  pointer-events: auto;
  row-gap: 8px;
  padding-right: 56px;
  box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -4px rgba(0, 0, 0, 0.1);
  backdrop-filter: blur(8px);

  &.has-controls {
    padding-right: 160px;
  }

  &--neutral {
    background: color-mix(in srgb, var(--td-bg-color-page) 95%, transparent);
    color: var(--td-text-color-primary);
  }

  &--blue {
    border-color: color-mix(in srgb, #0ea5e9 40%, transparent);
    background: color-mix(in srgb, #f0f9ff 95%, transparent);
    color: #082f49;
  }

  &--green {
    border-color: color-mix(in srgb, #10b981 40%, transparent);
    background: color-mix(in srgb, #ecfdf5 95%, transparent);
    color: #022c22;
  }

  &--amber {
    border-color: color-mix(in srgb, #f59e0b 40%, transparent);
    background: color-mix(in srgb, #fffbeb 95%, transparent);
    color: #451a03;
  }

  &--red {
    border-color: color-mix(in srgb, #ef4444 40%, transparent);
    background: color-mix(in srgb, #fef2f2 95%, transparent);
    color: #450a0a;
  }

  &--violet {
    border-color: color-mix(in srgb, #8b5cf6 40%, transparent);
    background: color-mix(in srgb, #f5f3ff 95%, transparent);
    color: #2e1065;
  }
}

:root[theme-mode='dark'] .announcement-banner-card {
  &--blue {
    border-color: color-mix(in srgb, #38bdf8 30%, transparent);
    background: color-mix(in srgb, #082f49 95%, transparent);
    color: #f0f9ff;
  }

  &--green {
    border-color: color-mix(in srgb, #34d399 30%, transparent);
    background: color-mix(in srgb, #022c22 95%, transparent);
    color: #ecfdf5;
  }

  &--amber {
    border-color: color-mix(in srgb, #fbbf24 30%, transparent);
    background: color-mix(in srgb, #451a03 95%, transparent);
    color: #fffbeb;
  }

  &--red {
    border-color: color-mix(in srgb, #f87171 30%, transparent);
    background: color-mix(in srgb, #450a0a 95%, transparent);
    color: #fef2f2;
  }

  &--violet {
    border-color: color-mix(in srgb, #a78bfa 30%, transparent);
    background: color-mix(in srgb, #2e1065 95%, transparent);
    color: #f5f3ff;
  }
}

.announcement-banner-card__title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.announcement-banner-card__description {
  width: 100%;
  max-height: min(50vh, 24rem);
  overflow-y: auto;
  color: currentColor;
  white-space: pre-wrap;
  word-break: break-word;
  opacity: 0.75;

  &.has-button {
    padding-bottom: 56px;
  }

  :deep(.announcement-rich-text) {
    color: currentColor;
    white-space: normal;
  }
}

.announcement-banner-card__button-row {
  position: absolute;
  right: 0;
  bottom: 12px;
  left: 0;
  display: flex;
  justify-content: flex-end;
  padding: 0 16px;
  pointer-events: none;

  .announcement-button {
    pointer-events: auto;
  }
}

.announcement-banner-card__actions {
  position: absolute;
  top: 8px;
  right: 12px;
  display: flex;
  align-items: flex-start;
  gap: 2px;

  .ann-btn {
    color: currentColor;
  }
}

.announcement-banner-card__position {
  min-width: 36px;
  font-size: var(--app-text-xs);
  font-weight: 500;
  line-height: 32px;
  text-align: center;
  font-variant-numeric: tabular-nums;
}
</style>
