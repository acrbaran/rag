import { del, get, patch, post } from '@/utils/request'

export const ANNOUNCEMENT_PREVIEW = 'announcement-preview'

export interface AnnouncementType {
  id: string
  name: string
  builtin: boolean
}

export type AudienceMode = 'all' | 'workspaces' | 'roles' | 'users'

export interface AnnouncementAudience {
  mode: AudienceMode
  ids: string[]
}

export interface AudienceCounts {
  users: number
  subscribedUsers: number
  devices: number
}

export interface AudienceOption {
  id: string
  label: string
  detail?: string
}

export const ALL_AUDIENCE: AnnouncementAudience = { mode: 'all', ids: [] }
export const ANNOUNCEMENT_BANNER_COLORS = ['neutral', 'blue', 'green', 'amber', 'red', 'violet'] as const
export type AnnouncementBannerColor = (typeof ANNOUNCEMENT_BANNER_COLORS)[number]

export interface NotificationRef {
  announcementId: string
  phase: 'published' | 'resolved'
}

export type NotificationAction = 'read' | 'unread' | 'delete' | 'seen' | 'dismiss'

export interface NotificationState {
  read: boolean
  deleted: boolean
  dismissedVersion?: number | null
}

export interface AnnouncementDelivery {
  pending: number
  accepted: number
  failed: number
  cancelled: number
}

export type AnnouncementStatus = 'draft' | 'maintenance' | 'withdrawn' | 'resolved'

export interface Announcement {
  id: string
  notificationState?: Partial<Record<NotificationRef['phase'], NotificationState>>
  showBanner?: boolean
  bannerColor?: AnnouncementBannerColor
  typeId?: string
  type?: AnnouncementType
  audience?: AnnouncementAudience
  audienceCounts?: AudienceCounts
  status: AnnouncementStatus
  version: number
  title: string
  body: string
  startsAt: number | null
  endsAt: number | null
  publishedAt: number | null
  resolvedAt: number | null
  expiresAt: number | null
  createdAt: number
  createdBy: string
  createdByName?: string | null
  createdByEmail?: string | null
  publishedBy?: string | null
  publishedByName?: string | null
  publishedByEmail?: string | null
  resolvedBy?: string | null
  resolutionTitle: string | null
  resolutionBody: string | null
  delivery?: AnnouncementDelivery
  viewerCount?: number
}

export interface DraftInput {
  showBanner?: boolean
  bannerColor?: AnnouncementBannerColor
  typeId?: string
  audience?: AnnouncementAudience
  title: string
  body: string
  startsAt?: number | null
  endsAt?: number | null
}

export interface AnnouncementList {
  items: Announcement[]
  audience: AudienceCounts
  types?: AnnouncementType[]
  pushConfigured: boolean
}

export interface CurrentAnnouncement {
  announcement: Announcement | null
  announcements?: Announcement[]
  items: Announcement[]
  pushConfigured: boolean
  publicKey: string | null
}

export interface AnnouncementViewers {
  total: number
  items: { userId: string; name: string; email: string; seenAt: number }[]
}

export class AnnouncementRequestError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.status = status
  }
}

type Envelope<T> = { success?: boolean; data?: T; error?: unknown; message?: string }

async function unwrap<T>(request: Promise<Envelope<T>>): Promise<T> {
  let payload: Envelope<T>
  try {
    payload = await request
  } catch (err: any) {
    throw new AnnouncementRequestError(
      err?.message || `Request failed (${err?.status ?? 0})`,
      Number(err?.status) || 0,
    )
  }
  if (!payload || !('data' in payload)) {
    throw new AnnouncementRequestError(payload?.message || 'Request failed', 0)
  }
  return payload.data as T
}

const ADMIN = '/api/v1/system/admin/announcements'
const USER = '/api/v1/announcements'
const withSignal = (signal?: AbortSignal) => (signal ? { signal } : undefined)
const idPath = (id: string) => `${ADMIN}/${encodeURIComponent(id)}`

export const announcementsApi = {
  list: () => unwrap<AnnouncementList>(get(ADMIN)),
  viewers: (id: string, phase: NotificationRef['phase'], page: number, signal?: AbortSignal) =>
    unwrap<AnnouncementViewers>(get(`${idPath(id)}/viewers?phase=${phase}&page=${page}`, withSignal(signal))),
  createType: (name: string) => unwrap<AnnouncementType>(post(`${ADMIN}/types`, { name })),
  audience: (selection: AnnouncementAudience, signal?: AbortSignal) =>
    unwrap<AudienceCounts>(post(`${ADMIN}/audience`, selection, withSignal(signal))),
  options: (mode: string, query: string, ids?: string[], signal?: AbortSignal) => {
    const params = new URLSearchParams({ mode, q: query })
    ids?.forEach((id) => params.append('id', id))
    return unwrap<AudienceOption[] | null>(get(`${ADMIN}/options?${params}`, withSignal(signal)))
      .then((items) => items ?? [])
  },
  current: (signal?: AbortSignal) =>
    unwrap<CurrentAnnouncement>(get(`${USER}/current`, withSignal(signal))),
  updateNotifications: (action: NotificationAction, items: NotificationRef[]) =>
    unwrap<{ items: Announcement[] }>(patch(`${USER}/notifications`, { action, items })),
  create: (input: DraftInput & { idempotencyKey: string }) => unwrap<Announcement>(post(ADMIN, input)),
  update: (id: string, input: DraftInput & { version: number }) =>
    unwrap<Announcement>(patch(idPath(id), input)),
  delete: (id: string, input: { version: number }) =>
    unwrap<{ deleted: boolean }>(del(idPath(id), input)),
  publish: (id: string, input: { version: number; idempotencyKey: string; keepDraft: boolean }) =>
    unwrap<Announcement>(post(`${idPath(id)}/publish`, input)),
  unpublish: (id: string, input: { version: number }) =>
    unwrap<Announcement>(post(`${idPath(id)}/unpublish`, input)),
}

export type AnnouncementTone = 'maintenance' | 'info' | 'update' | 'warning' | 'custom'

export function announcementTone(typeId?: string): AnnouncementTone {
  return typeId === 'maintenance' || typeId === 'info' || typeId === 'update' || typeId === 'warning'
    ? typeId
    : 'custom'
}

export const AUDIENCE_LABELS: Record<AudienceMode, string> = {
  all: 'announcements.admin.audienceAll',
  workspaces: 'announcements.admin.audienceWorkspaces',
  roles: 'announcements.admin.audienceRoles',
  users: 'announcements.admin.audienceUsers',
}

export function announcementTypeLabel(type: AnnouncementType | undefined, t: (key: string) => string): string {
  const id = type?.id ?? 'maintenance'
  if (type?.builtin !== false) {
    if (id === 'maintenance') return t('announcements.typeMaintenance')
    if (id === 'info') return t('announcements.typeInfo')
    if (id === 'update') return t('announcements.typeUpdate')
    if (id === 'warning') return t('announcements.typeWarning')
  }
  return type?.name ?? id
}
