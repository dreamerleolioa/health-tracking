# 首頁 Dashboard 串接真實數據

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 將首頁 (`/`) 的 mock 數據替換成真實 API 資料，並實作 30 天體重趨勢圖（含睡眠異常標記 + 步數熱度條）。

**Architecture:** 在 `frontend/src/routes/+page.ts` 以 SvelteKit SSR fetch 並行載入最近 30 天的 body metrics、sleep logs、daily activities；`+page.svelte` 使用 `layerchart` LineChart 繪製趨勢圖，模式與 `body-metrics/+page.svelte` 一致。`listSleepLogs` / `listDailyActivities` 需補上 `fetchFn` 參數以支援 SSR 呼叫。

**Tech Stack:** SvelteKit 5 (`$props`, `$derived`, `$state`)、layerchart `LineChart`、Tailwind CSS

---

## 背景

| 檔案 | 現狀 |
|------|------|
| `frontend/src/routes/+page.svelte` | 使用 hardcoded mock 數據，趨勢圖是 placeholder |
| `frontend/src/routes/+page.ts` | **不存在** |
| `frontend/src/lib/api/sleep-logs.ts` | `listSleepLogs` 缺少 `fetchFn` 參數 |
| `frontend/src/lib/api/daily-activities.ts` | `listDailyActivities` 缺少 `fetchFn` 參數 |

參考實作：`frontend/src/routes/body-metrics/+page.ts` 和 `body-metrics/+page.svelte` 已有完整的圖表 + anomaly marker + steps heatmap 模式，直接沿用。

---

## Task 1：補上 `fetchFn` 支援

**Files:**
- Modify: `frontend/src/lib/api/sleep-logs.ts`
- Modify: `frontend/src/lib/api/daily-activities.ts`

`listBodyMetrics` 已有 `fetchFn?: typeof fetch` 參數作為範本：

```typescript
// sleep-logs.ts 修改後
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
```

```typescript
// daily-activities.ts 修改後
import { api, createApi } from './client'
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

export async function listDailyActivities(
  params?: { from?: string; to?: string },
  fetchFn?: typeof fetch
): Promise<ListResponse<DailyActivity>> {
  const query = new URLSearchParams()
  if (params?.from) query.set('from', params.from)
  if (params?.to) query.set('to', params.to)
  const qs = query.toString() ? `?${query}` : ''
  const client = fetchFn ? createApi(fetchFn) : api
  return client.get<ListResponse<DailyActivity>>(`/daily-activities${qs}`)
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
```

**Step 1: 修改 `sleep-logs.ts`**

在 `import` 行加入 `createApi`，並在 `listSleepLogs` 加上 `fetchFn` 參數與 `const client = fetchFn ? createApi(fetchFn) : api`，改用 `client.get(...)` 呼叫。

**Step 2: 修改 `daily-activities.ts`**

同上模式。

**Step 3: 確認 TypeScript 無錯誤**

```bash
cd frontend && npx tsc --noEmit
```
預期：無錯誤輸出。

**Step 4: ⏸ 暫停，請手動 commit**

```
git add frontend/src/lib/api/sleep-logs.ts frontend/src/lib/api/daily-activities.ts
git commit -m "feat: add fetchFn support to listSleepLogs and listDailyActivities"
```

---

## Task 2：建立首頁 `+page.ts` 資料載入

**Files:**
- Create: `frontend/src/routes/+page.ts`

```typescript
import { listBodyMetrics } from '$lib/api/body-metrics';
import { listSleepLogs } from '$lib/api/sleep-logs';
import { listDailyActivities } from '$lib/api/daily-activities';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
  const today = new Date();
  const from = new Date(today);
  from.setDate(today.getDate() - 30);
  const fromStr = from.toISOString().slice(0, 10);
  const toStr = today.toISOString().slice(0, 10);

  const [metricsRes, sleepRes, activityRes] = await Promise.allSettled([
    listBodyMetrics({ from: fromStr, to: toStr, limit: 90 }, fetch),
    listSleepLogs({ from: fromStr, to: toStr }, fetch),
    listDailyActivities({ from: fromStr, to: toStr }, fetch),
  ]);

  return {
    metrics: metricsRes.status === 'fulfilled' ? metricsRes.value.data : [],
    sleepLogs: sleepRes.status === 'fulfilled' ? sleepRes.value.data : [],
    activities: activityRes.status === 'fulfilled' ? activityRes.value.data : [],
  };
};
```

**Step 1: 建立 `+page.ts`**

複製上方程式碼到 `frontend/src/routes/+page.ts`。

**Step 2: 確認 TypeScript 無錯誤**

```bash
cd frontend && npx tsc --noEmit
```

**Step 3: 啟動 dev server，確認首頁可正常載入（F12 Network 無 4xx/5xx）**

```bash
cd frontend && npm run dev
```

在瀏覽器開啟 `http://localhost:5173`，確認頁面不 crash。

**Step 4: ⏸ 暫停，請手動 commit**

```
git add frontend/src/routes/+page.ts
git commit -m "feat: add home page data loader with 30-day range"
```

---

## Task 3：更新首頁數據卡片（替換 mock 數據）

**Files:**
- Modify: `frontend/src/routes/+page.svelte`

**說明：**
- 從 `data.metrics` 取第一筆（最新一筆，API 回傳 ORDER BY recorded_at DESC）作為 "今日" 數值
- 若無資料，顯示 `—`
- `data.sleepLogs[0]` 為最近一筆睡眠紀錄
- `data.activities` 找 `activity_date === today` 的那筆

**Step 1: 修改 `<script>` 區塊**

將原本的 mock `metrics` 陣列，改成從 `data` props 計算的 `$derived` 值：

```svelte
<script lang="ts">
  import type { PageData } from './$types';
  import type { BodyMetric } from '$lib/types';

  let { data }: { data: PageData } = $props();

  const today = new Date().toLocaleDateString('zh-TW', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    weekday: 'long',
  });

  const todayStr = new Date().toISOString().slice(0, 10);

  // 最新一筆 body metric（API 回傳 DESC，第一筆最新）
  const latest = $derived(data.metrics[0] as BodyMetric | undefined);

  const metricCards = $derived([
    {
      label: '體重',
      value: latest?.weight_kg != null ? String(latest.weight_kg) : '—',
      unit: latest?.weight_kg != null ? 'kg' : '',
      prefix: '',
      color: '#0EA5E9',
    },
    {
      label: '體脂率',
      value: latest?.body_fat_pct != null ? String(latest.body_fat_pct) : '—',
      unit: latest?.body_fat_pct != null ? '%' : '',
      prefix: '',
      color: '#F59E0B',
    },
    {
      label: '肌肉率',
      value: latest?.muscle_pct != null ? String(latest.muscle_pct) : '—',
      unit: latest?.muscle_pct != null ? '%' : '',
      prefix: '',
      color: '#10B981',
    },
    {
      label: '內臟脂肪',
      value: latest?.visceral_fat != null ? String(latest.visceral_fat) : '—',
      unit: '',
      prefix: latest?.visceral_fat != null ? 'Lv.' : '',
      color: '#8B5CF6',
    },
  ]);

  // 今日活動步數
  const todayActivity = $derived(
    data.activities.find(a => a.activity_date === todayStr)
  );

  // 最近睡眠
  const latestSleep = $derived(data.sleepLogs[0]);
</script>
```

**Step 2: 更新 HTML 使用 `metricCards` 取代舊 `metrics`**

template 中的 `{#each metrics as metric}` 改為 `{#each metricCards as metric}`，其餘結構不變。

**Step 3: 在現有卡片區塊下方加入今日活動摘要區塊（選填 row）**

```svelte
<!-- 今日摘要 -->
{#if todayActivity || latestSleep}
  <div class="grid grid-cols-2 gap-4 mb-8">
    {#if latestSleep}
      <div class="bg-white/5 rounded-2xl p-4">
        <p class="text-gray-400 text-xs tracking-wide mb-2">上次睡眠</p>
        <p class="text-white font-bold text-lg">
          {latestSleep.duration_min != null
            ? `${Math.floor(latestSleep.duration_min / 60)}h ${latestSleep.duration_min % 60}m`
            : '—'}
        </p>
        {#if latestSleep.abnormal_wake}
          <p class="text-orange-400 text-xs mt-1">▲ 異常喚醒</p>
        {/if}
      </div>
    {/if}
    {#if todayActivity?.steps != null}
      <div class="bg-white/5 rounded-2xl p-4">
        <p class="text-gray-400 text-xs tracking-wide mb-2">今日步數</p>
        <p class="text-white font-bold text-lg">{todayActivity.steps.toLocaleString()}</p>
        <p class="text-gray-500 text-xs mt-1">步</p>
      </div>
    {/if}
  </div>
{/if}
```

**Step 4: 驗證頁面**

啟動 dev server，以登入狀態開啟首頁，確認：
- 數據卡片顯示真實數值（或 `—`）
- 今日摘要區塊正常顯示

**Step 5: ⏸ 暫停，請手動 commit**

```
git add frontend/src/routes/+page.svelte
git commit -m "feat: connect home page metric cards to real API data"
```

---

## Task 4：首頁趨勢圖（30 天）

**Files:**
- Modify: `frontend/src/routes/+page.svelte`

**說明：** 沿用 `body-metrics/+page.svelte` 的圖表模式：`layerchart` LineChart + 異常睡眠標記條 + 步數熱度條。

**Step 1: 在 `<script>` 加入圖表相關 import 與 derived**

```svelte
<script lang="ts">
  import { browser } from '$app/environment';
  import { LineChart } from 'layerchart';
  // ... 其他已有 import

  // 圖表用：同一天只取最新一筆（metrics 已 DESC，取第一個 key 出現者）
  const chartData = $derived.by(() => {
    const seen = new Set<string>();
    return [...data.metrics]
      .filter(m => {
        const key = new Date(m.recorded_at).toLocaleDateString('en-CA');
        if (seen.has(key)) return false;
        seen.add(key);
        return true;
      })
      .reverse(); // 轉成 ASC 供圖表 x 軸使用
  });

  const abnormalDates = $derived(
    new Set(
      data.sleepLogs
        .filter(l => l.abnormal_wake)
        .map(l => new Date(l.wake_at).toLocaleDateString('en-CA'))
    )
  );

  const stepsMap = $derived(
    new Map(
      data.activities
        .filter(a => a.steps != null)
        .map(a => [a.activity_date, a.steps as number])
    )
  );

  const maxSteps = $derived(Math.max(1, ...[...stepsMap.values()]));

  const chartDataWithMeta = $derived(
    chartData.map(m => {
      const dateKey = new Date(m.recorded_at).toLocaleDateString('en-CA');
      return {
        ...m,
        isAbnormal: abnormalDates.has(dateKey),
        steps: stepsMap.get(dateKey) ?? 0,
      };
    })
  );
</script>
```

**Step 2: 替換趨勢圖 placeholder**

找到目前 `<!-- 趨勢圖區 -->` 區塊，替換為：

```svelte
<!-- 趨勢圖區 -->
{#if browser}
  <div class="rounded-2xl bg-white overflow-hidden shadow-md mb-8">
    <div class="flex items-center justify-between px-5 pt-4 pb-3 border-b border-gray-100">
      <div class="flex items-baseline gap-2">
        <span class="font-black tracking-widest text-sm text-gray-800">TRENDS</span>
        <span class="text-gray-400 text-xs">近 30 天</span>
      </div>
      <div class="flex items-center gap-4 text-xs text-gray-500">
        <span class="flex items-center gap-1.5">
          <span class="w-2.5 h-2.5 rounded-full bg-[#0EA5E9]"></span>體重
        </span>
        <span class="flex items-center gap-1.5">
          <span class="w-2.5 h-2.5 rounded-full bg-[#F59E0B]"></span>體脂率
        </span>
        <span class="flex items-center gap-1.5">
          <span class="w-2.5 h-2.5 rounded-full bg-[#10B981]"></span>肌肉率
        </span>
      </div>
    </div>

    {#if chartData.length >= 2}
      <div class="h-[240px]">
        <LineChart
          data={chartData}
          x={(d: BodyMetric) => new Date(d.recorded_at)}
          series={[
            { key: 'weight_kg', label: '體重 (kg)', color: '#0EA5E9' },
            { key: 'body_fat_pct', label: '體脂率 (%)', color: '#F59E0B' },
            { key: 'muscle_pct', label: '肌肉率 (%)', color: '#10B981' },
          ]}
          props={{
            tooltip: {
              root: {
                class: 'bg-white border border-gray-200 shadow-lg rounded-lg text-xs pointer-events-none',
              },
            },
          }}
        />
      </div>

      <!-- 異常睡眠標記條 -->
      <div class="relative h-6 px-10 flex items-center">
        {#each chartDataWithMeta as point}
          {@const dates = chartDataWithMeta.map(d => new Date(d.recorded_at).getTime())}
          {@const minT = Math.min(...dates)}
          {@const maxT = Math.max(...dates)}
          {@const curT = new Date(point.recorded_at).getTime()}
          {@const pct = maxT === minT ? 50 : ((curT - minT) / (maxT - minT)) * 100}
          {#if point.isAbnormal}
            <span
              class="absolute text-orange-500 text-[10px] leading-none -translate-x-1/2"
              style="left: {pct}%"
              title="異常喚醒 {new Date(point.recorded_at).toLocaleDateString('zh-TW')}"
            >▲</span>
          {/if}
        {/each}
      </div>

      <!-- 步數熱度條 -->
      <div class="relative h-3 px-10 flex items-stretch mb-1 rounded-b overflow-hidden">
        {#each chartDataWithMeta as point}
          {@const dates = chartDataWithMeta.map(d => new Date(d.recorded_at).getTime())}
          {@const minT = Math.min(...dates)}
          {@const maxT = Math.max(...dates)}
          {@const curT = new Date(point.recorded_at).getTime()}
          {@const pct = maxT === minT ? 50 : ((curT - minT) / (maxT - minT)) * 100}
          {@const opacity = point.steps > 0 ? 0.15 + (point.steps / maxSteps) * 0.6 : 0}
          <div
            class="absolute inset-y-0 w-2 -translate-x-1/2 bg-emerald-500 rounded-sm"
            style="left: {pct}%; opacity: {opacity}"
            title="步數 {point.steps.toLocaleString()}"
          ></div>
        {/each}
      </div>
    {:else}
      <div class="h-[200px] flex flex-col items-center justify-center gap-1 text-gray-400">
        <p class="text-sm font-medium">資料不足</p>
        <p class="text-xs">新增 2 筆以上體位數據即可查看趨勢</p>
      </div>
    {/if}
  </div>
{/if}
```

**Step 3: 確認 TypeScript 無錯誤**

```bash
cd frontend && npx tsc --noEmit
```

**Step 4: 在瀏覽器驗證**

- 有資料時：圖表正常顯示三條折線，異常睡眠日期顯示橘色 ▲，步數顯示熱度條
- 資料少於 2 筆時：顯示 "資料不足" 提示

**Step 5: ⏸ 暫停，請手動 commit**

```
git add frontend/src/routes/+page.svelte
git commit -m "feat: implement 30-day trend chart on home dashboard"
```

---

## Task 5：更新 SRS 標記 M1、M2、M4 完成

**Files:**
- Modify: `docs/SRS.md`

**Step 1: 在 SRS 加入 Milestone 4 區塊並標記完成**

在 `### Milestone 3` 與 `### Milestone 5` 之間新增：

```markdown
### Milestone 4 — 首頁 Dashboard（第 9 週）

- [x] 首頁 `+page.ts` 資料載入（並行呼叫三支 API，30 天範圍）
- [x] 首頁數據卡片接上真實 body metrics 最新值
- [x] 首頁今日睡眠摘要 + 今日步數摘要區塊
- [x] 首頁 30 天體重趨勢圖（layerchart LineChart）
- [x] 異常睡眠標記疊加至首頁趨勢圖
- [x] 步數熱度條疊加至首頁趨勢圖
- [x] `listSleepLogs` / `listDailyActivities` 補上 `fetchFn` 支援（SSR 相容）
```

同時將 M1、M2 全部 checklist 標記為 `[x]`（對應功能已在先前 commit 實作完成）。

**Step 2: ⏸ 暫停，請手動 commit**

```
git add docs/SRS.md
git commit -m "docs: mark M1, M2, M4 as complete in SRS"
```

---

## 完成驗收標準

| 驗收項目 | 方法 |
|----------|------|
| 首頁卡片顯示最新 body metric 真實數值 | 瀏覽器目測 |
| 首頁卡片無資料時顯示 `—` | 刪除所有 body_metrics 後重整（測完再還原）|
| 趨勢圖在有 ≥2 筆資料時正常繪製 | 瀏覽器目測 |
| 趨勢圖在 < 2 筆時顯示提示文字 | 瀏覽器目測 |
| 異常睡眠日顯示橘色 ▲ | 確認 sleep_log 中有 abnormal_wake=true 的資料 |
| `npx tsc --noEmit` 無錯誤 | Terminal |
