import { api, createApi } from './client';
import type { BodyMetric, ListResponse, ItemResponse } from '$lib/types';

export type CreateBodyMetricInput = {
  weight_kg?: number;
  body_fat_pct?: number;
  muscle_pct?: number;
  visceral_fat?: number;
  recorded_at: string;
  note?: string;
};

export async function createBodyMetric(data: CreateBodyMetricInput): Promise<BodyMetric> {
  const res = await api.post<ItemResponse<BodyMetric>>('/body-metrics', data);
  return res.data;
}

export async function listBodyMetrics(
  params?: { from?: string; to?: string; limit?: number },
  fetchFn?: typeof fetch
): Promise<ListResponse<BodyMetric>> {
  const query = new URLSearchParams();
  if (params?.from) query.set('from', params.from);
  if (params?.to) query.set('to', params.to);
  if (params?.limit) query.set('limit', String(params.limit));
  const qs = query.toString() ? `?${query}` : '';
  const client = fetchFn ? createApi(fetchFn) : api;
  return client.get<ListResponse<BodyMetric>>(`/body-metrics${qs}`);
}

export async function updateBodyMetric(
  id: string,
  data: Partial<CreateBodyMetricInput>
): Promise<BodyMetric> {
  const res = await api.patch<ItemResponse<BodyMetric>>(`/body-metrics/${id}`, data);
  return res.data;
}

export async function deleteBodyMetric(id: string): Promise<void> {
  return api.delete(`/body-metrics/${id}`);
}
