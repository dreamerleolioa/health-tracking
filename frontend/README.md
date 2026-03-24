# Frontend

這個目錄包含健康追蹤產品的前端應用，使用 SvelteKit、TypeScript、Tailwind CSS 與 Vite 建構，負責呈現儀表板、資料輸入頁面與後續登入後的使用者操作流程。

## 技術選型

- SvelteKit
- Svelte 5
- TypeScript
- Tailwind CSS v4
- Vite

## 開發指令

先安裝依賴：

```bash
pnpm install
```

啟動本機開發伺服器：

```bash
pnpm serve
```

型別與 Svelte 檢查：

```bash
pnpm check
```

建置正式版：

```bash
pnpm build
```

預覽正式版輸出：

```bash
pnpm preview
```

## 環境變數

前端 API client 會讀取公開環境變數 `PUBLIC_API_BASE_URL` 作為後端 API 的 base URL。

範例：

```env
PUBLIC_API_BASE_URL=http://localhost:8080/v1
```

如果未正確設定，前端呼叫 API 時會指向錯誤位置。

## 目前目錄結構

```text
src/
	lib/
		api/
			client.ts        API 請求封裝
		assets/            靜態資產
		components/        可重用元件
		types/             前端型別定義
	routes/
		+layout.svelte     全站版型與主導覽
		+page.svelte       首頁 / 儀表板入口
		activities/        每日活動頁面
		body-metrics/      體位數據頁面
		sleep/             睡眠頁面
```

## 路由規劃

- `/` 首頁儀表板
- `/body-metrics` 體位數據功能頁
- `/sleep` 睡眠功能頁
- `/activities` 每日活動功能頁

目前主導覽已經有基礎結構，其中體位數據頁可導覽，其餘頁面保留為後續功能擴充入口。

## API 呼叫方式

統一透過 [src/lib/api/client.ts](src/lib/api/client.ts) 進行請求，封裝了：

- `get`
- `post`
- `patch`
- `delete`
- `ApiException`

錯誤格式預期為後端統一的 `error` 物件，因此新增 API 時應沿用相同回傳格式，避免前端錯誤處理邏輯分岔。

## 目前 UI 狀態

- [src/routes/+layout.svelte](src/routes/+layout.svelte) 提供紅黑主視覺導覽列與頁面容器
- [src/routes/+page.svelte](src/routes/+page.svelte) 目前為首頁儀表板雛形
- 首頁數據卡片仍使用 mock data
- 趨勢圖區目前為 placeholder，待串接真實資料與圖表元件

## 開發約定

- 使用 TypeScript
- 以 SvelteKit routes 作為頁面切分主軸
- API 邏輯集中在 `src/lib/api`
- 共用型別集中在 `src/lib/types`
- 共用元件集中在 `src/lib/components`
- 新功能優先沿用現有版型與視覺語言

## 下一步開發重點

- 將首頁 mock data 改為串接 body metrics API
- 補齊 `body-metrics` 頁面的新增、列表與編輯流程
- 導入趨勢圖元件
- 補齊 `sleep` 與 `activities` 頁面
- 加入登入狀態與 route guard
- 加入 PWA 與離線提示
