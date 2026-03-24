# Milestone 1 執行計畫 — MVP：體位數據

> 目標：完成 `body_metrics` 的完整 CRUD 流程，從後端 API 到前端頁面與基礎折線圖。

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

| Query 名稱 | SQL 動作 | 對應 API |
|-----------|---------|---------|
| `CreateBodyMetric` | INSERT | POST `/v1/body-metrics` |
| `GetBodyMetric` | SELECT by id | （內部用） |
| `ListBodyMetrics` | SELECT with date range + limit | GET `/v1/body-metrics` |
| `UpdateBodyMetric` | UPDATE partial | PATCH `/v1/body-metrics/:id` |
| `DeleteBodyMetric` | DELETE by id | DELETE `/v1/body-metrics/:id` |

### 1-3 執行生成

```bash
cd backend
sqlc generate
```

產出 `db/sqlc/body_metrics.sql.go` 與 `db/sqlc/db.go`。

---

## Step 2 — 後端 API

### 2-1 型別定義 `internal/handler/body_metrics.go`

需定義 request / response struct：

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

**驗證規則：**
- `visceral_fat`：1–30
- `recorded_at`：必填，需為合法時間
- `from` / `to`：格式 `YYYY-MM-DD`，`to` 不得早於 `from`

**錯誤格式**（符合 SRS 規範）：
```json
{ "error": { "code": "VALIDATION_ERROR", "message": "...", "details": [...] } }
```

### 2-3 注冊路由（`cmd/api/main.go`）

```go
v1.POST("/body-metrics", handler.CreateBodyMetric(db))
v1.GET("/body-metrics", handler.ListBodyMetrics(db))
v1.PATCH("/body-metrics/:id", handler.UpdateBodyMetric(db))
v1.DELETE("/body-metrics/:id", handler.DeleteBodyMetric(db))
```

### 2-4 手動驗證

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

封裝 `fetch`，處理：
- Base URL（讀取 `PUBLIC_API_BASE_URL` 環境變數）
- `Content-Type: application/json`
- 統一錯誤格式解析，拋出 `ApiError`

### 3-2 `src/lib/api/body-metrics.ts`

```typescript
export type BodyMetric = {
  id: string
  weight_kg: number | null
  body_fat_pct: number | null
  muscle_pct: number | null
  visceral_fat: number | null
  recorded_at: string
  note: string | null
  created_at: string
}

export function createBodyMetric(data: CreateBodyMetricInput): Promise<BodyMetric>
export function listBodyMetrics(params: { from?: string; to?: string; limit?: number }): Promise<{ data: BodyMetric[]; meta: Meta }>
export function updateBodyMetric(id: string, data: Partial<CreateBodyMetricInput>): Promise<BodyMetric>
export function deleteBodyMetric(id: string): Promise<void>
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

### 4-2 新增表單

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
- [ ] 後端讀取正確的 `.env.local` / `.env.production`
- [ ] 前端 `PUBLIC_API_BASE_URL` 指向正確
- [ ] CORS 不報錯

---

## 完成標準

全部勾選才算 Milestone 1 完成：

- [ ] `POST /v1/body-metrics` 可新增資料並存入 DB
- [ ] `GET /v1/body-metrics` 支援 `from` / `to` 篩選並回傳正確格式
- [ ] `PATCH /v1/body-metrics/:id` 可 partial update
- [ ] `DELETE /v1/body-metrics/:id` 回傳 204
- [ ] 前端列表頁顯示資料
- [ ] 前端可新增一筆資料並即時反映
- [ ] 前端可刪除一筆資料
- [ ] 折線圖正確顯示體重 / 體脂率 / 肌肉率趨勢
- [ ] Local / Production 環境切換正常
