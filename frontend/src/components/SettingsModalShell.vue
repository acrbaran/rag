<template>
  <Teleport to="body">
    <Transition name="settings-modal-shell">
      <div v-if="visible" class="settings-modal-shell settings-overlay" :class="overlayClass" :style="{ zIndex }"
        @click.self="emit('close')">
        <div class="settings-modal">
          <div v-if="loading" class="editor-initializing" role="status" :aria-label="$t('common.loading')">
            <t-loading size="medium" :text="$t('common.loading')" />
          </div>

          <button class="close-btn" type="button" @click="emit('close')" :aria-label="$t('common.close')">
            <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
              <path d="M15 5L5 15M5 5L15 15" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
            </svg>
          </button>

          <div class="settings-container">
            <aside class="settings-sidebar">
              <div class="sidebar-header">
                <h2 class="sidebar-title">{{ title }}</h2>
                <slot name="sidebar-header-extra" />
              </div>
              <nav class="settings-nav" :data-guide="navGuide || undefined">
                <slot name="nav">
                  <template v-for="group in navGroups" :key="group.key">
                    <div class="nav-group-title">{{ group.label }}</div>
                    <div v-for="item in group.items" :key="item.key" :class="['nav-item', { active: modelValue === item.key }]"
                      :data-guide="navItemGuidePrefix ? `${navItemGuidePrefix}-${item.key}` : undefined"
                      @click="emit('update:modelValue', item.key)">
                      <slot name="nav-icon" :item="item" :active="modelValue === item.key">
                        <t-icon :name="item.icon" class="nav-icon" />
                      </slot>
                      <span class="nav-label">{{ item.label }}</span>
                      <span v-if="showBadge(item)" :class="['nav-badge', item.badgeClass]">{{ item.badge }}</span>
                    </div>
                  </template>
                </slot>
              </nav>
            </aside>

            <div class="settings-content">
              <div class="settings-body" :class="{ 'is-inner-scroll': innerScroll }">
                <slot />
              </div>
              <div v-if="$slots.footer || $slots['footer-note']" class="settings-footer">
                <slot name="footer-note" />
                <div class="settings-footer-actions">
                  <slot name="footer" />
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
/**
 * Tam ekran "ayar" açılır pencere kabuğu: maske + 1080×780 panel + sol grup gezintisi + sağ içerik + isteğe bağlı alt çubuk.
 * Settings / AgentEditorModal / OrganizationSettingsModal / KnowledgeBaseEditorModal tarafından ortak kullanılır,
 * Kapatma davranışı (Esc / maske / kaydedilmemiş değişiklik koruması) tüketici tarafından useModalShell ile belirlenir; bu bileşen yalnızca `close` yayar.
 *
 * Stiller scoped değildir ve `.settings-modal-shell` kök önekini kullanır; böylece `nav` slotuyla gezintiyi özelleştiren tüketiciler
 * de .nav-item / .nav-icon gibi temel stilleri yeniden kullanabilir.
 */
export interface SettingsModalNavItem {
  key: string
  label: string
  icon?: string
  badge?: number | string | null
  /** Ek rozet class'ı, örneğin 'nav-badge-count'*/
  badgeClass?: string
  /** Rozet 0 olduğunda da gösterilip gösterilmeyeceği*/
  showZeroBadge?: boolean
  [extra: string]: unknown
}

export interface SettingsModalNavGroup {
  key: string
  label: string
  items: SettingsModalNavItem[]
}

withDefaults(
  defineProps<{
    visible: boolean
    title: string
    /** Geçerli bölüm key değeri (v-model)*/
    modelValue?: string
    navGroups?: SettingsModalNavGroup[]
    loading?: boolean
    zIndex?: number
    overlayClass?: string
    /** Kaydırmayı slot içeriği (ör. .content-wrapper) üstlenir; gövde kaymaz ve gutter ayırmaz, çubuk sağ kenara yaslanır*/
    innerScroll?: boolean
    navGuide?: string
    navItemGuidePrefix?: string
  }>(),
  {
    modelValue: '',
    navGroups: () => [],
    loading: false,
    zIndex: 1100,
    overlayClass: '',
    innerScroll: false,
    navGuide: '',
    navItemGuidePrefix: '',
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', key: string): void
  (e: 'close'): void
}>()

function showBadge(item: SettingsModalNavItem): boolean {
  if (item.badge == null || item.badge === '') return false
  if (typeof item.badge === 'number' && item.badge <= 0) return !!item.showZeroBadge
  return true
}
</script>

<style lang="less">
// Rethra settings dialog: 960×820 panel, 248px sidebar, pill nav (reference: rag-platform SettingsDialog).
.settings-modal-shell.settings-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.3);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  backdrop-filter: blur(2px);
  overscroll-behavior: none;
}

.settings-modal-shell {
  .settings-modal {
    position: relative;
    width: min(960px, calc(100vw - 2rem));
    height: min(820px, calc(100dvh - 2rem));
    background: var(--td-bg-color-page);
    border-radius: var(--app-radius-token-xl);
    border: none;
    box-shadow: 0 2px 8px -2px rgba(0, 0, 0, 0.16);
    overflow: hidden;
    display: flex;
    flex-direction: column;
    font-family: var(--app-font-family);
    color: var(--td-text-color-primary);
  }

  .editor-initializing {
    position: absolute;
    inset: 0;
    z-index: 20;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--td-bg-color-page);
  }

  .close-btn {
    position: absolute;
    top: 12px;
    right: 16px;
    width: 28px;
    height: 28px;
    padding: 0;
    border: none;
    background: transparent;
    border-radius: var(--app-radius-pill);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #383835;
    transition: background-color var(--app-motion-fast) ease, color var(--app-motion-fast) ease;
    z-index: 10;

    svg {
      width: 16px;
      height: 16px;
    }

    path {
      stroke-width: 1.75;
    }

    &:hover {
      background: rgb(236, 236, 236);
      color: var(--td-text-color-primary);
    }
  }

  .settings-container {
    display: flex;
    height: 100%;
    width: 100%;
    overflow: hidden;
  }

  .settings-sidebar {
    width: 248px;
    box-sizing: border-box;
    padding: 8px;
    background-color: #fff;
    border-right: 1px solid rgb(242, 242, 242);
    flex-shrink: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    font-family: var(--app-font-heading);
  }

  .sidebar-header {
    padding: 12px 4px 8px;
    flex-shrink: 0;
  }

  // The reference sidebar has no visible title; keep it for screen readers.
  .sidebar-title {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border: 0;
  }

  .sidebar-header:not(:has(> :not(.sidebar-title))) {
    padding: 4px 0 0;
  }

  .settings-nav {
    flex: 1;
    padding: 0 4px 8px;
    overflow-y: auto;
    min-height: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
    scrollbar-width: thin;
    scrollbar-color: transparent transparent;

    &:hover {
      scrollbar-color: rgba(0, 0, 0, 0.18) transparent;
    }
  }

  .nav-group-title {
    padding: 20px 12px 8px;
    color: var(--td-text-color-secondary);
    font-family: var(--app-font-heading);
    font-size: var(--app-text-xs);
    line-height: 15px;
    font-weight: 500;
    letter-spacing: 0;
  }

  .settings-nav > .nav-group-title:first-child {
    padding-top: 12px;
  }

  .nav-item {
    display: flex;
    align-items: center;
    gap: 10px;
    height: 32px;
    min-height: 32px;
    box-sizing: border-box;
    padding: 0 10px 0 12px;
    margin: 0;
    border-radius: var(--app-radius-pill);
    cursor: pointer;
    transition: background-color var(--app-motion-fast) ease, color var(--app-motion-fast) ease;
    font-family: var(--app-font-heading);
    font-size: var(--app-text-settings-nav);
    line-height: 17.8125px;
    font-weight: 500;
    color: #383835;
    user-select: none;
    flex-shrink: 0;

    &:hover {
      background-color: rgb(236, 236, 236);
      color: rgb(38, 38, 38);
    }

    &.active {
      background-color: rgb(236, 236, 236);
      color: rgb(38, 38, 38);
      font-weight: 500;
    }
  }

  .nav-icon {
    margin: 0;
    width: 15px;
    height: 15px;
    font-size: var(--app-text-lg);
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    color: inherit;

    &.nav-icon-img {
      width: 15px;
      height: 15px;
    }

  }

  .nav-label {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .nav-badge {
    flex-shrink: 0;
    margin-left: 2px;
    padding: 0 6px;
    border-radius: var(--app-radius-pill);
    background: rgb(242, 242, 242);
    color: var(--td-text-color-secondary);
    font-family: var(--app-font-family);
    font-size: var(--app-text-xs);
    line-height: 16px;
    font-weight: 500;
    text-align: center;

    &.nav-badge-count {
      min-width: 20px;
    }
  }

  .settings-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
    min-height: 0;
    overflow: hidden;
    background-color: var(--td-bg-color-page);
  }

  // Scroll container and flex column: consumers may scroll their own .content-wrapper (flex:1 + overflow:auto)
  // or keep natural height and let this body scroll.
  .settings-body {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    overflow-y: auto;
    scrollbar-gutter: stable;
    scrollbar-width: thin;
    scrollbar-color: rgba(0, 0, 0, 0.18) transparent;

    &.is-inner-scroll {
      overflow: hidden;
      scrollbar-gutter: auto;

      > * {
        scrollbar-gutter: stable;
        scrollbar-width: thin;
        scrollbar-color: rgba(0, 0, 0, 0.18) transparent;
      }
    }
  }

  .settings-footer {
    padding: 12px 24px;
    border-top: 1px solid rgb(242, 242, 242);
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 12px;
    flex-shrink: 0;
    background-color: var(--td-bg-color-page);
  }

  .settings-footer-note {
    margin: 0;
    margin-right: auto;
    flex: 1;
    min-width: 0;
    display: flex;
    align-items: flex-start;
    gap: 6px;
    font-size: var(--app-text-xs);
    line-height: 1.625;
    color: var(--td-text-color-secondary);

    strong {
      margin-right: 4px;
      color: var(--td-text-color-primary);
      font-weight: 500;
    }

    &__icon {
      flex-shrink: 0;
      margin-top: 2px;
      font-size: var(--app-text-base);
      color: var(--td-success-color);
    }
  }

  .settings-footer-actions {
    display: flex;
    gap: 8px;
    flex-shrink: 0;
  }
}

html[theme-mode="dark"] .settings-modal-shell {
  .settings-modal,
  .settings-content,
  .settings-footer,
  .editor-initializing {
    background: var(--td-bg-color-container);
    color: var(--td-text-color-primary);
  }

  .settings-sidebar {
    background: var(--td-bg-color-settings-modal);
    border-right-color: var(--td-border-level-1-color);
  }

  .nav-item,
  .close-btn {
    color: var(--app-nav-fg);

    &:hover,
    &.active {
      background: var(--app-nav-active);
      color: var(--td-text-color-primary);
    }
  }

  .nav-group-title {
    color: var(--app-nav-muted);
  }
}
</style>
