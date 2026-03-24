# Milestone 1 執行計畫 — MVP：體位數據

> 目標：完成 `body_metrics` 的完整 CRUD 流程，從後端 API 到前端頁面與基礎折線圖。
>
> **本 Milestone 不含：** 使用者認證（Auth）、分頁、多使用者支援。

---

## 任務總覽

```
Step 1  撰寫 sqlc query
Step 2  後端 API（Gin handler）
Step 3  前端 API client
Step 4  前端頁面（新增 / 列表）
Step 5  體重趨勢折線圖
Step 6  .env 環境切換驗證
```

依序執行，後步依賴前步。

---

## Step 1 — sqlc query

**目標：** 從 SQL 自動生成 type-safe 的 Go 程式碼，供 handler 使用。

### 1-1 建立 `sqlc.yaml`（後端根目錄）

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "db/queries/"
    schema: "db/migrations/"
    gen:
      go:
        package: "db"
        out: "db/sqlc"
        emit_json_tags: true
        emit_params_struct_pointers: true
        null_style: "omit_zero_value"
```

### 1-2 撰寫 `db/queries/body_metrics.sql`

需包含以下 query：

| Query 名稱 | SQL 動作 | 對應用途 |
|-----------|---------|---------|
| `CreateBodyMetric` | INSERT | POST `/v1/body-metrics` |
| `GetBodyMetric` | SELECT by id | Update/Delete handler 內部用（判斷 404） |
| `ListBodyMetrics` | SELECT with date range + limit，`ORDER BY recorded_at DESC` | GET `/v1/body-metrics` |
| `UpdateBodyMetric` | UPDATE partial（使用 COALESCE，null 表示不更新該欄位） | PATCH `/v1/body-metrics/:id` |
| `DeleteBodyMetric` | DELETE by id | DELETE `/v1/body-metrics/:id` |

> **注意：** `ListBodyMetrics` 需 `ORDER BY recorded_at DESC`，前端折線圖使用時需 dedup（同一天只取第一筆，即最新一筆）。

### 1-3 執行生成

```bash
cd backend
sqlc generate
```

產出 `db/sqlc/body_metrics.sql.go` 與 `db/sqlc/db.go`。

---

## Step 2 — 後端 API

### 2-1 型別定義 `internal/handler/body_metrics.go`

**Store interface（讓 handler 依賴抽象，方便測試 mock）：**

```go
type Store interface {
    CreateBodyMetric(ctx context.Context, params db.CreateBodyMetricParams) (db.BodyMetric, error)
    GetBodyMetric(ctx context.Context, id uuid.UUID) (db.BodyMetric, error)
    ListBodyMetrics(ctx context.Context, params db.ListBodyMetricsParams) ([]db.BodyMetric, error)
    UpdateBodyMetric(ctx context.Context, params db.UpdateBodyMetricParams) (db.BodyMetric, error)
    DeleteBodyMetric(ctx context.Context, id uuid.UUID) error
}
```

> sqlc 產生的 `*db.Queries` 會自動實作此 interface，handler 只依賴 `Store`，測試時注入 mock。

**Request / Response struct：**

```go
type CreateBodyMetricRequest struct {
    WeightKg    *float64  `json:"weight_kg"`
    BodyFatPct  *float64  `json:"body_fat_pct"`
    MusclePct   *float64  `json:"muscle_pct"`
    VisceralFat *int16    `json:"visceral_fat" validate:"omitempty,min=1,max=30"`
    RecordedAt  time.Time `json:"recorded_at"`
    Note        *string   `json:"note"`
}
```

### 2-2 實作 4 個 handler

| Handler | Method | Path |
|---------|--------|------|
| `CreateBodyMetric` | POST | `/v1/body-metrics` |
| `ListBodyMetrics` | GET | `/v1/body-metrics?from=&to=&limit=` |
| `UpdateBodyMetric` | PATCH | `/v1/body-metrics/:id` |
| `DeleteBodyMetric` | DELETE | `/v1/body-metrics/:id` |

**驗證套件：** 使用 `github.com/go-playground/validator/v10`（需 `go get` 加入 go.mod）

**驗證規則：**
- `weight_kg`：30–300（選填，有提供時驗證）
- `body_fat_pct`：1–70（選填，有提供時驗證）
- `muscle_pct`：10–80（選填，有提供時驗證）
- `visceral_fat`：1–30（選填，有提供時驗證）
- `recorded_at`：必填，需為合法時間
- `from` / `to`：格式 `YYYY-MM-DD`，`to` 不得早於 `from`；若都不傳，預設回傳全部資料
- `limit`：預設 100，最大不限（MVP 個人用量小）

**404 處理：**
Update/Delete handler 執行後若 sqlc 回傳 `pgx.ErrNoRows`（或 `sql.ErrNoRows`），回傳 404。不需獨立的 Get endpoint。

**回應格式：**
- POST → 201，body：`{ "data": { ...BodyMetric } }`
- GET → 200，body：`{ "data": [...], "meta": { "total": n, "from": "...", "to": "..." } }`
- PATCH → 200，body：`{ "data": { ...BodyMetric } }`
- DELETE → 204，無 body

**錯誤格式**（符合 SRS 規範）：
```json
{ "error": { "code": "VALIDATION_ERROR", "message": "...", "details": [...] } }
```

### 2-3 注冊路由（`cmd/api/main.go`）

```go
// queries 為 sqlc 產生的 *db.Queries，實作 handler.Store interface
queries := db.New(database)

v1.POST("/body-metrics", handler.CreateBodyMetric(queries))
v1.GET("/body-metrics", handler.ListBodyMetrics(queries))
v1.PATCH("/body-metrics/:id", handler.UpdateBodyMetric(queries))
v1.DELETE("/body-metrics/:id", handler.DeleteBodyMetric(queries))
```

### 2-5 Handler 測試 `internal/handler/body_metrics_test.go`

每個 handler 實作完成後立即補對應測試，使用 `net/http/httptest` + mock Store。

**Mock 結構：**

```go
type mockStore struct {
    createFn func(context.Context, db.CreateBodyMetricParams) (db.BodyMetric, error)
    getFn    func(context.Context, uuid.UUID) (db.BodyMetric, error)
    listFn   func(context.Context, db.ListBodyMetricsParams) ([]db.BodyMetric, error)
    updateFn func(context.Context, db.UpdateBodyMetricParams) (db.BodyMetric, error)
    deleteFn func(context.Context, uuid.UUID) error
}
// 實作 Store interface 的每個方法，呼叫對應的 fn
```

**各 handler 必須覆蓋的測試案例：**（使用 table-driven tests 寫法，見 SRS 4.3）

| Handler | 測試案例 |
|---------|---------|
| `CreateBodyMetric` | ✅ 正常新增（201）、❌ 缺少 `recorded_at`（400）、❌ `weight_kg` 超出範圍（400） |
| `ListBodyMetrics` | ✅ 無參數回傳列表、✅ 帶 `from`/`to` 篩選、❌ `to` 早於 `from`（400） |
| `UpdateBodyMetric` | ✅ 正常更新（200）、❌ id 不存在（404）、❌ 空 body → 400（無欄位可更新視為無效請求） |
| `DeleteBodyMetric` | ✅ 正常刪除（204）、❌ id 不存在（404） |

> **401 測試**：SRS 4.3 要求每個 endpoint 測試未授權情境，但 auth middleware 在 Milestone 3 實作。M1 的 handler mock 測試**不含 401 案例**，待 M3 加入 auth middleware 後補齊。

**測試工具：** 標準庫 `testing` + `net/http/httptest`，不需額外套件。

**執行測試：**
```bash
cd backend
go test ./...
```

**CI/CD：** `.github/workflows/test.yml` 目前設為手動觸發（`workflow_dispatch`）。上雲後再接入 deploy pipeline，屆時補上 push/PR trigger 或在 deploy job 加 `needs: test`。

### 2-5b 整合測試 `internal/repository/body_metrics_test.go`

對應 SRS 4.3 的「整合測試」層：使用 `testcontainers-go` 啟動真實 PostgreSQL 容器，驗證 sqlc query 與 DB 行為。

**安裝：**
```bash
cd backend
go get github.com/testcontainers/testcontainers-go
go get github.com/testcontainers/testcontainers-go/modules/postgres
```

**測試範圍：**

| 測試案例 | 目的 |
|---------|------|
| `CreateBodyMetric` → `GetBodyMetric` 確認寫入 | 基本 CRUD 正確性 |
| `ListBodyMetrics` with date range | 篩選邏輯與 ORDER BY |
| `UpdateBodyMetric` COALESCE 行為 | 只更新有值的欄位，null 欄位保持不變 |
| `DeleteBodyMetric` → `GetBodyMetric` 確認 ErrNoRows | 刪除後 404 觸發正確 |

**TestMain 結構：**
```go
func TestMain(m *testing.M) {
    ctx := context.Background()
    container, connStr := setupPostgres(ctx) // 啟動 testcontainer，執行 migration
    db = connectDB(connStr)
    code := m.Run()
    container.Terminate(ctx)
    os.Exit(code)
}
```

> GitHub Actions ubuntu-latest 有 Docker，CI 不需額外設定即可跑 testcontainers。

### 2-6 手動驗證

```bash
# 新增
curl -X POST http://localhost:8080/v1/body-metrics \
  -H "Content-Type: application/json" \
  -d '{"weight_kg":72.5,"body_fat_pct":18.2,"muscle_pct":35.2,"visceral_fat":8,"recorded_at":"2026-03-24T08:00:00+08:00"}'

# 列表
curl "http://localhost:8080/v1/body-metrics?from=2026-01-01&to=2026-03-31"
```

---

## Step 3 — 前端 API Client

**目標：** 封裝後端 API 呼叫，統一錯誤處理。

### 3-1 `src/lib/api/client.ts`

已存在，封裝 `fetch`，處理：
- Base URL（讀取 `PUBLIC_API_BASE_URL` 環境變數）
- `Content-Type: application/json`
- 統一錯誤格式解析，拋出 `ApiException`

### 3-2 新建 `src/lib/api/body-metrics.ts`

```typescript
import { api } from './client'
import type { BodyMetric, ListResponse, ItemResponse } from '$lib/types'

export type CreateBodyMetricInput = {
  weight_kg?: number
  body_fat_pct?: number
  muscle_pct?: number
  visceral_fat?: number
  recorded_at: string
  note?: string
}

export async function createBodyMetric(data: CreateBodyMetricInput): Promise<BodyMetric> {
  const res = await api.post<ItemResponse<BodyMetric>>('/body-metrics', data)
  return res.data
}

export async function listBodyMetrics(params?: {
  from?: string
  to?: string
  limit?: number
}): Promise<ListResponse<BodyMetric>> {
  const query = new URLSearchParams()
  if (params?.from) query.set('from', params.from)
  if (params?.to) query.set('to', params.to)
  if (params?.limit) query.set('limit', String(params.limit))
  const qs = query.toString() ? `?${query}` : ''
  return api.get<ListResponse<BodyMetric>>(`/body-metrics${qs}`)
}

export async function updateBodyMetric(
  id: string,
  data: Partial<CreateBodyMetricInput>
): Promise<BodyMetric> {
  const res = await api.patch<ItemResponse<BodyMetric>>(`/body-metrics/${id}`, data)
  return res.data
}

export async function deleteBodyMetric(id: string): Promise<void> {
  return api.delete(`/body-metrics/${id}`)
}
```

### 3-3 `frontend/.env`

```
PUBLIC_API_BASE_URL=http://localhost:8080/v1
```

---

## Step 4 — 前端頁面

### 4-1 列表頁 `src/routes/body-metrics/+page.svelte`

功能：
- 載入最近 90 筆資料（`+page.ts` load function）
- 以表格顯示：日期、體重、體脂率、肌肉率、內臟脂肪
- 每列有「刪除」按鈕
- 頁面頂部有「新增」按鈕，展開 inline form 或跳轉新增頁

### 4-2 `src/routes/body-metrics/+page.ts`

```typescript
import { listBodyMetrics } from '$lib/api/body-metrics'
import type { PageLoad } from './$types'

export const load: PageLoad = async () => {
  try {
    const res = await listBodyMetrics({ limit: 90 })
    return { metrics: res.data, meta: res.meta }
  } catch {
    return { metrics: [], meta: { total: 0 } }
  }
}
```

> 失敗時回傳空陣列，頁面顯示「無資料」狀態，不 throw error。
> 注意：load 回傳 `metrics`（非 `data`）避免與 SvelteKit `data` prop 命名衝突。

### 4-3 新增表單

欄位：
| 欄位 | 元件 | 備註 |
|------|------|------|
| 記錄時間 | `<input type="datetime-local">` | 預設當下時間 |
| 體重 (kg) | `<input type="number" step="0.1">` | |
| 體脂率 (%) | `<input type="number" step="0.1">` | |
| 肌肉率 (%) | `<input type="number" step="0.1">` | |
| 內臟脂肪 | `<input type="number">` | 1–30 |
| 備註 | `<textarea>` | 選填 |

送出後刷新列表。

---

## Step 5 — 體重趨勢折線圖

**套件選擇：** `layerchart`（SvelteKit 原生友善，基於 d3）

```bash
cd frontend
pnpm add layerchart
```

### 圖表規格

- X 軸：日期
- Y 軸：數值（共用軸或各自軸）
- 三條線：體重(kg)、體脂率(%)、肌肉率(%)
- Tooltip：hover 顯示當天數值
- 資料來源：列表頁載入的同一份資料，不額外打 API
- **Dedup 邏輯：** 同一天多筆資料，折線圖只取 `recorded_at` 最新的那筆（資料已 `ORDER BY recorded_at DESC`，取每天第一次出現的即可）。此邏輯實作於 chart component 內，作為 derived/computed 變數，不抽成獨立 utility。
- **SSR 注意：** layerchart 若有 `window is not defined` 問題，用 `{#if browser}` 包住 chart 元件（`import { browser } from '$app/environment'`）。

### 放置位置

列表頁上方，佔滿頁寬，高度約 300px。

---

## Step 6 — .env 環境切換驗證

確認兩個環境都能正常啟動：

**Local：**
```bash
cd backend && APP_ENV=local go run ./cmd/api
cd frontend && pnpm dev
```

**Production 模擬：**
```bash
cd backend && APP_ENV=production go run ./cmd/api
cd frontend && pnpm build && pnpm preview
```

檢查項目：
- [ ] 後端讀取正確的 `.env.local` / `.env.production`（key 為 `DATABASE_URL`，非 `DB_DSN`）
- [ ] 前端 `PUBLIC_API_BASE_URL` 指向正確
- [ ] CORS 不報錯

---

## 完成標準

全部勾選才算 Milestone 1 完成：

**後端 API（每項須同時通過測試才算完成）**
- [x] `POST /v1/body-metrics` 可新增資料並存入 DB
- [x] `GET /v1/body-metrics` 支援 `from` / `to` 篩選並回傳正確格式
- [x] `PATCH /v1/body-metrics/:id` 可 partial update
- [x] `DELETE /v1/body-metrics/:id` 回傳 204
- [x] Update/Delete 找不到 id 時回傳 404
- [x] `go test ./...` 全數通過（含 Step 2-5 handler mock 測試、Step 2-5b 整合測試）

**前端**（程式碼已完成，待手動驗證後勾選）
- [ ] 前端列表頁顯示資料
- [ ] 前端可新增一筆資料並即時反映
- [ ] 前端可刪除一筆資料
- [ ] 折線圖正確顯示體重 / 體脂率 / 肌肉率趨勢（同天取最新一筆）

**環境**（設定已驗證正確，待實際啟動確認）
- [ ] Local / Production 環境切換正常
