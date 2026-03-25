import { api } from './client'
import type { DailyActivity, CommuteMode, ListResponse, ItemResponse } from '$lib/types'

export type CreateDailyActivityInput = {
  activity_date: string
  steps?: number
  commute_mode?: CommuteMode
  commute_minutes?: number
  note?: string
}

export async function createDailyActivity(data: CreateDailyActivityInput): Promise<DailyActivity> {
  const res = await api.post<ItemResponse<DailyActivity>>('/daily-activities', data)
  return res.data
}

export async function listDailyActivities(params?: {
  from?: string
  to?: string
}): Promise<ListResponse<DailyActivity>> {
  const query = new URLSearchParams()
  if (params?.from) query.set('from', params.from)
  if (params?.to) query.set('to', params.to)
  const qs = query.toString() ? `?${query}` : ''
  return api.get<ListResponse<DailyActivity>>(`/daily-activities${qs}`)
}

export async function updateDailyActivity(
  id: string,
  data: Partial<Omit<CreateDailyActivityInput, 'activity_date'>>
): Promise<DailyActivity> {
  const res = await api.patch<ItemResponse<DailyActivity>>(`/daily-activities/${id}`, data)
  return res.data
}

export async function deleteDailyActivity(id: string): Promise<void> {
  return api.delete(`/daily-activities/${id}`)
}
