# SvelteKit 學習指南

> 針對本專案的 SvelteKit 入門，從零開始說明核心概念。

---

## 1. 什麼是 SvelteKit？

SvelteKit 是基於 Svelte 的**全端框架**，類似 Next.js 之於 React。

| 概念 | SvelteKit | 你可能熟悉的對應 |
|------|-----------|----------------|
| 框架 | SvelteKit | Next.js / Nuxt |
| UI 語言 | Svelte | JSX / Vue SFC |
| 打包工具 | Vite | Webpack / Vite |
| 路由 | 檔案系統路由 | Next.js App Router |

Svelte 的最大特點：**沒有 Virtual DOM**，編譯時直接產生操作 DOM 的 JS，效能更好、bundle 更小。

---

## 2. 專案結構

```
frontend/
├── src/
│   ├── app.html          ← HTML 模板（整個 app 的殼）
│   ├── app.d.ts          ← TypeScript 全域型別宣告
│   ├── lib/              ← 共用元件、工具、API client
│   │   └── index.ts      ← $lib 的入口（可 re-export）
│   └── routes/           ← 頁面路由（檔案即路由）
│       ├── +layout.svelte    ← 全域 layout（導覽列等）
│       └── +page.svelte      ← 首頁 /
├── static/               ← 靜態資源（favicon 等）
├── .env                  ← 環境變數（不進 git）
├── .env.example          ← 範本（進 git）
├── svelte.config.js      ← SvelteKit 設定
├── vite.config.ts        ← Vite 設定
└── tsconfig.json         ← TypeScript 設定
```

---

## 3. 檔案路由系統

SvelteKit 以 `src/routes/` 內的**資料夾結構**決定 URL：

| 檔案路徑 | 對應 URL |
|----------|---------|
| `src/routes/+page.svelte` | `/` |
| `src/routes/body-metrics/+page.svelte` | `/body-metrics` |
| `src/routes/body-metrics/[id]/+page.svelte` | `/body-metrics/:id` |
| `src/routes/sleep/+page.svelte` | `/sleep` |

### 特殊檔名規則

| 檔名 | 用途 |
|------|------|
| `+page.svelte` | 頁面元件 |
| `+layout.svelte` | 包住子頁面的 layout |
| `+page.ts` | 頁面的資料載入（load function） |
| `+page.server.ts` | 僅在 server 執行的資料載入 |
| `+error.svelte` | 錯誤頁面 |

---

## 4. Svelte 元件語法

每個 `.svelte` 檔分三段：

```svelte
<script lang="ts">
  // JS / TypeScript 邏輯
  let count = $state(0);  // 響應式狀態（Svelte 5 runes 語法）
</script>

<!-- HTML 模板 -->
<button onclick={() => count++}>
  點了 {count} 次
</button>

<style>
  /* CSS（自動 scoped，不會汙染其他元件） */
  button {
    background: blue;
  }
</style>
```

### Svelte 5 Runes（本專案使用的語法）

| Rune | 用途 | 類比 |
|------|------|------|
| `$state(value)` | 響應式變數 | `ref()` in Vue |
| `$derived(expr)` | 計算值 | `computed()` in Vue |
| `$effect(() => {})` | 副作用 | `watch()` in Vue |
| `$props()` | 接收父元件傳入的 props | `defineProps()` in Vue |

---

## 5. 資料載入（load function）

在 `+page.ts` 定義 `load` function，SvelteKit 在渲染頁面前自動執行：

```typescript
// src/routes/body-metrics/+page.ts
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
  const res = await fetch('/v1/body-metrics');
  const data = await res.json();
  return { metrics: data.data };
};
```

```svelte
<!-- src/routes/body-metrics/+page.svelte -->
<script lang="ts">
  let { data } = $props();
  // data.metrics 就是 load function 回傳的資料
</script>

{#each data.metrics as metric}
  <p>{metric.weight_kg} kg</p>
{/each}
```

---

## 6. 環境變數

SvelteKit 有兩種環境變數：

| 前綴 | 可在哪裡用 | 範例 |
|------|-----------|------|
| `PUBLIC_` | 前端 + 後端都能用 | `PUBLIC_API_BASE_URL` |
| 無前綴 | 只能在 server 端（`+page.server.ts`） | `DATABASE_URL` |

使用方式：

```typescript
import { PUBLIC_API_BASE_URL } from '$env/static/public';

const res = await fetch(`${PUBLIC_API_BASE_URL}/body-metrics`);
```

---

## 7. 本專案的 src/ 規劃

```
src/
├── lib/
│   ├── api/              ← API client（呼叫後端的函式）
│   │   ├── body-metrics.ts
│   │   ├── sleep-logs.ts
│   │   └── client.ts     ← fetch 封裝（base URL、error handling）
│   ├── components/       ← 共用 UI 元件
│   │   ├── charts/       ← 圖表元件
│   │   └── ui/           ← Button、Card 等基礎元件
│   └── types/            ← TypeScript 型別定義（對應後端 API schema）
│       └── index.ts
└── routes/
    ├── +layout.svelte    ← 全域 layout（側欄導覽）
    ├── +page.svelte      ← 儀表板首頁
    ├── body-metrics/
    │   └── +page.svelte  ← 體位數據列表 + 新增
    ├── sleep/
    │   └── +page.svelte  ← 睡眠紀錄
    ├── activities/
    │   └── +page.svelte  ← 每日活動
    ├── magic/
    │   └── +page.svelte  ← 魔術練習
    └── maple/
        └── +page.svelte  ← MapleStory 快照
```

---

## 8. 啟動開發伺服器

```bash
cd frontend
pnpm dev
```

開啟 `http://localhost:5173`

---

## 9. 常用指令

| 指令 | 用途 |
|------|------|
| `pnpm dev` | 啟動開發伺服器（hot reload） |
| `pnpm build` | 打包正式版 |
| `pnpm preview` | 預覽打包結果 |
| `pnpm check` | TypeScript 型別檢查 |

---

## 10. 學習資源

- [Svelte 官方互動教學](https://learn.svelte.dev)（強烈推薦，30 分鐘內上手）
- [SvelteKit 官方文件](https://kit.svelte.dev/docs)
- [Svelte 5 Runes 說明](https://svelte.dev/docs/svelte/what-are-runes)
