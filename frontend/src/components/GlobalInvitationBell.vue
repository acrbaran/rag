<template>
  <div v-if="!authStore.isLiteMode" class="global-invitation-bell">
    <t-popup :visible="popupVisible" trigger="click" placement="bottom-left" :show-arrow="false"
      overlay-inner-class-name="announcement-bell-popup" :popper-options="{ modifiers: [{ name: 'offset', options: { offset: [0, 8] } }] }"
      @visible-change="onVisibleChange">
      <button type="button" class="global-invitation-bell__btn" :title="$t('announcements.notificationsTitle')"
        :aria-label="$t('announcements.notificationsTitle')" :aria-expanded="popupVisible">
        <LucideIcon name="bell" :stroke-width="1.75"
          :class="{ 'announcement-bell-attention': unreadCount > 0 && !popupVisible }" />
        <span v-if="pendingInvitationCount > 0 || unreadCount > 0" class="global-invitation-bell__dot">
          <span class="ann-sr-only">
            {{ pendingInvitationCount + unreadCount }} {{ $t('announcements.unread') }}
          </span>
        </span>
      </button>
      <template #content>
        <div class="announcement-bell" :aria-label="$t('announcements.notificationsTitle')">
          <div class="announcement-bell__tabs">
            <div class="ann-tabs" role="tablist">
              <button v-for="tab in tabs" :id="`announcement-bell-tab-${tab.key}`" :key="tab.key" type="button"
                role="tab" class="ann-tabs__trigger" :aria-selected="activeTab === tab.key"
                :aria-controls="`announcement-bell-panel-${tab.key}`" :tabindex="activeTab === tab.key ? 0 : -1"
                @click="activeTab = tab.key" @keydown.left.prevent="moveTab(-1)" @keydown.right.prevent="moveTab(1)">
                {{ tab.label }}
                <span v-if="tab.count > 0" class="ann-tabs__count">{{ tab.count > 99 ? '99+' : tab.count }}</span>
              </button>
            </div>
          </div>
          <div v-if="activeTab === 'invitations'" id="announcement-bell-panel-invitations" role="tabpanel"
            aria-labelledby="announcement-bell-tab-invitations">
            <MyInvitationsList :active="popupVisible && activeTab === 'invitations'" compact @close="closePopup" />
          </div>
          <div v-else id="announcement-bell-panel-announcements" role="tabpanel"
            aria-labelledby="announcement-bell-tab-announcements">
            <AnnouncementNotifications @close="closePopup" @open-all="openAll" @details="openDetails" />
          </div>
        </div>
      </template>
    </t-popup>
    <AnnouncementDetailsDialog :visible="details !== null" :title="details?.title" :body="details?.body ?? ''"
      @update:visible="(value) => { if (!value) details = null }" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useUIStore } from '@/stores/ui'
import { useAnnouncements } from '@/composables/useAnnouncements'
import MyInvitationsList from '@/components/MyInvitationsList.vue'
import AnnouncementNotifications from '@/components/announcements/AnnouncementNotifications.vue'
import AnnouncementDetailsDialog from '@/components/announcements/AnnouncementDetailsDialog.vue'
import LucideIcon from '@/components/announcements/LucideIcon.vue'

type BellTab = 'invitations' | 'announcements'

const { t } = useI18n()
const authStore = useAuthStore()
const uiStore = useUIStore()
const { state, clearPreview } = useAnnouncements()

const pendingInvitationCount = computed(() => authStore.pendingInvitationCount)
const unreadCount = computed(() =>
  (state.data?.items ?? []).filter((item) => {
    const itemState = item.notificationState?.published
    return !itemState?.deleted && !itemState?.read
  }).length,
)
// Admin ekranındaki "Test" yalnızca liste önizlemesi üretir; referanstaki gibi açılır pencereyi açar.
const localPreview = computed(() => (state.preview && !state.preview.showBanner ? state.preview : null))

const open = ref(false)
const popupVisible = computed(() => open.value || localPreview.value !== null)
const activeTab = ref<BellTab>('invitations')
const details = ref<{ title: string; body: string } | null>(null)

const tabs = computed(() => [
  { key: 'invitations' as const, label: t('announcements.bellTabs.invitations'), count: pendingInvitationCount.value },
  { key: 'announcements' as const, label: t('announcements.bellTabs.announcements'), count: unreadCount.value },
])

function moveTab(step: number) {
  const index = tabs.value.findIndex((tab) => tab.key === activeTab.value)
  const next = tabs.value[(index + step + tabs.value.length) % tabs.value.length]
  activeTab.value = next.key
  document.getElementById(`announcement-bell-tab-${next.key}`)?.focus()
}

function onVisibleChange(visible: boolean) {
  if (visible) {
    // Bekleyen davetiye yoksa ve okunmamış duyuru varsa doğrudan duyurular sekmesiyle aç.
    activeTab.value = pendingInvitationCount.value === 0 && unreadCount.value > 0 ? 'announcements' : activeTab.value
    open.value = true
    return
  }
  closePopup()
}

function closePopup() {
  open.value = false
  if (localPreview.value) clearPreview()
}

function openAll() {
  closePopup()
  uiStore.openSettings('announcements')
}

function openDetails(value: { title: string; body: string }) {
  closePopup()
  details.value = value
}

watch(localPreview, (preview) => {
  if (preview) activeTab.value = 'announcements'
})
</script>

<style lang="less" scoped>
.global-invitation-bell {
  display: flex;
  flex: none;
}

.global-invitation-bell__btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  position: relative;
  width: 30px;
  height: 30px;
  padding: 0;
  border: 0;
  border-radius: var(--app-radius-control);
  background: transparent;
  color: #8f8f8f;
  cursor: pointer;
  transition: background-color var(--app-motion-fast) ease, color var(--app-motion-fast) ease;

  &:hover,
  &[aria-expanded='true'] {
    background-color: var(--app-nav-hover);
    color: var(--td-text-color-primary);
  }

  &:focus-visible {
    outline: 2px solid var(--td-brand-color-focus);
    outline-offset: 1px;
  }
}

.global-invitation-bell__dot {
  position: absolute;
  top: 4px;
  right: 4px;
  width: 8px;
  height: 8px;
  border: 2px solid var(--td-bg-color-sidebar);
  border-radius: 50%;
  background: var(--td-brand-color);
  box-sizing: content-box;
}
</style>

<style lang="less">
// Referans PopoverContent: w-[min(24rem,calc(100vw-2rem))] rounded-2xl border p-0 shadow-lg
.t-popup__content.announcement-bell-popup {
  width: min(24rem, calc(100vw - 2rem));
  padding: 0;
  overflow: hidden;
  border-radius: var(--app-radius-token-2xl);
  background: var(--td-bg-color-container);
}

.announcement-bell__tabs {
  padding: 8px;
  border-bottom: 1px solid var(--td-component-stroke);
}
</style>
