<template>
  <div class="announcement-targeting">
    <div class="announcement-targeting__field">
      <label for="announcement-type" class="announcement-targeting__label">{{ t('announcements.admin.typeLabel') }}</label>
      <div class="announcement-targeting__type-row">
        <t-select id="announcement-type" class="announcement-targeting__type-select" :value="typeId" :disabled="busy"
          :popup-props="{ overlayClassName: 'announcement-select-popup' }" @change="emit('type-change', String($event))">
          <t-option v-for="type in types" :key="type.id" :value="type.id" :label="announcementTypeLabel(type, t)" />
        </t-select>
        <button type="button" class="ann-btn ann-btn--outline" :disabled="busy" :aria-expanded="addingType"
          @click="addingType = !addingType">
          {{ t('announcements.admin.addType') }}
        </button>
      </div>
      <div v-if="addingType" class="announcement-targeting__box">
        <label for="announcement-type-name" class="announcement-targeting__box-label">
          {{ t('announcements.admin.typeName') }}
        </label>
        <div class="announcement-targeting__type-row">
          <t-input id="announcement-type-name" v-model="name" :maxlength="60" :disabled="busy"
            @enter="(_: unknown, ctx: { e: KeyboardEvent }) => { ctx.e.preventDefault(); void createType() }" />
          <button type="button" class="ann-btn ann-btn--default" :disabled="busy || !name.trim()"
            @click="createType">
            {{ t(creating ? 'announcements.admin.saving' : 'announcements.admin.createType') }}
          </button>
        </div>
        <p v-if="typeError" role="alert" class="announcement-targeting__error">{{ typeError }}</p>
      </div>
    </div>
    <div class="announcement-targeting__field">
      <label for="announcement-audience" class="announcement-targeting__label">
        {{ t('announcements.admin.audienceTitle') }}
      </label>
      <t-select id="announcement-audience" :value="audience.mode" :disabled="busy"
        :popup-props="{ overlayClassName: 'announcement-select-popup' }" @change="changeMode">
        <t-option v-for="mode in AUDIENCE_MODES" :key="mode" :value="mode" :label="t(AUDIENCE_LABELS[mode])" />
      </t-select>
      <div v-if="audience.mode !== 'all'" class="announcement-targeting__box">
        <t-input v-if="audience.mode !== 'roles'" v-model="query" :aria-label="t('announcements.admin.searchTargets')"
          :placeholder="t('announcements.admin.searchTargets')" :maxlength="100" :disabled="busy"
          @enter="(_: unknown, ctx: { e: KeyboardEvent }) => ctx.e.preventDefault()" />
        <div class="announcement-targeting__options" role="group" :aria-label="t('announcements.admin.audienceTitle')">
          <label v-for="item in items ?? []" :key="item.id" class="announcement-targeting__option">
            <t-checkbox :checked="audience.ids.includes(item.id)"
              :disabled="busy || (!audience.ids.includes(item.id) && audience.ids.length >= 200)"
              @change="(checked: boolean) => toggle(item.id, checked)" />
            <span class="announcement-targeting__option-text">
              {{ audience.mode === 'roles' ? roleLabel(item.id, item.label) : item.label }}
              <span v-if="item.detail" class="announcement-targeting__option-detail">{{ item.detail }}</span>
            </span>
          </label>
          <p v-if="!items && !searchError" class="announcement-targeting__option-status">
            {{ t('announcements.admin.loading') }}
          </p>
          <p v-if="items && items.length === 0" class="announcement-targeting__option-status">
            {{ t('announcements.admin.noTargets') }}
          </p>
        </div>
        <p class="announcement-targeting__hint">
          {{ t('announcements.admin.selectedTargets', { count: audience.ids.length }) }}
        </p>
        <p v-if="audience.mode !== 'roles'" class="announcement-targeting__hint">
          {{ t('announcements.admin.searchTargetsHint') }}
        </p>
        <p v-if="audience.ids.length === 0" class="announcement-targeting__hint">
          {{ t('announcements.admin.selectTarget') }}
        </p>
        <p v-if="searchError" role="alert" class="announcement-targeting__error">{{ searchError }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  announcementsApi,
  announcementTypeLabel,
  AUDIENCE_LABELS,
  type AnnouncementAudience,
  type AnnouncementType,
  type AudienceMode,
  type AudienceOption,
} from '@/api/announcements'

const props = defineProps<{
  types: AnnouncementType[]
  typeId: string
  audience: AnnouncementAudience
  disabled: boolean
}>()

const emit = defineEmits<{
  (e: 'type-change', id: string): void
  (e: 'type-created', type: AnnouncementType): void
  (e: 'audience-change', audience: AnnouncementAudience): void
}>()

const AUDIENCE_MODES: AudienceMode[] = ['all', 'workspaces', 'roles', 'users']

const { t, te } = useI18n()
const addingType = ref(false)
const name = ref('')
const creating = ref(false)
const typeError = ref<string | null>(null)
const query = ref('')
const options = ref<{ key: string; items: AudienceOption[] } | null>(null)
const searchError = ref<string | null>(null)

const busy = computed(() => props.disabled || creating.value)
const selectionKey = computed(() => JSON.stringify(props.audience))
const searchKey = computed(() => JSON.stringify([props.audience.mode, query.value]))
const items = computed(() => (options.value?.key === searchKey.value ? options.value.items : null))

function roleLabel(id: string, fallback: string) {
  const key = `announcements.admin.roles.${id}`
  return te(key) ? t(key) : fallback
}

let timer: ReturnType<typeof setTimeout> | null = null
let controller: AbortController | null = null

function cancelSearch() {
  if (timer) clearTimeout(timer)
  timer = null
  controller?.abort()
  controller = null
}

watch([selectionKey, searchKey], () => {
  cancelSearch()
  const selection = JSON.parse(selectionKey.value) as AnnouncementAudience
  if (selection.mode === 'all') return
  const key = searchKey.value
  const search = query.value
  const current = new AbortController()
  controller = current
  timer = setTimeout(() => {
    searchError.value = null
    void Promise.all([
      announcementsApi.options(selection.mode, search, undefined, current.signal),
      selection.ids.length
        ? announcementsApi.options(selection.mode, '', selection.ids, current.signal)
        : Promise.resolve([] as AudienceOption[]),
    ]).then(([found, selected]) => {
      if (current.signal.aborted) return
      const merged = new Map([...found, ...selected].map((item) => [item.id, item]))
      for (const id of selection.ids) {
        if (!merged.has(id)) merged.set(id, { id, label: t('announcements.admin.unavailableTarget') })
      }
      options.value = { key, items: [...merged.values()] }
    }).catch(() => {
      if (!current.signal.aborted) searchError.value = t('announcements.admin.targetsFailed')
    })
  }, 200)
}, { immediate: true })

onBeforeUnmount(cancelSearch)

function changeMode(value: unknown) {
  emit('audience-change', { mode: value as AudienceMode, ids: [] })
  query.value = ''
}

function toggle(id: string, checked: boolean) {
  const ids = props.audience.ids
  emit('audience-change', {
    mode: props.audience.mode,
    ids: checked ? [...ids, id].sort() : ids.filter((value) => value !== id),
  })
}

async function createType() {
  if (creating.value || !name.value.trim()) return
  creating.value = true
  typeError.value = null
  try {
    const result = await announcementsApi.createType(name.value.trim())
    emit('type-created', result)
    emit('type-change', result.id)
    addingType.value = false
    name.value = ''
  } catch (error) {
    typeError.value = error instanceof Error ? error.message : t('announcements.admin.typeFailed')
  } finally {
    creating.value = false
  }
}
</script>

<style lang="less" scoped>
.announcement-targeting {
  display: flex;
  flex-direction: column;
  gap: 16px;

  p {
    margin: 0;
  }
}

.announcement-targeting__field {
  display: grid;
  gap: 6px;
}

.announcement-targeting__label {
  font-size: var(--app-text-base);
  font-weight: 500;
}

.announcement-targeting__type-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;

  > .t-input__wrap,
  > .t-select__wrap {
    flex: 1;
    min-width: 160px;
  }
}

.announcement-targeting__type-select {
  flex: 1;
  min-width: 160px;
}

.announcement-targeting__box {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 8px;
  padding: 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-token-xl);
}

.announcement-targeting__box-label {
  font-size: var(--app-text-base);
}

.announcement-targeting__options {
  display: flex;
  max-height: 176px;
  flex-direction: column;
  gap: 4px;
  overflow-y: auto;
}

.announcement-targeting__option {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px;
  border-radius: var(--app-radius-token);
  cursor: pointer;
  transition: background-color var(--app-motion-fast) ease;

  &:hover {
    background: color-mix(in srgb, var(--app-surface-muted) 50%, transparent);
  }

  :deep(.t-checkbox) {
    margin: 0;
  }

  :deep(.t-checkbox__label) {
    display: none;
  }
}

.announcement-targeting__option-text {
  min-width: 0;
  font-size: var(--app-text-base);
  word-break: break-word;
}

.announcement-targeting__option-detail {
  display: block;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);
}

.announcement-targeting__option-status {
  padding: 8px;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-base);
}

.announcement-targeting__hint {
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);
}

.announcement-targeting__error {
  color: var(--td-error-color);
  font-size: var(--app-text-base);
}
</style>
