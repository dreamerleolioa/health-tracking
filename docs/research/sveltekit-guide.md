# SvelteKit 學習指南

> 針對本專案的前端學習筆記，重點放在 Svelte / SvelteKit 的核心觀念，以及如何套用到這個健康追蹤專案。

---

## 1. 什麼是 SvelteKit？

SvelteKit 是基於 Svelte 的**全端框架**，類似 Next.js 之於 React。

| 概念     | SvelteKit    | 你可能熟悉的對應   |
| -------- | ------------ | ------------------ |
| 框架     | SvelteKit    | Next.js / Nuxt     |
| UI 語言  | Svelte       | JSX / Vue SFC      |
| 打包工具 | Vite         | Webpack / Vite     |
| 路由     | 檔案系統路由 | Next.js App Router |

Svelte 的最大特點：**沒有 Virtual DOM**，編譯時直接產生操作 DOM 的 JS，效能更好、bundle 更小。

---

## 2. 專案結構

```
frontend/
├── src/
│   ├── app.html              ← HTML 模板（整個 app 的殼）
│   ├── app.css               ← 全域 CSS（只有 @import "tailwindcss"）
│   ├── app.d.ts              ← TypeScript 全域型別宣告
│   ├── lib/
│   │   ├── api/
│   │   │   ├── client.ts     ← fetch 封裝（base URL、error handling、createApi）
│   │   │   ├── body-metrics.ts
│   │   │   ├── sleep-logs.ts
│   │   │   └── daily-activities.ts
│   │   ├── assets/           ← 圖示等靜態資產
│   │   ├── components/       ← 共用元件（目前為空，待拆）
│   │   ├── stores/
│   │   │   └── auth.ts       ← 認證狀態管理（authStore、isLoading）
│   │   └── types/
│   │       └── index.ts      ← 所有前端型別（BodyMetric、SleepLog 等）
│   └── routes/
│       ├── +layout.svelte    ← 全域 layout（導覽列 + auth guard）
│       ├── +page.svelte      ← 首頁（mock 數據，M4 會接真實 API）
│       ├── login/
│       │   └── +page.svelte  ← 登入頁（Google 登入按鈕）
│       ├── auth/
│       │   └── callback/
│       │       └── +page.svelte ← OAuth callback 處理
│       ├── body-metrics/
│       │   ├── +page.svelte  ← 體位數據 CRUD + 趨勢圖
│       │   └── +page.ts      ← 資料載入（body metrics + sleep logs + activities）
│       ├── sleep-logs/
│       │   ├── +page.svelte  ← 睡眠紀錄 CRUD
│       │   └── +page.ts
│       └── daily-activities/
│           ├── +page.svelte  ← 每日活動 CRUD
│           └── +page.ts
├── static/
├── .env                      ← 環境變數（不進 git）
├── .env.example
├── svelte.config.js
├── vite.config.ts
└── tsconfig.json
```

---

## 3. 檔案路由系統

SvelteKit 以 `src/routes/` 內的**資料夾結構**決定 URL：

| 檔案路徑                                         | 對應 URL                    |
| ------------------------------------------------ | --------------------------- |
| `src/routes/+page.svelte`                        | `/`（首頁 dashboard）       |
| `src/routes/login/+page.svelte`                  | `/login`                    |
| `src/routes/auth/callback/+page.svelte`          | `/auth/callback`            |
| `src/routes/body-metrics/+page.svelte`           | `/body-metrics`             |
| `src/routes/sleep-logs/+page.svelte`             | `/sleep-logs`               |
| `src/routes/daily-activities/+page.svelte`       | `/daily-activities`         |

### 特殊檔名規則

| 檔名              | 用途                            |
| ----------------- | ------------------------------- |
| `+page.svelte`    | 頁面元件                        |
| `+layout.svelte`  | 包住子頁面的 layout             |
| `+page.ts`        | 頁面的資料載入（load function） |
| `+page.server.ts` | 僅在 server 執行的資料載入      |
| `+error.svelte`   | 錯誤頁面                        |

---

## 4. Svelte 元件語法

每個 `.svelte` 檔通常分三段：

```svelte
<script lang="ts">
  let count = $state(0);
</script>

<button onclick={() => count++}>
  點了 {count} 次
</button>

<style>
  button { background: blue; }
</style>
```

### Svelte 5 Runes

目前這個專案實際用到的 rune：

| Rune                | 用途                         | 類比                   |
| ------------------- | ---------------------------- | ---------------------- |
| `$state(value)`     | 響應式變數                   | `ref()` in Vue         |
| `$derived(expr)`    | 計算值（單行）               | `computed()` in Vue    |
| `$derived.by(fn)`   | 計算值（複雜邏輯，需要函式） | `computed()` in Vue    |
| `$effect(() => {})` | 依賴變動時執行副作用         | `watchEffect()` in Vue |
| `$props()`          | 接收父元件傳入的 props       | `defineProps()` in Vue |

實際使用範例（取自 `body-metrics/+page.svelte`）：

```svelte
<script lang="ts">
  let { data }: { data: PageData } = $props();

  let showForm = $state(false);
  let submitting = $state(false);

  // $derived：簡單計算
  const maxSteps = $derived(Math.max(1, ...[...stepsMap.values()]));

  // $derived.by：需要多行邏輯（dedup 同一天的資料）
  const chartData = $derived.by(() => {
    const seen = new Set<string>();
    return data.metrics
      .filter(m => {
        const key = new Date(m.recorded_at).toLocaleDateString('en-CA');
        if (seen.has(key)) return false;
        seen.add(key);
        return true;
      })
      .reverse();
  });
</script>
```

---

## 5. 模板控制語法

### 條件渲染

```svelte
{#if showForm}
  <form>...</form>
{/if}

{#if item.enabled}
  <a href={item.href}>{item.label}</a>
{:else}
  <span class="opacity-40">{item.label}</span>
{/if}
```

### 清單渲染

```svelte
{#each data.metrics as m, i (m.id)}
  <tr>...</tr>
{/each}
```

注意 `(m.id)` 是 key，告訴 Svelte 用 id 追蹤每筆資料，避免動畫和狀態跑掉。

### 內嵌運算（`{@const}`）

在模板裡做一次性計算用 `{@const}`，避免重複寫：

```svelte
{#each chartDataWithMeta as point}
  {@const dates = chartDataWithMeta.map(d => new Date(d.recorded_at).getTime())}
  {@const minT = Math.min(...dates)}
  {@const pct = maxT === minT ? 50 : ((curT - minT) / (maxT - minT)) * 100}
  {#if point.isAbnormal}
    <span style="left: {pct}%">▲</span>
  {/if}
{/each}
```

---

## 6. 資料載入（load function）

在 `+page.ts` 定義 `load` function，SvelteKit 在渲染頁面前自動執行。

**實際範例**（`body-metrics/+page.ts`）：

```typescript
import { listBodyMetrics } from '$lib/api/body-metrics';
import { listSleepLogs } from '$lib/api/sleep-logs';
import { listDailyActivities } from '$lib/api/daily-activities';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
  const [metricsRes, sleepRes, activityRes] = await Promise.allSettled([
    listBodyMetrics({ limit: 90 }, fetch),
    listSleepLogs({}, fetch),
    listDailyActivities({}, fetch),
  ]);

  return {
    metrics: metricsRes.status === 'fulfilled' ? metricsRes.value.data : [],
    meta: metricsRes.status === 'fulfilled' ? metricsRes.value.meta : { total: 0 },
    sleepLogs: sleepRes.status === 'fulfilled' ? sleepRes.value.data : [],
    activities: activityRes.status === 'fulfilled' ? activityRes.value.data : [],
  };
};
```

重點：

- `Promise.allSettled` 讓三個 API 並行請求，任何一個失敗不會影響其他
- `fetch` 是 SvelteKit 提供的 SSR 版 fetch，**一定要透過 API 函式傳進去**，不能直接用全域 `fetch`（見下節）
- `+page.svelte` 用 `let { data } = $props()` 接收 load 回傳的資料

### `+page.ts` vs `+page.server.ts`

- `+page.ts`：可跑在 server + client（導頁時也會跑）
- `+page.server.ts`：只跑在 server（可用 server-only 環境變數）

這個專案 API 是獨立 Go backend，前端多用 `+page.ts`。

---

## 7. API Client 架構

### 三層結構

```
client.ts            ← 最底層：fetch 封裝、base URL、error handling
body-metrics.ts      ← 語意化 API：createBodyMetric、listBodyMetrics…
+page.ts / 頁面      ← 呼叫 API 函式，處理 UI 狀態
```

### SSR fetch 的問題與解法

SvelteKit 在 `+page.ts` 的 `load` 函式裡，`fetch` 是框架提供的增強版（帶 cookies、可 SSR）。如果直接用全域 `fetch`，SSR 時 cookie 帶不進去，會導致 401。

解法：API 函式接受一個 `fetchFn` 參數：

```typescript
// body-metrics.ts
export async function listBodyMetrics(
  params?: { from?: string; to?: string; limit?: number },
  fetchFn?: typeof fetch          // ← 接受 SvelteKit 的 fetch
): Promise<ListResponse<BodyMetric>> {
  const client = fetchFn ? createApi(fetchFn) : api;  // ← 用傳進來的 fetch
  return client.get<ListResponse<BodyMetric>>(`/body-metrics${qs}`);
}
```

在 `+page.ts` 呼叫時把 fetch 傳進去：

```typescript
export const load: PageLoad = async ({ fetch }) => {
  return listBodyMetrics({ limit: 90 }, fetch);  // ← 傳入 SvelteKit fetch
};
```

在元件內直接呼叫（互動後重抓）時不傳 fetch（用全域 fetch 即可）：

```typescript
await createBodyMetric({ ... });   // 不需要傳 fetch
```

---

## 8. Auth Guard（認證守衛）

`+layout.svelte` 負責全域的認證狀態管理，未登入者會自動導向 `/login`：

```svelte
<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { authStore, isLoading } from '$lib/stores/auth';

  const PUBLIC_ROUTES = ['/login', '/auth/callback'];
  const isPublicRoute = $derived(
    PUBLIC_ROUTES.some(r => $page.url.pathname.startsWith(r))
  );

  onMount(async () => {
    await authStore.init();  // 頁面載入時驗證目前 token
  });

  $effect(() => {
    if (!$isLoading && !$authStore && !isPublicRoute) {
      goto('/login');  // 未登入且不是公開頁面 → 導向登入
    }
  });
</script>
```

`authStore` 封裝在 `src/lib/stores/auth.ts`，負責：
- `init()`：呼叫 `GET /v1/auth/me` 確認 token 是否有效
- `logout()`：呼叫 `POST /v1/auth/logout` 並清除狀態

---

## 9. 環境變數

SvelteKit 有兩種環境變數：

| 前綴      | 可在哪裡用              | 範例                  |
| --------- | ----------------------- | --------------------- |
| `PUBLIC_` | 前端 + 後端都能用       | `PUBLIC_API_BASE_URL` |
| 無前綴    | 只能在 server 端        | `DATABASE_URL`        |

```typescript
// src/lib/api/client.ts
import { PUBLIC_API_BASE_URL } from '$env/static/public';
```

重點：只要需要在瀏覽器端讀取，就必須加 `PUBLIC_` 前綴。

---

## 10. 表單處理

這個專案的表單統一採用以下模式（取自 `body-metrics/+page.svelte`）：

```svelte
<script lang="ts">
  let showForm = $state(false);
  let submitting = $state(false);
  let weight_kg = $state('');

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    submitting = true;
    try {
      await createBodyMetric({
        weight_kg: parseFloat(weight_kg),
        recorded_at: new Date(recorded_at).toISOString(),
      });
      showForm = false;
      await invalidateAll();    // ← 重新跑 load function，更新列表
    } catch (err) {
      alert('新增失敗：' + (err instanceof Error ? err.message : '未知錯誤'));
    } finally {
      submitting = false;
    }
  }
</script>

<form onsubmit={handleSubmit}>
  <input type="number" step="0.1" bind:value={weight_kg} />
  <button type="submit" disabled={submitting}>
    {submitting ? '儲存中…' : '儲存'}
  </button>
</form>
```

### `invalidateAll()`

送出表單後，呼叫 `invalidateAll()` 讓 SvelteKit 重新執行頁面的 `load` function，資料列表就會自動更新。不需要手動更新本地 state 陣列。

### 表單狀態三件套

每個表單至少有三種狀態：

| 狀態        | 說明                          |
| ----------- | ----------------------------- |
| 欄位值      | `let weight_kg = $state('')`  |
| 送出中      | `let submitting = $state(false)` |
| 顯示/隱藏   | `let showForm = $state(false)` |

---

## 11. Svelte Transitions（過場動畫）

這個專案已大量使用 Svelte 內建的 transition 指令：

```svelte
<script lang="ts">
  import { slide, fade, fly } from 'svelte/transition';
</script>

<!-- 新增表單展開/收合 -->
{#if showForm}
  <div transition:slide={{ duration: 180 }}>
    <form>...</form>
  </div>
{/if}

<!-- Modal 背景遮罩淡入淡出 -->
{#if confirmDeleteId}
  <button transition:fade={{ duration: 150 }} class="fixed inset-0 bg-black/60">
  </button>

  <!-- Modal 本體由下往上滑入 -->
  <div transition:fly={{ y: 16, duration: 200 }}>
    <div>確認刪除...</div>
  </div>
{/if}
```

| Transition | 效果           | 常見用途           |
| ---------- | -------------- | ------------------ |
| `slide`    | 展開/收合      | inline 表單        |
| `fade`     | 淡入淡出       | 遮罩、提示訊息     |
| `fly`      | 帶位移的淡入   | Modal、Drawer      |

---

## 12. SSR 與瀏覽器限定邏輯

SvelteKit 有 SSR 能力，有些 API 只有瀏覽器才有（如 `window`、`localStorage`）。

```typescript
import { browser } from '$app/environment';

if (browser) {
  // 只有瀏覽器端才執行
}
```

這個專案的圖表就用了這個模式：

```svelte
{#if browser}
  <div class="h-[240px]">
    <LineChart ... />
  </div>
{/if}
```

原因：`layerchart` 圖表依賴 DOM API，SSR 時會報錯，必須限制在瀏覽器端執行。

---

## 13. 圖表（layerchart）

目前 `body-metrics/+page.svelte` 使用 `layerchart` 的 `LineChart`：

```svelte
<script lang="ts">
  import { LineChart } from 'layerchart';
</script>

{#if browser}
  <div class="h-[240px]">
    <LineChart
      data={chartData}
      x={(d: BodyMetric) => new Date(d.recorded_at)}
      series={[
        { key: 'weight_kg', label: '體重 (kg)', color: '#0EA5E9' },
        { key: 'body_fat_pct', label: '體脂率 (%)', color: '#F59E0B' },
        { key: 'muscle_pct', label: '肌肉率 (%)', color: '#10B981' },
      ]}
    />
  </div>
{/if}
```

注意：
- `data` 必須是 **ASC** 排序（x 軸從左到右）；API 回傳是 DESC，所以 load 後要 `.reverse()`
- 同一天多筆資料要 dedup，只取最新一筆（用 `$derived.by` 加 Set 過濾）
- 圖表外層必須有明確高度，否則不會顯示

---

## 14. 反應式思維：state vs derived

### 原則

- 能推導出來的值，就不要重複存
- state 越少，bug 越少

```svelte
<script lang="ts">
  // ✅ 只有「原始資料」放 state
  let { data } = $props();

  // ✅ 從原始資料推導出來的，用 derived
  const latestMetric = $derived(data.metrics[0]);
  const chartData = $derived.by(() => { /* dedup + reverse */ });
  const abnormalDates = $derived(new Set(
    data.sleepLogs.filter(l => l.abnormal_wake).map(...)
  ));
</script>
```

---

## 15. 常見坑

### 1. 不傳 SvelteKit fetch 給 API 函式

在 `+page.ts` 的 `load` 裡呼叫 API 時，一定要把 `fetch` 傳給 API 函式，否則 SSR 時 cookie 帶不進去會 401。

### 2. 圖表忘記用 `{#if browser}` 包起來

SSR 時 layerchart 會嘗試存取 DOM 導致錯誤。

### 3. 圖表資料忘記轉成 ASC

API 回傳 DESC（最新在前），但 LineChart 需要 ASC（時間由小到大）。load 後要 `.reverse()`。

### 4. 前端環境變數改了卻沒重啟 dev server

`.env` 修改後需要重啟才生效。

### 5. 把太多責任塞進頁面元件

一個頁面如果同時做 API 呼叫、表單驗證、錯誤處理、圖表轉換，很快會難維護。API 邏輯放 `lib/api/`，型別放 `lib/types/`。

### 6. `invalidateAll` 遺漏

新增、編輯、刪除後記得呼叫 `invalidateAll()`，不然資料不會更新。

---

## 16. 啟動開發伺服器

```bash
cd frontend
pnpm serve
```

開啟 `http://localhost:5173`

---

## 17. 常用指令

| 指令               | 用途                         |
| ------------------ | ---------------------------- |
| `pnpm serve`       | 啟動開發伺服器（hot reload） |
| `pnpm build`       | 打包正式版                   |
| `pnpm preview`     | 預覽打包結果                 |
| `pnpm check`       | TypeScript 型別檢查          |
| `pnpm check:watch` | 持續監看型別與 Svelte 檢查   |

---

## 18. 專案閱讀順序

1. `src/routes/+layout.svelte`：理解 auth guard、導覽列與頁面容器
2. `src/lib/api/client.ts`：理解 fetch 封裝與 createApi 模式
3. `src/lib/stores/auth.ts`：理解 authStore 怎麼管理登入狀態
4. `src/routes/body-metrics/+page.ts`：理解 load function 與 Promise.allSettled 並行載入
5. `src/routes/body-metrics/+page.svelte`：理解完整的 CRUD 頁面（表單、列表、Modal、圖表）
6. `src/routes/sleep-logs/+page.svelte`：對比看看另一個功能是怎麼用同樣的模式實作
7. `src/routes/+page.svelte`：目前還是 mock 數據，M4 會接真實 API

---

## 19. 目前摘要

目前前端已完成：

- **認證流程**：Google OAuth 登入、auth guard、登出
- **三個完整 CRUD 頁面**：體位數據、睡眠紀錄、每日活動
  - 每頁都有：新增表單、Desktop table / Mobile cards、Edit modal、Delete confirmation modal
  - 所有操作都有 loading state 與錯誤提示
- **體位數據頁有完整圖表**：30 天趨勢折線圖 + 異常睡眠標記 + 步數熱度條

待完成：

- **首頁 Dashboard**（M4）：把 mock 數據替換成真實 API 資料，並加入趨勢圖

---

## 20. 學習資源

- [Svelte 官方互動教學](https://learn.svelte.dev)（強烈推薦，30 分鐘內上手）
- [SvelteKit 官方文件](https://kit.svelte.dev/docs)
- [Svelte 5 Runes 說明](https://svelte.dev/docs/svelte/what-are-runes)
- [layerchart 文件](https://layerchart.com)
