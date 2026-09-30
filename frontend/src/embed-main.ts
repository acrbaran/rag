import { computed, createApp, h, watch } from 'vue'
import { createPinia } from 'pinia'
import { createRouter, createWebHistory, RouterView } from 'vue-router'
import TDesign, { ConfigProvider } from 'tdesign-vue-next'
import enUSConfig from 'tdesign-vue-next/esm/locale/en_US'
import 'tdesign-vue-next/es/style/index.css'
import '@/assets/theme/theme.css'
import { installTDesignIconOfflineGuard } from '@/utils/tdesign-icon-offline'
import trTRConfig from '@/i18n/tdesignTr'
import appI18n from '@/i18n'
import i18n from './i18n/embed'
import EmbedPage from '@/views/embed/EmbedPage.vue'
import ProtectedResourcePreview from '@/components/ProtectedResourcePreview.vue'

installTDesignIconOfflineGuard()

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/embed/:channelId',
      name: 'embed',
      component: EmbedPage,
    },
  ],
})

// TDesign components default to zh_CN copy; follow the embed locale instead.
const tdLocaleMap: Record<string, object> = {
  'en-US': enUSConfig,
  'tr-TR': trTRConfig,
}
const tdGlobalConfig = computed(() => tdLocaleMap[i18n.global.locale.value] || trTRConfig)

// Shared API helpers (axios interceptor, SSE stream) read the main app i18n to
// build Accept-Language. Mirror the embed locale there so backend-localized
// fields (e.g. built-in agent names) match what the visitor sees.
watch(
  () => i18n.global.locale.value,
  (next) => {
    if (next && appI18n.global.locale.value !== next) {
      appI18n.global.locale.value = next
    }
    // Keep <html lang> in sync so screen readers, hyphenation and spellcheck
    // follow the visitor-facing language.
    if (next && typeof document !== 'undefined') {
      document.documentElement.lang = next.split('-')[0]
    }
  },
  { immediate: true },
)

// Runtime-only Vue build cannot compile string templates — use a render fn.
const app = createApp({
  render: () =>
    h(ConfigProvider, { globalConfig: tdGlobalConfig.value }, {
      default: () => [h(RouterView), h(ProtectedResourcePreview)],
    }),
})

app.use(TDesign)
app.use(createPinia())
app.use(router)
app.use(i18n)

router.isReady().finally(() => {
  app.mount('#embed-app')
})
