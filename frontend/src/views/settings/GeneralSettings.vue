<template>
  <div class="general-settings">
    <div class="section-header">
      <h2>{{ $t('general.title') }}</h2>
      <p class="section-description">{{ $t('general.description') }}</p>
    </div>

    <div class="settings-group">
      <!-- Dil seçimi-->
      <div class="setting-row">
        <div class="setting-info">
          <label>{{ $t('language.language') }}</label>
          <p class="desc">{{ $t('language.languageDescription') }}</p>
        </div>
        <div class="setting-control">
          <t-select
            v-model="localLanguage"
            :placeholder="$t('language.selectLanguage')"
            @change="handleLanguageChange"
          >
            <t-option value="en-US" :label="$t('language.enUS')">{{ $t('language.enUS') }}</t-option>
            <t-option value="tr-TR" label="Türkçe">Türkçe</t-option>
          </t-select>
        </div>
      </div>

      <!-- Tema ayarları-->
      <div class="setting-row">
        <div class="setting-info">
          <label>{{ $t('theme.theme') }}</label>
          <p class="desc">{{ $t('theme.themeDescription') }}</p>
        </div>
        <div class="setting-control">
          <t-select
            v-model="localTheme"
            :placeholder="$t('theme.selectTheme')"
            @change="handleThemeChange"
          >
            <t-option value="light" :label="$t('theme.light')">{{ $t('theme.light') }}</t-option>
            <t-option value="dark" :label="$t('theme.dark')">{{ $t('theme.dark') }}</t-option>
            <t-option value="system" :label="$t('theme.system')">{{ $t('theme.system') }}</t-option>
          </t-select>
        </div>
      </div>

      <!-- Arayüz yazı tipi -->
      <div class="setting-row">
        <div class="setting-info">
          <label>{{ $t('font.uiFont') }}</label>
          <p class="desc">{{ $t('font.uiFontDescription') }}</p>
        </div>
        <div class="setting-control">
          <t-select
            v-model="localSansFont"
            :placeholder="$t('font.selectFont')"
            @change="handleSansFontChange"
          >
            <t-option
              v-for="opt in sansFontOptions"
              :key="opt.value"
              :value="opt.value"
              :label="opt.label"
            >
              <span :style="{ fontFamily: opt.preview }">{{ opt.label }}</span>
            </t-option>
          </t-select>
        </div>
      </div>

      <!-- Kod yazı tipi -->
      <div class="setting-row">
        <div class="setting-info">
          <label>{{ $t('font.monoFont') }}</label>
          <p class="desc">{{ $t('font.monoFontDescription') }}</p>
        </div>
        <div class="setting-control">
          <t-select
            v-model="localMonoFont"
            :placeholder="$t('font.selectFont')"
            @change="handleMonoFontChange"
          >
            <t-option
              v-for="opt in monoFontOptions"
              :key="opt.value"
              :value="opt.value"
              :label="opt.label"
            >
              <span :style="{ fontFamily: opt.preview }">{{ opt.label }}</span>
            </t-option>
          </t-select>
        </div>
      </div>

      <!-- Yazı tipi boyutu -->
      <div class="setting-row">
        <div class="setting-info">
          <label>{{ $t('font.fontSize') }}</label>
          <p class="desc">{{ $t('font.fontSizeDescription') }}</p>
        </div>
        <div class="setting-control">
          <t-radio-group
            v-model="localFontSize"
            @change="handleFontSizeChange"
          >
            <t-radio-button value="small">{{ $t('font.size.small') }}</t-radio-button>
            <t-radio-button value="normal">{{ $t('font.size.normal') }}</t-radio-button>
            <t-radio-button value="large">{{ $t('font.size.large') }}</t-radio-button>
          </t-radio-group>
        </div>
      </div>

    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useTheme, type ThemeMode } from '@/composables/useTheme'
import {
  useFont,
  SANS_STACKS,
  MONO_STACKS,
  visibleSansKeys,
  visibleMonoKeys,
  type FontKey,
  type MonoFontKey,
  type FontSizeKey,
} from '@/composables/useFont'

const { t, locale } = useI18n()
const authStore = useAuthStore()
const { currentTheme, setTheme } = useTheme()
const {
  currentSans,
  currentMono,
  currentSize,
  setSansFont,
  setMonoFont,
  setFontSize,
} = useFont()

// Yerel durum
const localLanguage = ref('tr-TR')
const localTheme = ref<ThemeMode>(currentTheme.value)
const localSansFont = ref<FontKey>(currentSans.value)
const localMonoFont = ref<MonoFontKey>(currentMono.value)
const localFontSize = ref<FontSizeKey>(currentSize.value)

// Keep the form in sync if preferences change externally (e.g. on user switch).
watch(currentTheme, (val) => { localTheme.value = val })
watch(currentSans, (val) => { localSansFont.value = val })
watch(currentMono, (val) => { localMonoFont.value = val })
watch(currentSize, (val) => { localFontSize.value = val })

const sansFontOptions = computed<{ value: FontKey; label: string; preview: string }[]>(() =>
  visibleSansKeys().map((key) => ({
    value: key,
    label: t(`font.sans.${key}`),
    preview: SANS_STACKS[key],
  })),
)

const monoFontOptions = computed<{ value: MonoFontKey; label: string; preview: string }[]>(() =>
  visibleMonoKeys().map((key) => ({
    value: key,
    label: t(`font.mono.${key}`),
    preview: MONO_STACKS[key],
  })),
)

// Başlangıç yüklemesi
onMounted(() => {
  // Dil ayarlarını localStorage'dan yükle
  localLanguage.value = locale.value
})

// Dil değişikliğini işle
const handleLanguageChange = () => {
  locale.value = localLanguage.value
  localStorage.setItem('locale', localLanguage.value)
  MessagePlugin.success(t('language.languageSaved'))
    }

// Tema değişikliğini işle
const handleThemeChange = (val: ThemeMode) => {
  if (!setTheme(val)) {
    // Setter rejected the value (validation guard); roll the form back to
    // the canonical state so the UI doesn't drift.
    localTheme.value = currentTheme.value
    return
  }
}

// Yazı tipi değişikliğini işle
const handleSansFontChange = (val: FontKey) => {
  if (!setSansFont(val)) {
    localSansFont.value = currentSans.value
    return
  }
}

const handleMonoFontChange = (val: MonoFontKey) => {
  if (!setMonoFont(val)) {
    localMonoFont.value = currentMono.value
    return
  }
}

const handleFontSizeChange = (val: FontSizeKey) => {
  if (!setFontSize(val)) {
    localFontSize.value = currentSize.value
    return
  }
}
</script>

<style lang="less" scoped>
@import (reference) '@/components/css/settings-section.less';

.general-settings {
  width: 100%;
}

.section-header {
  .settings-section-header();
}

.settings-group {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.setting-row {
  .setting-row();
}

.setting-info {
  .setting-info();
}

.setting-control {
  .setting-control();
}

</style>
