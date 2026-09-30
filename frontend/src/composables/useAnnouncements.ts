import { computed, reactive } from 'vue'
import i18n from '@/i18n'
import { useAuthStore } from '@/stores/auth'
import { isEmbedPage } from '@/utils/authRefresh'
import {
  ANNOUNCEMENT_PREVIEW,
  announcementsApi,
  type AnnouncementBannerColor,
  type AnnouncementType,
  type CurrentAnnouncement,
  type NotificationAction,
  type NotificationRef,
} from '@/api/announcements'

export interface AnnouncementPreview {
  title: string
  body: string
  type?: AnnouncementType
  bannerColor?: AnnouncementBannerColor
  showBanner: boolean
}

const POLL_INTERVAL_MS = 15_000

// Tek bir paylaşılan durum: zil, ayarlar sayfası ve banner aynı veriyi okur.
const state = reactive({
  running: false,
  data: null as CurrentAnnouncement | null,
  unavailable: false,
  preview: null as AnnouncementPreview | null,
  notificationsBusy: false,
  notificationsError: null as string | null,
})

// Oturum değiştiğinde (çıkış / başka kullanıcı) eski isteklerin sonucu yok sayılır.
let epoch = 0
let revision = 0
let notificationPending: number | null = null
let refreshPending = false
let controller: AbortController | null = null
let pollTimer: ReturnType<typeof setInterval> | null = null
let stopViews: (() => void) | null = null

async function refresh() {
  if (!state.running || refreshPending || notificationPending === epoch) return
  refreshPending = true
  const started = epoch
  const startedRevision = revision
  const signal = controller?.signal
  try {
    const data = await announcementsApi.current(signal)
    if (signal?.aborted || started !== epoch || startedRevision !== revision) return
    state.data = data
    state.unavailable = false
  } catch {
    if (!signal?.aborted && started === epoch && startedRevision === revision) state.unavailable = true
  } finally {
    refreshPending = false
  }
}

const onRefreshEvent = () => void refresh()

const onPreview = (event: Event) => {
  const { title, body, type, bannerColor, showBanner } = (event as CustomEvent<AnnouncementPreview>).detail
  state.preview = { title, body, type, bannerColor, showBanner }
}

// Başlık görünür sekmede en az bir saniye kaldıysa duyuru "görüldü" sayılır.
function observeAnnouncementViews(send: (items: NotificationRef[]) => Promise<unknown>) {
  const selector = '[data-announcement-id][data-announcement-phase]'
  const visible = new Map<Element, number>()
  const observed = new Set<Element>()
  const sent = new Set<string>()
  let pending = false
  let stopped = false
  const observer = new IntersectionObserver((entries) => {
    for (const entry of entries) {
      if (entry.isIntersecting && entry.intersectionRatio >= 0.5) visible.set(entry.target, Date.now())
      else visible.delete(entry.target)
    }
  }, { threshold: 0.5 })
  const scan = () => {
    for (const node of observed) {
      if (!node.isConnected || !node.matches(selector)) {
        observer.unobserve(node)
        observed.delete(node)
        visible.delete(node)
      }
    }
    document.querySelectorAll(selector).forEach((node) => {
      if (!observed.has(node)) {
        observed.add(node)
        observer.observe(node)
      }
    })
  }
  const resetVisibility = () => {
    for (const node of visible.keys()) visible.set(node, Date.now())
  }
  const mutations = new MutationObserver((records) => {
    for (const record of records) {
      if (record.type === 'attributes' && visible.has(record.target as Element)) {
        visible.set(record.target as Element, Date.now())
      }
    }
    scan()
  })
  mutations.observe(document.body, {
    childList: true,
    subtree: true,
    attributes: true,
    attributeFilter: ['data-announcement-id', 'data-announcement-phase'],
  })
  document.addEventListener('visibilitychange', resetVisibility)
  scan()
  const timer = setInterval(async () => {
    if (stopped || pending || document.visibilityState !== 'visible') return
    const batch = new Map<string, NotificationRef>()
    for (const [node, since] of visible) {
      const announcementId = node.getAttribute('data-announcement-id')
      const phase = node.getAttribute('data-announcement-phase')
      const key = `${announcementId}:${phase}`
      if (node.isConnected && announcementId && phase === 'published' && !sent.has(key) && Date.now() - since >= 1000) {
        batch.set(key, { announcementId, phase })
        if (batch.size === 100) break
      }
    }
    if (!batch.size) return
    pending = true
    try {
      await send([...batch.values()])
      for (const key of batch.keys()) sent.add(key)
    } catch {
      // Bağlantı hatasında görünür duyurular bir sonraki turda yeniden denenir.
    } finally {
      pending = false
    }
  }, 1000)
  return () => {
    stopped = true
    clearInterval(timer)
    observer.disconnect()
    mutations.disconnect()
    document.removeEventListener('visibilitychange', resetVisibility)
  }
}

export function startAnnouncements() {
  if (state.running || isEmbedPage()) return
  epoch++
  state.running = true
  controller = new AbortController()
  const started = epoch
  void refresh()
  pollTimer = setInterval(() => void refresh(), POLL_INTERVAL_MS)
  window.addEventListener('focus', onRefreshEvent)
  window.addEventListener('online', onRefreshEvent)
  window.addEventListener(ANNOUNCEMENT_PREVIEW, onPreview)
  stopViews = observeAnnouncementViews((items) => {
    if (started !== epoch) return Promise.reject(new Error('Session changed'))
    return announcementsApi.updateNotifications('seen', items)
  })
}

export function stopAnnouncements() {
  if (!state.running) return
  epoch++
  state.running = false
  controller?.abort()
  controller = null
  if (pollTimer) clearInterval(pollTimer)
  pollTimer = null
  stopViews?.()
  stopViews = null
  window.removeEventListener('focus', onRefreshEvent)
  window.removeEventListener('online', onRefreshEvent)
  window.removeEventListener(ANNOUNCEMENT_PREVIEW, onPreview)
  notificationPending = null
  state.data = null
  state.unavailable = false
  state.preview = null
  state.notificationsBusy = false
  state.notificationsError = null
}

async function updateNotifications(action: NotificationAction, items: NotificationRef[]) {
  if (!state.running || notificationPending === epoch || !items.length) return
  const started = epoch
  notificationPending = started
  revision++
  state.notificationsBusy = true
  state.notificationsError = null
  try {
    const result = await announcementsApi.updateNotifications(action, items)
    if (started !== epoch || !state.data) return
    const previous = state.data
    const current = previous.announcement
    const updatedCurrent = current ? result.items.find((item) => item.id === current.id) : undefined
    state.data = {
      ...previous,
      announcement: current && updatedCurrent
        ? { ...current, notificationState: updatedCurrent.notificationState }
        : current,
      announcements: previous.announcements?.map((announcement) => {
        const updated = result.items.find((item) => item.id === announcement.id)
        return updated ? { ...announcement, notificationState: updated.notificationState } : announcement
      }),
      items: result.items,
    }
  } catch {
    if (started === epoch) state.notificationsError = i18n.global.t('announcements.notificationActionFailed')
  } finally {
    if (started === epoch) state.notificationsBusy = false
    if (notificationPending === started) notificationPending = null
  }
}

export function publishAnnouncementPreview(preview: AnnouncementPreview) {
  window.dispatchEvent(new CustomEvent<AnnouncementPreview>(ANNOUNCEMENT_PREVIEW, { detail: preview }))
}

export function useAnnouncements() {
  const authStore = useAuthStore()
  const authenticated = computed(() => state.running && authStore.isLoggedIn && !authStore.isLiteMode)
  return {
    state,
    authenticated,
    clearPreview: () => { state.preview = null },
    updateNotifications,
    refresh,
  }
}
