import { get, put } from '@/utils/request'

export type Feedback = {
  id: string
  session_id: string
  tenant_id: number
  user_id: string
  helpful: boolean | null
  category: 'suggestion' | 'complaint' | null
  comment: string | null
  created_at: string
  updated_at: string
  session_title?: string
  user_name?: string
  user_email?: string
  workspace_name?: string
}

export const getFeedback = (sessionId: string) => get(`/api/v1/sessions/${encodeURIComponent(sessionId)}/feedback`)
export const saveFeedback = (sessionId: string, data: Partial<Pick<Feedback, 'helpful' | 'category' | 'comment'>>) =>
  put(`/api/v1/sessions/${encodeURIComponent(sessionId)}/feedback`, data)
export const listFeedback = (global: boolean, query: URLSearchParams) =>
  get(`/api/v1/${global ? 'system/admin/feedback' : 'feedback'}?${query}`)
export const getFeedbackDetail = (global: boolean, id: string) =>
  get(`/api/v1/${global ? 'system/admin/feedback' : 'feedback'}/${encodeURIComponent(id)}`)
