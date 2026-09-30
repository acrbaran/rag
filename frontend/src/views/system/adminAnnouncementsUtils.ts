export interface SubmissionKey {
  fingerprint: string
  key: string
}

export function datetimeLocalValue(timestamp: number | null): string {
  if (timestamp == null) return ''
  const date = new Date(timestamp)
  if (Number.isNaN(date.getTime())) return ''
  const pad = (value: number) => String(value).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

export function datetimeTimestamp(value: string): number | null {
  if (!value) return null
  const timestamp = new Date(value).getTime()
  return Number.isNaN(timestamp) ? null : timestamp
}

function randomKey(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') return crypto.randomUUID()
  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}-${Math.random().toString(36).slice(2)}`
}

export function stableSubmissionKey(previous: SubmissionKey | null, fingerprint: string): SubmissionKey {
  return previous?.fingerprint === fingerprint ? previous : { fingerprint, key: randomKey() }
}
