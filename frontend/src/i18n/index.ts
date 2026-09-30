import { createI18n } from 'vue-i18n'
import enUS from './locales/en-US.ts'
import trTR from './locales/tr-TR.ts'
import { BUILT_IN_DEFAULT, resolveDefaultLocale, SUPPORTED_LOCALES } from './resolveDefaultLocale.ts'

const messages = {
  'en-US': enUS,
  'tr-TR': trTR
}

// User's explicit past choice wins; otherwise use the deployment default.
const savedLocale = localStorage.getItem('locale')
const defaultLocale = resolveDefaultLocale(
  window.__RUNTIME_CONFIG__?.DEFAULT_LOCALE,
  import.meta.env.VITE_DEFAULT_LOCALE,
)
const initialLocale = SUPPORTED_LOCALES.includes(savedLocale as typeof SUPPORTED_LOCALES[number])
  ? savedLocale!
  : defaultLocale

const i18n = createI18n({
  legacy: false,
  locale: initialLocale,
  fallbackLocale: BUILT_IN_DEFAULT,
  globalInjection: true,
  // Some translations intentionally embed `<strong>` markup (e.g. agent step summaries).
  // We render them via v-html with our own sanitization, so silence vue-i18n's HTML warning
  // to avoid flooding the console and slowing renders during history loads.
  warnHtmlMessage: false,
  messages
})

export default i18n
