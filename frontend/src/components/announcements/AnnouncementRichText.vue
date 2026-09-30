<template>
  <!-- eslint-disable-next-line vue/no-v-html -->
  <div class="announcement-rich-text" v-html="html" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { renderAnnouncementMarkdown } from './announcementBody'

const props = defineProps<{ body: string }>()

// Çıktı DOMPurify ile temizlenir; bağlantılar yalnızca http(s) olabilir.
const html = computed(() => renderAnnouncementMarkdown(props.body))
</script>
