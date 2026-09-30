<template>
  <nav v-if="pageCount >= 1" role="navigation" class="numeric-pagination"
    :aria-label="label || t('announcements.pagination.label')">
    <ul class="numeric-pagination__list">
      <li v-for="token in tokens" :key="token">
        <button v-if="typeof token === 'number'" type="button" class="ann-btn ann-btn--icon-sm"
          :class="token === page ? 'ann-btn--outline' : 'ann-btn--ghost'" :disabled="disabled"
          :aria-current="token === page ? 'page' : undefined"
          :aria-label="t('announcements.pagination.page', { page: token })" @click="emit('page', token)">
          {{ token }}
        </button>
        <span v-else class="numeric-pagination__ellipsis" aria-hidden="true">
          <LucideIcon name="ellipsis" />
        </span>
      </li>
    </ul>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import LucideIcon from './LucideIcon.vue'

const props = defineProps<{ page: number; pageCount: number; label?: string; disabled?: boolean }>()
const emit = defineEmits<{ (e: 'page', page: number): void }>()
const { t } = useI18n()

function pageTokens(current: number, total: number): (number | string)[] {
  if (total <= 7) return Array.from({ length: total }, (_, index) => index + 1)
  const pages = [...new Set([1, current - 1, current, current + 1, total])]
    .filter((value) => value >= 1 && value <= total)
    .sort((a, b) => a - b)
  const tokens: (number | string)[] = []
  pages.forEach((value, index) => {
    const previous = pages[index - 1]
    if (previous !== undefined && value - previous > 1) tokens.push(`ellipsis-${previous}`)
    tokens.push(value)
  })
  return tokens
}

const tokens = computed(() => pageTokens(props.page, props.pageCount))
</script>

<style lang="less" scoped>
.numeric-pagination {
  display: flex;
  width: auto;
  max-width: 100%;
  justify-content: center;
}

.numeric-pagination__list {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.numeric-pagination__ellipsis {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  color: var(--td-text-color-primary);
}
</style>
