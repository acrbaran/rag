<template>
  <t-popup :placement="placement" trigger="hover" overlay-class-name="wk-popover permissions-popup-overlay"
    :overlay-inner-style="permissionsPopupInnerStyle">
    <button type="button" class="permissions-trigger-btn" :aria-label="title" :title="hint || title">
      <t-icon name="info-circle" size="16px" />
    </button>
    <template #content>
      <div class="permissions-compact permissions-compact--popover">
        <div class="permissions-compact-header">
          <span class="permissions-compact-title">{{ title }}</span>
          <span v-if="description" class="permissions-compact-desc">{{ description }}</span>
        </div>
        <div class="permissions-compact-grid">
          <div v-for="role in roles" :key="role.key" :class="['perm-role-block', role.key, { 'is-me': highlight === role.key }]">
            <div class="perm-role-tag">
              <t-icon :name="role.icon" size="12px" />
              <span>{{ role.label }}</span>
              <span v-if="highlight === role.key" class="me-badge">{{ highlightLabel || $t('common.me') }}</span>
            </div>
            <div class="perm-items">
              <span v-for="(perm, i) in role.perms" :key="i" :class="['perm-item', perm.has ? 'has' : 'no']">
                <t-icon :name="perm.has ? 'check' : 'close'" size="12px" />
                {{ perm.label }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </template>
  </t-popup>
</template>

<script setup lang="ts">
/**
 * (i) simgesiyle açılan rol → yetki matrisi kartı. İçerik tamamen prop'larla
 * verilir; çalışma alanı rolleri için TenantRolePermissionsPopover, sistem
 * rolleri için SystemRolePermissionsPopover bu bileşeni sarar.
 */
export type RoleMatrixEntry = {
  key: string
  label: string
  icon: string
  perms: { label: string; has: boolean }[]
}

withDefaults(defineProps<{
  title: string
  description?: string
  /** Simgenin üzerine gelindiğinde gösterilen yerel ipucu; verilmezse başlık kullanılır. */
  hint?: string
  roles: RoleMatrixEntry[]
  /** Vurgulanacak rolün key'i; boşsa hiçbir rol vurgulanmaz. */
  highlight?: string
  /** Vurgulanan rolün rozet metni; varsayılan "ben". */
  highlightLabel?: string
  placement?: 'bottom-start' | 'bottom-end' | 'bottom' | 'top-start' | 'top-end' | 'left' | 'right'
}>(), {
  description: '',
  hint: '',
  highlight: '',
  highlightLabel: '',
  placement: 'bottom-start',
})

/** Hover katmanı görünüm alanı içinde sınırlandırılır, içerik dahili kaydırma ile gösterilir */
const permissionsPopupInnerStyle = {
  boxSizing: 'border-box' as const,
  padding: '0',
  width: 'min(520px, calc(100vw - 24px))',
  maxWidth: 'min(520px, calc(100vw - 24px))',
  maxHeight: 'calc(100vh - 24px)',
  overflow: 'hidden',
}
</script>

<style lang="less" scoped>
.permissions-trigger-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 22px;
  height: 22px;
  margin: 0;
  padding: 0;
  border: none;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  line-height: 0;
  transition: background-color var(--app-motion-base) ease, color var(--app-motion-base) ease;

  :deep(.t-icon) {
    display: block;
  }

  &:hover {
    background-color: var(--td-bg-color-secondarycontainer);
    color: var(--td-brand-color);
  }

  &:focus-visible {
    outline: 2px solid var(--td-brand-color-focus);
    outline-offset: 1px;
  }
}
</style>

<style lang="less">
/* Yetki açıklaması açılır katmanı (t-popup body'ye eklenir, genel stil gerekir) */
.permissions-popup-overlay {
  z-index: 3050 !important;

  .permissions-compact.permissions-compact--popover {
    padding: 12px 14px;
    margin: 0;
    max-height: calc(100vh - 32px);
    overflow-x: hidden;
    overflow-y: auto;

    .permissions-compact-header {
      display: flex;
      flex-direction: column;
      gap: 2px;
      margin-bottom: 10px;

      .permissions-compact-title {
        font-size: var(--app-text-md);
        font-weight: 500;
        color: var(--td-text-color-primary);
      }

      .permissions-compact-desc {
        font-size: var(--app-text-sm);
        line-height: 1.45;
        color: var(--td-text-color-secondary);
      }
    }

    .permissions-compact-grid {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 8px;
    }

    .perm-role-block {
      border: 1px solid var(--td-component-stroke);
      border-radius: var(--app-radius-sm);
      padding: 8px 10px;
      background: var(--td-bg-color-container);

      &.is-me {
        border-color: var(--td-brand-color);
        background: var(--td-brand-color-light);
      }

      .perm-role-tag {
        display: flex;
        align-items: center;
        gap: 4px;
        font-size: var(--app-text-sm);
        font-weight: 600;
        color: var(--td-text-color-primary);
        margin-bottom: 6px;

        .me-badge {
          margin-left: auto;
          font-size: var(--app-text-2xs);
          font-weight: 500;
          color: var(--td-brand-color);
          padding: 1px 5px;
          background: var(--td-brand-color-light);
          border-radius: var(--app-radius-xs);
        }
      }

      .perm-items {
        display: flex;
        flex-direction: column;
        gap: 3px;

        .perm-item {
          display: flex;
          align-items: flex-start;
          gap: 4px;
          font-size: var(--app-text-xs);
          line-height: 1.35;
          color: var(--td-text-color-secondary);

          .t-icon {
            margin-top: 1px;
            flex-shrink: 0;
          }

          &.has .t-icon {
            color: var(--td-brand-color);
          }

          &.no {
            color: var(--td-text-color-disabled);

            .t-icon {
              color: var(--td-text-color-disabled);
            }
          }
        }
      }
    }
  }
}

:root[theme-mode='dark'] .permissions-popup-overlay .t-popup__content {
  background: rgba(36, 36, 36, 0.92) !important;
  border-color: rgba(255, 255, 255, 0.08) !important;
  box-shadow:
    0 0 0 0.5px rgba(255, 255, 255, 0.05),
    0 2px 4px rgba(0, 0, 0, 0.12),
    0 8px 32px rgba(0, 0, 0, 0.28) !important;
}

@media (max-width: 480px) {
  .permissions-popup-overlay .permissions-compact.permissions-compact--popover .permissions-compact-grid {
    grid-template-columns: 1fr;
  }
}
</style>
