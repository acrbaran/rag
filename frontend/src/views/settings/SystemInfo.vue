<template>
  <div class="system-info">
    <div class="section-header">
      <h2>{{ $t('system.title') }}</h2>
      <p class="section-description">{{ $t('system.sectionDescription') }}</p>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="shell" aria-busy="true" :aria-label="$t('system.loadingInfo')">
      <div class="core hero hero--skeleton">
        <t-skeleton
          animation="gradient"
          :row-col="[{ width: '120px', height: '22px' }, { width: '220px', height: '48px' }, { width: '260px' }]"
        />
      </div>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="error-inline">
      <t-alert theme="error" :message="error">
        <template #operation>
          <t-button size="small" @click="loadInfo">{{ $t('system.retry') }}</t-button>
        </template>
      </t-alert>
    </div>

    <template v-else>
      <!-- Release hero -->
      <section class="shell" style="--i: 0" aria-labelledby="system-release-version">
        <div class="core hero">
          <div class="hero-main">
            <div class="hero-eyebrow">
              <span class="eyebrow-pill">Rethra</span>
              <span v-if="editionLabel" class="edition-pill" :class="{ 'is-lite': isLite }">{{ editionLabel }}</span>
            </div>

            <h3 id="system-release-version" class="hero-version">
              <template v-if="appVersion">
                <span class="hero-version-prefix" aria-hidden="true">v</span>{{ appVersion }}
              </template>
              <template v-else>{{ $t('system.unknown') }}</template>
            </h3>

            <p class="hero-status" :class="`is-${status}`" role="status">
              <span class="status-dot" aria-hidden="true" />
              <span>{{ statusText }}</span>
            </p>
          </div>

          <button type="button" class="island-btn" @click="copyDiagnostics">
            <span>{{ $t('system.copyDiagnostics') }}</span>
            <span class="island-btn-icon" aria-hidden="true"><t-icon name="copy" /></span>
          </button>
        </div>
      </section>

      <!-- Components -->
      <section class="shell" style="--i: 1" aria-labelledby="system-components-title">
        <div class="core panel">
          <header class="panel-head">
            <span class="panel-icon" aria-hidden="true"><t-icon name="layers" /></span>
            <div class="panel-titles">
              <h3 id="system-components-title">{{ $t('system.componentsTitle') }}</h3>
              <p>{{ $t('system.componentsHint') }}</p>
            </div>
          </header>

          <ul class="component-list">
            <li v-for="item in components" :key="item.key" class="component-row">
              <span class="component-icon" aria-hidden="true"><t-icon :name="item.icon" /></span>

              <div class="component-info">
                <span class="component-name">{{ item.label }}</span>
                <span class="component-desc">{{ item.description }}</span>
              </div>

              <div class="component-meta">
                <span v-if="item.mismatch" class="mismatch-pill">
                  <t-icon name="error-circle" aria-hidden="true" />
                  {{ $t('system.versionMismatch') }}
                </span>
                <span class="version-pill">{{ item.version ? `v${item.version}` : $t('system.unknown') }}</span>
              </div>
            </li>
          </ul>
        </div>
      </section>

      <!-- DB migration error -->
      <section
        v-if="systemInfo?.db_migration_error"
        class="shell shell--danger"
        style="--i: 2"
        aria-labelledby="system-migration-title"
      >
        <div class="core panel migration">
          <header class="panel-head">
            <span class="panel-icon panel-icon--danger" aria-hidden="true"><t-icon name="error-circle" /></span>
            <div class="panel-titles">
              <h3 id="system-migration-title">{{ $t('system.dbMigrationFailedTitle') }}</h3>
              <p>{{ $t('system.dbMigrationFailedDesc') }}</p>
            </div>
          </header>

          <pre class="migration-detail">{{ systemInfo.db_migration_error }}</pre>

          <div class="migration-actions">
            <a class="island-btn island-btn--danger" :href="reportIssueURL" target="_blank" rel="noopener noreferrer">
              <span>{{ $t('system.dbMigrationReportIssue') }}</span>
              <span class="island-btn-icon" aria-hidden="true"><t-icon name="arrow-right-up" /></span>
            </a>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import type { SystemInfo } from '@/api/system'
import { useEditorResourcesStore } from '@/stores/editorResources'
import { copyWithToast } from '@/utils/clipboard'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

// Reactive state
const editorResources = useEditorResourcesStore()
const systemInfo = ref<SystemInfo | null>(null)
const loading = ref(true)
const error = ref('')
const frontendVersion = __FRONTEND_VERSION__
const frontendCommit = __FRONTEND_COMMIT__

type ReleaseStatus = 'healthy' | 'mismatch' | 'failed' | 'unknown'

/** "v0.8.2" and "0.8.2" describe the same release; "unknown" means no value. */
const normalizeVersion = (value?: string | null) => {
  const trimmed = (value || '').trim()
  if (!trimmed || trimmed.toLowerCase() === 'unknown') return ''
  return trimmed.replace(/^v(?=\d)/i, '')
}

const appVersion = computed(() => normalizeVersion(systemInfo.value?.version))
const uiVersion = normalizeVersion(frontendVersion)

const isLite = computed(() => systemInfo.value?.edition === 'lite')
const editionLabel = computed(() => {
  if (!systemInfo.value?.edition) return ''
  return isLite.value ? 'Lite' : 'Standard'
})

const isMismatch = computed(() => !!appVersion.value && !!uiVersion && appVersion.value !== uiVersion)

const status = computed<ReleaseStatus>(() => {
  if (systemInfo.value?.db_migration_error) return 'failed'
  if (isMismatch.value) return 'mismatch'
  if (!appVersion.value || !uiVersion) return 'unknown'
  return 'healthy'
})

const statusText = computed(() => {
  switch (status.value) {
    case 'failed':
      return t('system.statusMigrationFailed')
    case 'mismatch':
      return t('system.statusMismatch')
    case 'unknown':
      return t('system.statusUnknown')
    default:
      return t('system.statusHealthy')
  }
})

const components = computed(() => [
  {
    key: 'app',
    icon: 'server',
    label: t('system.versionLabel'),
    description: t('system.versionDescription'),
    version: appVersion.value,
    mismatch: false,
  },
  {
    key: 'ui',
    icon: 'desktop',
    label: t('system.frontendVersionLabel'),
    description: t('system.frontendVersionDescription'),
    version: uiVersion,
    mismatch: isMismatch.value,
  },
])

const formatVersion = (version: string) => (version ? `v${version}` : 'unknown')

const copyDiagnostics = () => {
  const lines = [
    `Rethra ${formatVersion(appVersion.value)}${editionLabel.value ? ` · ${editionLabel.value}` : ''}`,
    `rethra-app: ${formatVersion(appVersion.value)}`,
    `rethra-ui: ${formatVersion(uiVersion)}`,
  ]
  if (systemInfo.value?.db_version) lines.push(`db: ${systemInfo.value.db_version}`)
  if (systemInfo.value?.db_migration_error) lines.push(`db_migration_error: ${systemInfo.value.db_migration_error}`)
  copyWithToast(lines.join('\n'), 'system.diagnosticsCopied')
}

// Pre-fills a new issue with the current migration error so users don't have to
// paste it manually. Body is intentionally minimal — the bug template will fill
// in the rest. Encode aggressively to survive newlines / quotes.
const reportIssueURL = computed(() => {
  const base = 'https://github.com/acrbaran/rag/issues/new'
  const params = new URLSearchParams({
    template: 'bug_report.yml',
    title: '[Bug]: Database migration failed at startup',
    labels: 'bug',
  })
  const errMsg = systemInfo.value?.db_migration_error
  if (errMsg) {
    const body = [
      '### Environment',
      `- Rethra version: ${systemInfo.value?.version || 'unknown'}`,
      `- Commit: ${systemInfo.value?.commit_id || 'unknown'}`,
      `- Frontend version: ${frontendVersion} (${frontendCommit})`,
      `- DB version reported: ${systemInfo.value?.db_version || 'unknown'}`,
      '',
      '### Migration error',
      '```',
      errMsg,
      '```',
    ].join('\n')
    params.set('body', body)
  }
  return `${base}?${params.toString()}`
})

// Methods
const loadInfo = async () => {
  try {
    loading.value = true
    error.value = ''

    // Ayarlar paneli her açıldığında en güncel değerleri göstermeli; kenar çubuğu başlatılırken alınan anlık görüntü aynı store üzerinden kullanılmalı.
    await editorResources.ensureSystemInfo(true)

    if (editorResources.systemInfo) {
      systemInfo.value = editorResources.systemInfo
    } else {
      error.value = t('system.messages.fetchFailed')
    }
  } catch (err: any) {
    error.value = err?.message || t('system.messages.networkError')
  } finally {
    loading.value = false
  }
}

// Lifecycle
onMounted(() => {
  loadInfo()
})
</script>

<style lang="less" scoped>
@import (reference) '@/components/css/settings-section.less';

@ease: cubic-bezier(0.32, 0.72, 0, 1);
@shell-pad: 5px;

.system-info {
  width: 100%;
  container-type: inline-size;
  display: flex;
  flex-direction: column;
  gap: var(--app-space-4);

  --si-shell: var(--app-surface-muted);
  --si-card: var(--td-bg-color-container);
  --si-ring: color-mix(in srgb, var(--td-component-stroke) 70%, transparent);
  --si-highlight: inset 0 1px 0 rgba(255, 255, 255, 0.9);
  --si-lift: 0 1px 2px rgba(0, 0, 0, 0.03), 0 10px 28px -18px rgba(0, 0, 0, 0.14);
}

:root[theme-mode='dark'] .system-info {
  --si-shell: #1a1a1a;
  --si-card: #242424;
  --si-ring: rgba(255, 255, 255, 0.07);
  --si-highlight: inset 0 1px 0 rgba(255, 255, 255, 0.05);
  --si-lift: 0 1px 2px rgba(0, 0, 0, 0.3), 0 12px 32px -18px rgba(0, 0, 0, 0.7);
}

.section-header {
  .settings-section-header();
  margin-bottom: var(--app-space-2);
}

.error-inline {
  padding: var(--app-space-2) 0;
}

/* ---------- Double-bezel surfaces ---------- */

.shell {
  padding: @shell-pad;
  border-radius: var(--app-radius-2xl);
  background: var(--si-shell);
  box-shadow: 0 0 0 1px var(--si-ring);
  animation: system-rise 620ms @ease both;
  animation-delay: calc(var(--i, 0) * 80ms);

  &--danger {
    background: color-mix(in srgb, var(--td-error-color) 6%, var(--si-shell));
    box-shadow: 0 0 0 1px color-mix(in srgb, var(--td-error-color) 18%, transparent);
  }
}

.core {
  border-radius: calc(var(--app-radius-2xl) - @shell-pad);
  background: var(--si-card);
  box-shadow: 0 0 0 1px var(--si-ring), var(--si-highlight), var(--si-lift);
}

/* ---------- Release hero ---------- */

.hero {
  position: relative;
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: var(--app-space-6);
  padding: var(--app-space-8) var(--app-space-8) var(--app-space-6);
  overflow: hidden;
  background:
    radial-gradient(90% 140% at 100% 0%, color-mix(in srgb, var(--td-brand-color) 10%, transparent) 0%, transparent 60%),
    radial-gradient(60% 90% at 0% 100%, color-mix(in srgb, var(--td-brand-color) 5%, transparent) 0%, transparent 70%),
    var(--si-card);

  // Fine engraved baseline grid, fades toward the left.
  &::before {
    content: '';
    position: absolute;
    inset: 0;
    background-image: linear-gradient(to right, var(--si-ring) 1px, transparent 1px);
    background-size: 28px 100%;
    mask-image: linear-gradient(to left, rgba(0, 0, 0, 0.55), transparent 65%);
    pointer-events: none;
  }

  > * {
    position: relative;
  }

  &--skeleton {
    display: block;
  }
}

.hero-main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: var(--app-space-3);
}

.hero-eyebrow {
  display: flex;
  align-items: center;
  gap: 6px;
}

.eyebrow-pill,
.edition-pill {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 10px;
  border-radius: var(--app-radius-pill);
  font-size: var(--app-text-xs);
  font-weight: 600;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  line-height: 1;
}

.eyebrow-pill {
  background: var(--si-shell);
  box-shadow: inset 0 0 0 1px var(--si-ring);
  color: var(--td-text-color-secondary);
}

.edition-pill {
  background: color-mix(in srgb, var(--td-text-color-primary) 6%, transparent);
  color: var(--td-text-color-primary);

  &.is-lite {
    background: color-mix(in srgb, var(--td-brand-color) 12%, transparent);
    color: var(--td-brand-color-7);
  }
}

.hero-version {
  margin: 0;
  font-family: var(--app-font-heading);
  font-size: clamp(36px, 7cqi, 52px);
  font-weight: 600;
  line-height: 1;
  letter-spacing: -0.035em;
  font-variant-numeric: tabular-nums;
  color: var(--td-text-color-primary);
  overflow-wrap: anywhere;
}

.hero-version-prefix {
  margin-right: 0.04em;
  font-weight: 400;
  color: var(--td-text-color-placeholder);
}

.hero-status {
  display: inline-flex;
  align-items: center;
  gap: var(--app-space-2);
  margin: 0;
  font-size: var(--app-text-md);
  color: var(--td-text-color-secondary);

  --dot: var(--td-text-color-placeholder);

  &.is-healthy {
    --dot: var(--td-success-color);
  }

  &.is-mismatch {
    --dot: var(--td-warning-color);
  }

  &.is-failed {
    --dot: var(--td-error-color);
  }
}

.status-dot {
  position: relative;
  flex: none;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--dot);

  &::after {
    content: '';
    position: absolute;
    inset: 0;
    border-radius: 50%;
    background: var(--dot);
    animation: status-ping 2.4s @ease infinite;
  }

  .is-unknown &::after {
    display: none;
  }
}

/* ---------- Island button (button-in-button) ---------- */

.island-btn {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: var(--app-space-3);
  height: 40px;
  padding: 0 4px 0 var(--app-space-4);
  border: 0;
  border-radius: var(--app-radius-pill);
  background: var(--td-text-color-primary);
  color: var(--td-text-color-anti);
  font-family: inherit;
  font-size: var(--app-text-md);
  font-weight: 500;
  text-decoration: none;
  white-space: nowrap;
  cursor: pointer;
  transition: transform 420ms @ease, background-color var(--app-motion-base) @ease;

  &:hover {
    background: color-mix(in srgb, var(--td-text-color-primary) 88%, transparent);
  }

  &:hover .island-btn-icon {
    transform: translate(2px, -1px) scale(1.05);
  }

  &:active {
    transform: scale(0.98);
  }

  &:focus-visible {
    outline: 2px solid var(--app-ring-soft);
    outline-offset: 2px;
  }

  &--danger {
    background: var(--td-error-color);
    color: #fff;

    &:hover {
      background: var(--td-error-color-hover);
      color: #fff;
    }
  }
}

.island-btn-icon {
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.14);
  font-size: var(--app-text-lg);
  transition: transform 520ms @ease;
}

:root[theme-mode='dark'] .island-btn:not(.island-btn--danger) .island-btn-icon {
  background: rgba(0, 0, 0, 0.08);
}

/* ---------- Panels ---------- */

.panel {
  display: flex;
  flex-direction: column;
  gap: var(--app-space-5);
  padding: var(--app-space-6);
  min-width: 0;
}

.panel-head {
  display: flex;
  align-items: flex-start;
  gap: var(--app-space-3);
}

.panel-icon {
  flex: none;
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  border-radius: var(--app-radius-md);
  background: var(--si-shell);
  box-shadow: inset 0 0 0 1px var(--si-ring);
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xl);

  &--danger {
    background: color-mix(in srgb, var(--td-error-color) 10%, transparent);
    box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--td-error-color) 22%, transparent);
    color: var(--td-error-color);
  }
}

.panel-titles {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding-top: 1px;

  h3 {
    margin: 0;
    font-family: var(--app-font-heading);
    font-size: var(--app-text-lg);
    font-weight: 600;
    line-height: 1.35;
    color: var(--td-text-color-primary);
  }

  p {
    margin: 0;
    max-width: 68ch;
    font-size: var(--app-text-sm);
    line-height: 1.55;
    color: var(--td-text-color-secondary);
  }
}

/* ---------- Component list ---------- */

.component-list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
}

.component-row {
  display: flex;
  align-items: center;
  gap: var(--app-space-4);
  padding: var(--app-space-4) 0;

  & + & {
    border-top: 1px dashed var(--si-ring);
  }

  &:first-child {
    padding-top: 0;
  }

  &:last-child {
    padding-bottom: 0;
  }
}

.component-icon {
  flex: none;
  display: grid;
  place-items: center;
  width: 40px;
  height: 40px;
  border-radius: var(--app-radius-md);
  background: linear-gradient(160deg, var(--si-card), var(--si-shell));
  box-shadow: inset 0 0 0 1px var(--si-ring), var(--si-highlight);
  color: var(--td-text-color-primary);
  font-size: var(--app-text-2xl);
}

.component-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.component-name {
  font-size: var(--app-text-base);
  font-weight: 500;
  color: var(--td-text-color-primary);
}

.component-desc {
  font-size: var(--app-text-sm);
  line-height: 1.5;
  color: var(--td-text-color-secondary);
}

.component-meta {
  flex: none;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: 6px;
}

.version-pill,
.mismatch-pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 26px;
  padding: 0 10px;
  border-radius: var(--app-radius-pill);
  font-size: var(--app-text-sm);
  white-space: nowrap;
}

.version-pill {
  background: color-mix(in srgb, var(--td-text-color-primary) 5%, transparent);
  font-family: var(--app-font-family-mono);
  font-weight: 500;
  font-variant-numeric: tabular-nums;
  color: var(--td-text-color-primary);
}

.mismatch-pill {
  background: color-mix(in srgb, var(--td-warning-color) 12%, transparent);
  color: var(--td-warning-color-6);
  font-weight: 500;

  .t-icon {
    font-size: var(--app-text-md);
  }
}

/* ---------- Migration error ---------- */

.migration-detail {
  margin: 0;
  padding: var(--app-space-3) var(--app-space-4);
  max-height: 220px;
  overflow: auto;
  border-radius: var(--app-radius-md);
  background: var(--si-shell);
  box-shadow: inset 0 0 0 1px var(--si-ring);
  font-family: var(--app-font-family-mono);
  font-size: var(--app-text-sm);
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--td-text-color-secondary);
}

.migration-actions {
  display: flex;
  justify-content: flex-end;
}

/* ---------- Motion ---------- */

@keyframes system-rise {
  from {
    opacity: 0;
    transform: translateY(12px);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes status-ping {
  0% {
    opacity: 0.55;
    transform: scale(1);
  }

  70%,
  100% {
    opacity: 0;
    transform: scale(2.6);
  }
}

/* ---------- Narrow layouts ---------- */

@container (max-width: 640px) {
  .component-row {
    flex-wrap: wrap;
  }

  .component-meta {
    width: 100%;
    justify-content: flex-start;
    padding-left: calc(40px + var(--app-space-4));
  }
}

@container (max-width: 560px) {
  .hero {
    flex-direction: column;
    align-items: stretch;
    padding: var(--app-space-6) var(--app-space-5) var(--app-space-5);
  }

  .island-btn {
    justify-content: space-between;
  }

  .panel {
    padding: var(--app-space-5);
  }

  .component-meta {
    padding-left: 0;
  }

  .migration-actions .island-btn {
    width: 100%;
  }
}

@media (prefers-reduced-motion: reduce) {
  .shell,
  .status-dot::after {
    animation: none;
  }

  .island-btn,
  .island-btn-icon {
    transition: none;
  }
}
</style>
