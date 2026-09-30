<template>
  <svg xmlns="http://www.w3.org/2000/svg" :width="size" :height="size" viewBox="0 0 24 24" fill="none"
    stroke="currentColor" :stroke-width="strokeWidth" stroke-linecap="round" stroke-linejoin="round"
    class="lucide-icon" aria-hidden="true">
    <component :is="tag" v-for="([tag, attrs], index) in nodes" :key="index" v-bind="attrs" />
  </svg>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { LUCIDE_ICONS } from './lucideIcons'

const props = withDefaults(defineProps<{ name: string; size?: number | string; strokeWidth?: number | string }>(), {
  size: 16,
  strokeWidth: 2,
})

const nodes = computed(() => LUCIDE_ICONS[props.name] ?? [])
</script>

<style scoped>
.lucide-icon {
  flex-shrink: 0;
}
</style>
