import { api, createApi } from './client'
import type { SleepLog, ListResponse, ItemResponse } from '$lib/types'

export type CreateSleepLogInput = {
  sleep_at: string
  wake_at: string
  quality?: number
  note?: string
}

export async function createSleepLog(data: CreateSleepLogInput): Promise<SleepLog> {
  const res = await api.post<ItemResponse<SleepLog>>('/sleep-logs', data)
  return res.data
}

export async function listSleepLogs(
  params?: { from?: string; to?: string; abnormal_only?: boolean },
  fetchFn?: typeof fetch
): Promise<ListResponse<SleepLog>> {
  const query = new URLSearchParams()
  if (params?.from) query.set('from', params.from)
  if (params?.to) query.set('to', params.to)
  if (params?.abnormal_only) query.set('abnormal_only', 'true')
  const qs = query.toString() ? `?${query}` : ''
  const client = fetchFn ? createApi(fetchFn) : api
  return client.get<ListResponse<SleepLog>>(`/sleep-logs${qs}`)
}

export async function updateSleepLog(
  id: string,
  data: Partial<CreateSleepLogInput>
): Promise<SleepLog> {
  const res = await api.patch<ItemResponse<SleepLog>>(`/sleep-logs/${id}`, data)
  return res.data
}

export async function deleteSleepLog(id: string): Promise<void> {
  return api.delete(`/sleep-logs/${id}`)
}
