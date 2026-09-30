<template>
  <t-popup v-model:visible="open" trigger="click" placement="bottom-left" :disabled="disabled"
    overlay-inner-class-name="announcement-date-popup">
    <button :id="id" type="button" class="ann-btn ann-btn--outline ann-btn--block announcement-date-input__trigger"
      :disabled="disabled" :aria-expanded="open" :aria-invalid="invalid || undefined"
      :aria-describedby="invalid ? 'announcement-date-error' : undefined">
      <LucideIcon name="calendar-days" />
      <span class="announcement-date-input__label">{{ selected ? formatted : t('announcements.admin.chooseDate') }}</span>
    </button>
    <template #content>
      <div class="announcement-calendar" data-slot="calendar">
        <div class="announcement-calendar__nav">
          <button type="button" class="ann-btn ann-btn--ghost announcement-calendar__nav-button"
            :aria-label="previousMonthLabel" :disabled="disabled" @click="shiftMonth(-1)">
            <LucideIcon name="arrow-left" />
          </button>
          <button type="button" class="ann-btn ann-btn--ghost announcement-calendar__nav-button"
            :aria-label="nextMonthLabel" :disabled="disabled" @click="shiftMonth(1)">
            <LucideIcon name="arrow-right" />
          </button>
        </div>
        <div class="announcement-calendar__caption" aria-live="polite">{{ monthLabel }}</div>
        <table class="announcement-calendar__grid" role="grid" :aria-label="monthLabel">
          <thead>
            <tr class="announcement-calendar__weekdays">
              <th v-for="day in weekdays" :key="day.key" scope="col" class="announcement-calendar__weekday"
                :aria-label="day.long">
                {{ day.short }}
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(week, index) in weeks" :key="index" class="announcement-calendar__week">
              <td v-for="cell in week" :key="cell.key" class="announcement-calendar__day"
                :class="{ 'is-today': cell.today && !cell.selected, 'is-outside': cell.outside }"
                :data-selected="cell.selected || undefined">
                <button type="button" class="announcement-calendar__day-button" :disabled="disabled"
                  :data-selected-single="cell.selected || undefined" :aria-pressed="cell.selected"
                  :aria-label="cell.label" @click="selectDay(cell.date)">
                  {{ cell.date.getDate() }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="announcement-date-input__time">
        <label :for="`${id}-time`" class="announcement-date-input__time-label">{{ t('announcements.admin.time') }}</label>
        <input :id="`${id}-time`" type="time" class="announcement-date-input__time-input" :value="value.slice(11)"
          :disabled="disabled || !selected" @input="onTime" />
        <button v-if="selected" type="button" class="ann-btn ann-btn--ghost" :disabled="disabled" @click="clear">
          {{ t('announcements.admin.clearDate') }}
        </button>
      </div>
    </template>
  </t-popup>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { datetimeLocalValue } from '@/views/system/adminAnnouncementsUtils'
import LucideIcon from './LucideIcon.vue'

const props = withDefaults(defineProps<{
  id: string
  value: string
  disabled?: boolean
  invalid?: boolean
}>(), { disabled: false, invalid: false })

const emit = defineEmits<{ (e: 'update:value', value: string): void }>()
const { t, locale } = useI18n()

const open = ref(false)
const selected = computed(() => {
  if (!props.value) return null
  const date = new Date(props.value)
  return Number.isNaN(date.getTime()) ? null : date
})
const formatted = computed(() =>
  selected.value
    ? new Intl.DateTimeFormat(locale.value, { dateStyle: 'medium', timeStyle: 'short' }).format(selected.value)
    : '',
)

function monthStart(date: Date) {
  return new Date(date.getFullYear(), date.getMonth(), 1)
}

const month = ref(monthStart(selected.value ?? new Date()))
watch(open, (visible) => {
  if (visible) month.value = monthStart(selected.value ?? new Date())
})

const isTurkish = computed(() => String(locale.value).toLowerCase().startsWith('tr'))
// react-day-picker yerel ayarı gibi: tr haftayı pazartesi, en-US pazar başlatır.
const weekStartsOn = computed(() => (isTurkish.value ? 1 : 0))

const monthLabel = computed(() =>
  new Intl.DateTimeFormat(locale.value, { month: 'long', year: 'numeric' }).format(month.value),
)
const previousMonthLabel = computed(() => (isTurkish.value ? 'Önceki aya git' : 'Go to the Previous Month'))
const nextMonthLabel = computed(() => (isTurkish.value ? 'Sonraki aya git' : 'Go to the Next Month'))

const weekdays = computed(() => {
  const short = new Intl.DateTimeFormat(locale.value, { weekday: 'short' })
  const long = new Intl.DateTimeFormat(locale.value, { weekday: 'long' })
  // 2023-01-01 bir pazar günüdür.
  return Array.from({ length: 7 }, (_, index) => {
    const date = new Date(2023, 0, 1 + ((index + weekStartsOn.value) % 7))
    return { key: index, short: short.format(date), long: long.format(date) }
  })
})

function sameDay(a: Date, b: Date) {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate()
}

const weeks = computed(() => {
  const first = month.value
  const offset = (first.getDay() - weekStartsOn.value + 7) % 7
  const start = new Date(first.getFullYear(), first.getMonth(), 1 - offset)
  const daysInMonth = new Date(first.getFullYear(), first.getMonth() + 1, 0).getDate()
  const rows = Math.ceil((offset + daysInMonth) / 7)
  const today = new Date()
  const label = new Intl.DateTimeFormat(locale.value, { dateStyle: 'full' })
  return Array.from({ length: rows }, (_, row) =>
    Array.from({ length: 7 }, (_, column) => {
      const date = new Date(start.getFullYear(), start.getMonth(), start.getDate() + row * 7 + column)
      return {
        key: `${date.getMonth()}-${date.getDate()}`,
        date,
        label: label.format(date),
        outside: date.getMonth() !== first.getMonth(),
        today: sameDay(date, today),
        selected: selected.value ? sameDay(date, selected.value) : false,
      }
    }),
  )
})

function shiftMonth(delta: number) {
  month.value = new Date(month.value.getFullYear(), month.value.getMonth() + delta, 1)
}

function selectDay(date: Date) {
  if (selected.value && sameDay(date, selected.value)) {
    emit('update:value', '')
    return
  }
  emit('update:value', `${datetimeLocalValue(date.getTime()).slice(0, 10)}T${props.value.slice(11) || '00:00'}`)
  if (date.getMonth() !== month.value.getMonth()) month.value = monthStart(date)
}

function onTime(event: Event) {
  const time = (event.target as HTMLInputElement).value
  emit('update:value', `${props.value.slice(0, 10)}T${time || '00:00'}`)
}

function clear() {
  emit('update:value', '')
  open.value = false
}
</script>

<style lang="less">
.announcement-date-input__trigger {
  justify-content: flex-start;
  gap: 8px;
  font-weight: 400;

  &[aria-invalid='true'] {
    border-color: var(--td-error-color);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--td-error-color) 20%, transparent);
  }
}

.announcement-date-input__label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.t-popup__content.announcement-date-popup {
  width: auto;
  padding: 0;
  overflow: hidden;
  background: var(--td-bg-color-container);
}

.announcement-calendar {
  --cell-size: 32px;
  position: relative;
  width: fit-content;
  padding: 12px;
  color: var(--td-text-color-primary);
  font-size: var(--app-text-base);
}

.announcement-calendar__nav {
  position: absolute;
  top: 12px;
  right: 12px;
  left: 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 4px;
  pointer-events: none;
}

.announcement-calendar__nav-button {
  width: var(--cell-size);
  height: var(--cell-size);
  padding: 0;
  pointer-events: auto;
}

.announcement-calendar__caption {
  display: flex;
  align-items: center;
  justify-content: center;
  height: var(--cell-size);
  padding: 0 var(--cell-size);
  font-weight: 500;
  user-select: none;
}

.announcement-calendar__grid {
  width: 100%;
  margin-top: 16px;
  border-collapse: collapse;
}

.announcement-calendar__weekdays,
.announcement-calendar__week {
  display: flex;
  width: 100%;
}

.announcement-calendar__week {
  margin-top: 8px;
}

.announcement-calendar__weekday {
  flex: 1;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-caption);
  font-weight: 400;
  user-select: none;
}

.announcement-calendar__day {
  position: relative;
  width: 100%;
  height: 100%;
  padding: 0;
  aspect-ratio: 1;
  border-radius: var(--app-radius-token-4xl);
  text-align: center;
  user-select: none;

  &.is-today {
    background: var(--app-surface-muted);
  }

  &.is-outside .announcement-calendar__day-button:not([data-selected-single]) {
    color: var(--td-text-color-secondary);
  }
}

.announcement-calendar__day-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  min-width: var(--cell-size);
  height: auto;
  aspect-ratio: 1;
  padding: 0;
  border: 0;
  border-radius: var(--app-radius-token-4xl);
  background: transparent;
  color: inherit;
  font-family: inherit;
  font-size: var(--app-text-base);
  font-weight: 400;
  line-height: 1;
  cursor: pointer;
  outline: none;
  transition: background-color var(--app-motion-fast) ease, color var(--app-motion-fast) ease;

  &:hover {
    background: var(--app-surface-muted);
  }

  &:focus-visible {
    box-shadow: 0 0 0 1px var(--app-ring-select);
  }

  &[data-selected-single] {
    background: var(--td-brand-color);
    color: var(--td-text-color-anti);
  }

  &:disabled {
    cursor: not-allowed;
    opacity: 0.5;
  }
}

.announcement-date-input__time {
  display: grid;
  gap: 8px;
  padding: 12px;
  border-top: 1px solid var(--td-component-stroke);
}

.announcement-date-input__time-label {
  color: var(--td-text-color-primary);
  font-size: var(--app-text-base);
  font-weight: 500;
}

.announcement-date-input__time-input {
  width: 100%;
  height: 36px;
  padding: 0 14px;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-pill);
  background: var(--td-bg-color-page);
  color: var(--td-text-color-primary);
  font-family: var(--app-font-family);
  font-size: var(--app-text-base);
  outline: none;
  box-sizing: border-box;
  transition: border-color var(--app-motion-fast) ease;

  &:focus-visible {
    border-color: var(--app-ring-select);
  }

  &:disabled {
    cursor: not-allowed;
    opacity: 0.5;
  }
}

:root[theme-mode='dark'] .announcement-date-input__time-input {
  border-color: transparent;
  background: color-mix(in srgb, #fff 6%, transparent);
}
</style>
