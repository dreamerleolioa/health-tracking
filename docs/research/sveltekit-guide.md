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
│   ├── app.html          ← HTML 模板（整個 app 的殼）
│   ├── app.d.ts          ← TypeScript 全域型別宣告
│   ├── lib/              ← 共用元件、工具、API client、型別
│   │   ├── api/          ← API client 與各功能 API 封裝
│   │   ├── assets/       ← 圖示與靜態資產
│   │   ├── components/   ← 共用元件
│   │   ├── types/        ← 前端型別
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

| 檔案路徑                                    | 對應 URL            |
| ------------------------------------------- | ------------------- |
| `src/routes/+page.svelte`                   | `/`                 |
| `src/routes/body-metrics/+page.svelte`      | `/body-metrics`     |
| `src/routes/body-metrics/[id]/+page.svelte` | `/body-metrics/:id` |
| `src/routes/sleep/+page.svelte`             | `/sleep`            |

### 特殊檔名規則

| 檔名              | 用途                            |
| ----------------- | ------------------------------- |
| `+page.svelte`    | 頁面元件                        |
| `+layout.svelte`  | 包住子頁面的 layout             |
| `+page.ts`        | 頁面的資料載入（load function） |
| `+page.server.ts` | 僅在 server 執行的資料載入      |
| `+error.svelte`   | 錯誤頁面                        |

目前這個專案已經有：

- `src/routes/+layout.svelte`
- `src/routes/+page.svelte`
- `src/routes/activities/`
- `src/routes/body-metrics/`
- `src/routes/sleep/`

其中 `activities/`、`body-metrics/`、`sleep/` 目錄已經先建立，但頁面內容還在持續補齊。

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
  button {
    background: blue;
  }
</style>
```

補充：

- `<script lang="ts">` 放 TypeScript 邏輯
- HTML 區塊直接寫模板
- `<style>` 預設是 component-scoped
- Svelte 模板中的 `{}` 可以直接插入變數或表達式

### Svelte 5 Runes

| Rune                | 用途                   | 類比                   |
| ------------------- | ---------------------- | ---------------------- |
| `$state(value)`     | 響應式變數             | `ref()` in Vue         |
| `$derived(expr)`    | 計算值                 | `computed()` in Vue    |
| `$effect(() => {})` | 依賴變動時執行同步邏輯 | `watchEffect()` in Vue |
| `$props()`          | 接收父元件傳入的 props | `defineProps()` in Vue |

這個專案目前已經實際用到的 rune 是 `$props()`：

```svelte
<script lang="ts">
  let { children } = $props();
</script>
```

這段出現在 `+layout.svelte`，用來接收並渲染子頁面內容。

### 總結

學 Svelte 時，最重要的不是先背所有 API，而是先建立這個心智模型：

- `.svelte` 檔就是「邏輯 + 模板 + 樣式」的單位
- 畫面不是手動操作 DOM，而是讓資料去驅動模板
- Svelte 不是靠 runtime 的 Virtual DOM 比對，而是編譯期就把更新路徑準備好

如果用一句話記：

> 先把資料狀態想清楚，再讓模板自然長出來。

---

## 5. 模板控制語法

Svelte 最常用的模板控制語法有三種：

### 條件渲染

```svelte
{#if item.enabled}
  <a href={item.href}>{item.label}</a>
{:else}
  <span>{item.label}</span>
{/if}
```

這個專案的 `+layout.svelte` 就有使用 `{#if ...}{:else}{/if}` 來控制導覽項目是否可點擊。

### 清單渲染

```svelte
{#each metrics as metric}
  <div>{metric.label}</div>
{/each}
```

首頁 `+page.svelte` 目前就是用 `{#each}` 來渲染四張 mock 數據卡片。

### 內嵌表達式

```svelte
<p>{today}</p>
<div style="background-color: {metric.color};"></div>
```

這是 Svelte 最常見的資料綁定方式。

### 模板拆解順序

每次寫模板前，可以先問三件事：

1. 畫面是顯示單一值，還是顯示一個清單？
2. 某些區塊是否需要條件切換？
3. 樣式是固定的，還是會跟資料一起變化？

這三個問題通常就對應到：

- 單一值插值：`{value}`
- 條件切換：`{#if}`
- 清單渲染：`{#each}`
- 動態樣式：`class:` 或 `style="..."`

當這個拆法變自然，Svelte 的模板會很好寫。

---

## 6. 資料載入（load function）

在 `+page.ts` 定義 `load` function，SvelteKit 在渲染頁面前自動執行：

```typescript
// src/routes/body-metrics/+page.ts
import type { PageLoad } from "./$types";

export const load: PageLoad = async ({ fetch }) => {
  const res = await fetch("/v1/body-metrics");
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

這個模式很適合：

- 首次進頁面就要拿到資料
- 頁面資料需要 SSR / 預先載入
- 想把「抓資料」和「顯示資料」分開

目前專案中的 body metrics 頁面還沒正式補齊，這一節先作為後續頁面資料載入的參考模式。

### `load` 的思考方式

不要把 `load` 只當成「抓資料的地方」，更精確地說，它是：

- 進入頁面前要先準備好的資料
- 頁面 render 所依賴的初始狀態
- route 級別的資料邏輯

可以這樣區分：

- 如果資料屬於頁面初始化的一部分，用 `load`
- 如果資料屬於使用者互動後才觸發，用元件內函式或 API client

例如這個專案裡：

- 進 `/body-metrics` 時先拿最近 90 筆資料，很適合 `load`
- 按「新增」送出表單後再重抓資料，則比較像互動邏輯

### `+page.ts` 與 `+page.server.ts` 的差別

- `+page.ts` 可以跑在 server，也可以跑在 client 導頁過程
- `+page.server.ts` 只會跑在 server

簡單判斷方式：

- 需要安全地讀 server-only 資訊，用 `+page.server.ts`
- 只是做一般頁面資料準備，先從 `+page.ts` 開始

目前這個專案的 API 是獨立 Go backend，因此前端多半會使用 `+page.ts` 或元件內 API client 來呼叫後端。

---

## 7. 環境變數

SvelteKit 有兩種環境變數：

| 前綴      | 可在哪裡用                            | 範例                  |
| --------- | ------------------------------------- | --------------------- |
| `PUBLIC_` | 前端 + 後端都能用                     | `PUBLIC_API_BASE_URL` |
| 無前綴    | 只能在 server 端（`+page.server.ts`） | `DATABASE_URL`        |

使用方式：

```typescript
import { PUBLIC_API_BASE_URL } from "$env/static/public";

const res = await fetch(`${PUBLIC_API_BASE_URL}/body-metrics`);
```

這個專案目前在 `src/lib/api/client.ts` 中已經實際使用：

```typescript
import { PUBLIC_API_BASE_URL } from "$env/static/public";
```

重點：

- 只要需要在瀏覽器端讀取，就必須加上 `PUBLIC_` 前綴
- 不要把敏感資訊放在 `PUBLIC_` 變數裡

### 容易踩到的點

- `PUBLIC_API_BASE_URL` 可以被瀏覽器看到，所以不能放 secret
- 如果 API base URL 少了 `/v1`，前端請求路徑就會全部錯掉
- 環境變數改了之後，通常要重啟 dev server 才會生效

---

## 8. API client 寫法

這個專案沒有在每個頁面直接寫 `fetch(...)`，而是先集中到 `src/lib/api/client.ts`。

目前封裝了：

- `api.get`
- `api.post`
- `api.patch`
- `api.delete`
- `ApiException`

這種寫法的好處：

- base URL 只定義一次
- `Content-Type` header 可以統一設定
- 錯誤格式可以集中解析
- 後續新增 `body-metrics.ts`、`sleep-logs.ts` 等功能 API 會比較乾淨

學習重點不是「會不會 fetch」，而是「如何把 fetch 抽成專案可維護的 API 層」。

### 分層原則

頁面內如果同時出現下面這些內容，通常就表示責任開始混在一起：

- URL 字串
- `fetch` 呼叫
- `res.ok` 判斷
- `res.json()`
- 錯誤處理

短期雖然快，後面通常會變得難改、難讀、難重用。

比較穩的拆法可以分成三層：

1. `client.ts`：最低層 request 封裝
2. `body-metrics.ts` 這類功能檔：語意化 API
3. 頁面：使用資料與處理 UI 狀態

這樣整理之後，頁面會更接近產品邏輯，HTTP 細節則留在較底層處理。

---

## 9. 本專案的 src/ 規劃

```
src/
├── lib/
│   ├── api/              ← API client（呼叫後端的函式）
│   │   └── client.ts     ← fetch 封裝（base URL、error handling）
│   ├── components/       ← 共用 UI 元件
│   │   ├── charts/       ← 圖表元件
│   │   └── ui/           ← Button、Card 等基礎元件
│   └── types/            ← TypeScript 型別定義（對應後端 API schema）
│       └── index.ts
└── routes/
    ├── +layout.svelte    ← 全域 layout（主導覽與頁面容器）
    ├── +page.svelte      ← 儀表板首頁
    ├── body-metrics/
    │   └──               ← 體位數據功能預留目錄
    ├── sleep/
    │   └──               ← 睡眠功能預留目錄
    ├── activities/
    │   └──               ← 每日活動功能預留目錄
```

注意：這份規劃要以「目前 repo 真的存在的檔案」為準，不要再沿用舊草稿中 `magic/`、`maple/` 這類與本專案無關的內容。

### 檔案切分方式

如果之後要實作 `body-metrics`，可以優先這樣切：

- 頁面：`src/routes/body-metrics/+page.svelte`
- 頁面資料：`src/routes/body-metrics/+page.ts`
- API：`src/lib/api/body-metrics.ts`
- 型別：`src/lib/types/index.ts`
- 圖表元件：`src/lib/components/charts/...`

這樣切的原因是：

- route 資料準備放在 route 附近
- 可重用 API 放到 `lib`
- 可重用 UI 放到 `components`
- 型別集中管理，避免同一個 response 被重寫多次

---

## 10. 元件拆分與 props 設計

畫面開始成長後，就不能一直把所有東西都塞在同一個 `+page.svelte` 裡。

### 什麼時候該拆 component？

通常可以用這幾個條件判斷：

1. 同樣的 UI 結構出現超過 2 次
2. 某一段模板已經長到很難一眼看懂
3. 某個區塊未來很可能被別的頁面重用
4. 某個區塊有自己獨立的顯示邏輯

例如這個專案首頁的四張指標卡片，如果未來 dashboard、body metrics 頁、首頁摘要都會共用，就很適合拆成 `MetricCard.svelte`。

### 先做對，再抽象

畫面才剛有雛形就先抽出很多 component，通常只會讓重構成本變高，而不是變低。

整理順序：

- 先把頁面做對
- 等重複結構穩定出現再拆
- 拆出來後讓元件保持單一責任

### props 是什麼

props 就是父元件傳給子元件的資料。

在 Svelte 5 裡，可以用 `$props()` 接收：

```svelte
<script lang="ts">
  let { label, value, unit } = $props<{
    label: string;
    value: string;
    unit?: string;
  }>();
</script>

<div>
  <p>{label}</p>
  <strong>{value}</strong>
  {#if unit}
    <span>{unit}</span>
  {/if}
</div>
```

父元件就可以這樣使用：

```svelte
<MetricCard label="體重" value="72.5" unit="kg" />
```

### props 設計方式

設計 props 時，可以先問：

1. 這個元件真正需要哪些資料？
2. 哪些是必要欄位，哪些是選填？
3. 元件要不要接受樣式控制？
4. 這個元件只顯示資料，還是也要處理事件？

對這個健康追蹤專案來說，一張指標卡很可能需要：

- `label`
- `value`
- `unit`
- `color`
- `emoji`
- `prefix`

這些就很適合當 props。

### props 設計原則

- 子元件只拿它需要的資料
- 不要把整個巨大物件直接丟進去，除非它本來就是該元件的核心模型
- 型別要明確
- 優先傳資料，而不是傳一堆模糊旗標

例如這兩種寫法：

```svelte
<MetricCard metric={metric} />
```

```svelte
<MetricCard
  label={metric.label}
  value={metric.value}
  unit={metric.unit}
  color={metric.color}
  emoji={metric.emoji}
/>
```

沒有永遠正確的答案，但可以先用這個判斷方式：

- 如果 `metric` 是穩定的領域型別，可以直接傳物件
- 如果元件只依賴少數欄位，傳明確 props 會更清楚

### 資料往下，事件往上

component 拆出來後，除了資料往下傳，互動通常也要往上送。

例如未來如果卡片上有刪除按鈕，可以優先思考：

- 卡片自己呼叫 API
- 或卡片只負責告訴父層「使用者按了刪除」

通常會偏向第二種，因為：

- component 保持單純
- API 邏輯留在 page 或 feature 層
- 測試和重用會比較容易

### 套用到這個專案的做法

如果之後首頁 `metrics` 卡片要重構，可以優先拆成：

- `src/lib/components/ui/MetricCard.svelte`

然後頁面只負責：

- 準備資料
- 決定順序
- 用 `{#each}` 渲染元件

元件拆分原則可以先記成四句：

- 頁面負責整體流程
- component 負責可重用畫面片段
- props 負責把資料往下傳
- 事件負責把互動往上送

---

## 11. 讀懂目前首頁範例

目前首頁 `src/routes/+page.svelte` 很適合拿來理解 Svelte 基礎：

### 1. 一般 TypeScript 變數

```svelte
const today = new Date().toLocaleDateString('zh-TW', {
  year: 'numeric',
  month: 'long',
  day: 'numeric',
  weekday: 'long'
});
```

這種寫法適合不需要互動更新的靜態顯示資料。

### 2. 陣列驅動畫面

```svelte
const metrics = [
  { label: '體重', value: '72.5', unit: 'kg' }
];
```

再搭配 `{#each}` 生成 UI。這是 Svelte 很常見的資料驅動思維。

### 3. 依資料改變樣式

```svelte
<div class="h-1" style="background-color: {metric.color};"></div>
```

這表示樣式不一定只能靠 class，也可以由資料控制。

### 4. 先用 mock data，再替換成 API

首頁註解裡已經寫出「Mock 數值，之後接 API 替換」，這是很合理的前端開發順序：

1. 先把畫面做出來
2. 確認結構與樣式
3. 再接真實 API

### 從首頁可以學到的 Svelte 思維

首頁雖然現在資料是假的，但它已經示範一個很重要的前端流程：

- 先找出畫面的重複結構
- 把它抽成陣列資料
- 再用 `{#each}` 生成卡片

這個做法比一張一張卡片手寫更接近真正的產品開發方式。因為等 API 接上之後，只要把 `metrics` 的來源替換掉，模板通常可以不用大改。

---

## 12. Tailwind 在這個專案的使用方式

這個專案使用 Tailwind CSS v4，`app.css` 目前只有：

```css
@import "tailwindcss";
```

代表 Tailwind 是透過 Vite plugin 接入，樣式主要直接寫在 class 上，例如：

- `min-h-screen`
- `bg-[#1a1a2e]`
- `rounded-2xl`
- `font-black`
- `tracking-widest`

這種 utility-first 寫法會和模板緊密地寫在一起，先看懂版面、尺寸、顏色、字體、互動這幾類就夠用。

### Tailwind class 的閱讀方式

看到一長串 class 時，不要逐字背，先按功能拆：

- 版面：`grid`、`flex`、`gap-4`、`max-w-5xl`
- 尺寸：`h-48`、`px-6`、`py-8`
- 顏色：`bg-white/5`、`text-gray-400`
- 字體：`font-black`、`tracking-widest`、`text-2xl`
- 動畫互動：`transition-all`、`hover:-translate-y-1`

這樣讀 class 會比把整串當密碼記好很多。

---

## 13. 表單處理與雙向綁定

在健康追蹤專案裡，表單會是非常常見的操作，例如新增體重、體脂、睡眠紀錄。

Svelte 在表單上的優勢是語法很直接。

### 基本輸入綁定

```svelte
<script lang="ts">
  let weight = $state('72.5');
</script>

<input bind:value={weight} />
<p>目前輸入：{weight}</p>
```

`bind:value` 可以把 input 和狀態接起來。輸入改變時，畫面會同步更新。

### 數字輸入

```svelte
<script lang="ts">
  let bodyFat = $state<number | undefined>(undefined);
</script>

<input type="number" step="0.1" bind:value={bodyFat} />
```

這裡要注意：

- HTML input 本質上還是字串輸入
- 實際處理時要小心空值、`undefined`、`NaN`

### 送出表單

```svelte
<script lang="ts">
  async function handleSubmit() {
    // 驗證
    // 呼叫 API
    // 成功後更新 UI
  }
</script>

<form onsubmit|preventDefault={handleSubmit}>
  <button type="submit">送出</button>
</form>
```

整理重點：

- `preventDefault` 避免瀏覽器原生重整
- 把驗證、送出、錯誤顯示拆清楚
- 表單 state 不要和列表 state 混在一起

### 表單狀態通常至少有三種

- 輸入值
- 送出中狀態 `isSubmitting`
- 錯誤訊息 `errorMessage`

這三種狀態先想清楚，表單就不會寫得很亂。

---

## 14. 反應式思維：什麼該是 state，什麼該是 derived

這一節的重點是把「真正會變動的資料」和「可以推導出來的值」分開。

### 原始 state

真正需要被改變、被使用者操作的資料，才放進 state。

例如：

- 表單輸入值
- loading 狀態
- API 回來的原始列表

### derived state

如果某個值只是從其他資料算出來，就不要再存一份，直接 derived。

例如：

- 將 body metrics 過濾出最近 30 天
- 將同一天多筆紀錄 dedup
- 依據資料長度顯示「是否有資料」

概念上像這樣：

```svelte
<script lang="ts">
  let metrics = $state([]);
  let latestMetrics = $derived(metrics.slice(0, 30));
</script>
```

核心原則：

- 能推導出來的值，就不要重複存
- state 越少，bug 越少

---

## 15. 副作用與非同步操作

當資料改變後，需要做額外事情，才需要副作用。

例如：

- 某個條件成立時打 API
- 某個值改變時寫入 localStorage
- 元件載入後做瀏覽器限定邏輯

Svelte 5 可以用 `$effect()` 表達這種需求：

```svelte
<script lang="ts">
  let keyword = $state('');

  $effect(() => {
    console.log('keyword changed:', keyword);
  });
</script>
```

但要克制使用，因為很多時候其實只是「值推導」，那應該用 `$derived`，不是 `$effect`。

簡單判斷：

- 算值，用 `$derived`
- 做事，用 `$effect`

---

## 16. SSR、CSR 與瀏覽器限定邏輯

SvelteKit 不是只有 client-side app，它有 SSR 能力。所以你在寫程式時，要時常問自己：

> 這段程式是只有瀏覽器有，還是 server render 時也會跑到？

例如下面這種瀏覽器 API：

- `window`
- `document`
- `localStorage`
- `navigator`

如果直接在不對的位置使用，可能會遇到 `window is not defined`。

常見處理方式：

```typescript
import { browser } from "$app/environment";

if (browser) {
  // 只有瀏覽器端才執行
}
```

這在未來加入圖表套件或 PWA 邏輯時很重要。

---

## 17. 常見坑

### 1. 路由目錄存在，不代表頁面已完成

像這個專案的 `body-metrics/`、`sleep/`、`activities/` 目前只是預留目錄。看資料夾時，不要直接假設功能已經有了。

### 2. 前端環境變數改了卻沒重啟

這是最常見的小坑之一。改 `.env` 後請重啟 dev server。

### 3. 把太多責任塞進頁面元件

一個頁面如果同時負責：

- API 呼叫
- 表單驗證
- 錯誤處理
- 圖表資料轉換
- UI 呈現

很快就會變難維護。要學會拆到 `lib/api`、`lib/types`、`lib/components`。

### 4. 重複保存可以推導的值

這是很多前端 bug 的來源。例如「最新 30 筆資料」其實應該從原始資料推導，而不是再存一份獨立 state。

### 5. 太早做抽象

一開始先把頁面做對，比一開始就抽出十個 component 更重要。等重複模式真的出現，再抽共用元件。

---

## 18. 啟動開發伺服器

```bash
cd frontend
pnpm serve
```

開啟 `http://localhost:5173`

---

## 19. 常用指令

| 指令               | 用途                         |
| ------------------ | ---------------------------- |
| `pnpm serve`       | 啟動開發伺服器（hot reload） |
| `pnpm build`       | 打包正式版                   |
| `pnpm preview`     | 預覽打包結果                 |
| `pnpm check`       | TypeScript 型別檢查          |
| `pnpm check:watch` | 持續監看型別與 Svelte 檢查   |

---

## 20. 專案閱讀順序

閱讀順序：

1. 先看 `src/routes/+page.svelte`，理解模板、`{#each}`、`{#if}`、class 與 inline style
2. 再看 `src/routes/+layout.svelte`，理解 layout、`$props()` 與 `{@render children()}`
3. 接著看 `src/lib/api/client.ts`，理解前端如何組織 API 呼叫
4. 然後練習建立 `body-metrics/+page.ts` 與 `+page.svelte`，把 load function 與 UI 串起來
5. 最後再學圖表元件與表單互動

### 分段閱讀方式

如果要分幾次看，可以用這個節奏：

- 第 1 天：看首頁與 layout，先讀懂模板
- 第 2 天：自己手打一個小卡片元件，練 `{#each}` 和 props
- 第 3 天：練一個表單，熟悉 `bind:value` 和 submit
- 第 4 天：寫一個 `body-metrics.ts` API 函式
- 第 5 天：用 `+page.ts` 把資料載進頁面
- 第 6 天：練把資料轉成圖表可用格式

重點：每一步都能在專案裡找到對應場景，不要只停留在抽象概念。

---

## 21. 目前摘要

目前可以先記幾個重點：

- 它已經有基本 layout 和首頁 UI 雛形
- 它使用 Svelte 5 與 TypeScript
- API 呼叫已經開始集中封裝，而不是散落在頁面中
- body metrics 是最適合拿來做第一個完整 Svelte 練習的功能
- 後續最值得學的，是「如何把畫面、資料載入、表單與 API 拆得乾淨」

之後每做完一個功能，都可以把踩過的坑與最後的拆法補回來，這份文件會更接近真正有用的專案筆記。

---

## 22. 學習資源

- [Svelte 官方互動教學](https://learn.svelte.dev)（強烈推薦，30 分鐘內上手）
- [SvelteKit 官方文件](https://kit.svelte.dev/docs)
- [Svelte 5 Runes 說明](https://svelte.dev/docs/svelte/what-are-runes)
