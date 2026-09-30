<template>
  <div
    class="option-cards"
    :class="{ 'is-disabled': disabled }"
    :style="{ gridTemplateColumns: `repeat(${options.length}, minmax(0, 1fr))` }"
    role="radiogroup"
    :aria-label="ariaLabel"
    :aria-disabled="disabled || undefined"
  >
    <button
      v-for="opt in options"
      :key="opt.value"
      type="button"
      role="radio"
      :aria-checked="modelValue === opt.value"
      :disabled="disabled || opt.disabled"
      :class="['option-card', { 'is-active': modelValue === opt.value }]"
      @click="select(opt.value)"
    >
      <t-icon v-if="opt.icon" :name="opt.icon" size="16px" class="option-card__icon" />
      <span class="option-card__label">{{ opt.label }}</span>
    </button>
  </div>
</template>

<script lang="ts">
export interface OptionCardItem<T extends string = string> {
  value: T;
  label: string;
  icon?: string;
  disabled?: boolean;
}
</script>

<script setup lang="ts" generic="T extends string">
const props = defineProps<{
  modelValue: T;
  options: OptionCardItem<T>[];
  disabled?: boolean;
  ariaLabel?: string;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: T): void;
}>();

const select = (value: T) => {
  if (props.disabled || value === props.modelValue) return;
  emit('update:modelValue', value);
};
</script>

<style scoped lang="less">
.option-cards {
  display: grid;
  gap: 8px;
  width: 100%;
}

.option-card {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-width: 0;
  min-height: 38px;
  padding: 8px 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-md);
  background: var(--td-bg-color-container);
  color: var(--td-text-color-secondary);
  font: inherit;
  font-size: var(--app-text-xs);
  font-weight: 500;
  line-height: 1.3;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s ease, background-color 0.15s ease, color 0.15s ease;

  &:hover:not(:disabled) {
    border-color: var(--td-brand-color);
    color: var(--td-text-color-primary);
  }

  &:focus-visible {
    outline: 2px solid var(--td-brand-color);
    outline-offset: 2px;
  }

  &.is-active {
    border-color: var(--td-brand-color);
    background: var(--td-brand-color-light);
    color: var(--td-brand-color);

    .option-card__icon {
      color: var(--td-brand-color);
    }
  }

  &:disabled {
    cursor: not-allowed;
    opacity: 0.6;
  }

  &__icon {
    flex-shrink: 0;
    color: var(--td-text-color-placeholder);
    transition: color 0.15s ease;
  }

  &__label {
    min-width: 0;
    overflow-wrap: anywhere;
  }
}
</style>
