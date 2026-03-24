# Backend

這個目錄包含健康追蹤專案的後端 API、資料庫 migration、sqlc 查詢定義與 HTTP handler。後端使用 Go 與 Gin 建構，資料庫為 PostgreSQL。

## 技術組成

- Go 1.26+
- Gin
- PostgreSQL
- sqlc
- golang-migrate
- godotenv
- go-playground/validator
- testcontainers-go

## 目錄結構

```text
cmd/
  api/
    main.go                API 啟動入口

db/
  migrations/             資料庫 migration
  queries/                sqlc 查詢定義
  sqlc/                   sqlc 生成程式碼
  schema.sql              資料表 schema

internal/
  config/                 環境變數載入
  db/                     DB 連線與 migration 執行
  handler/                HTTP handlers 與測試
  middleware/             Gin middleware
  repository/             預留資料存取層擴充
```

## 本機開發需求

- Go 1.26 或以上
- PostgreSQL 15 或以上
- 可用的 `health_tracking` 資料庫

## 環境變數

後端啟動時會先讀取 `APP_ENV`，接著載入對應的 `.env.<env>`。若找不到，會回退到 `.env`。

可參考 [backend/.env.example](backend/.env.example)。

範例：

```env
APP_ENV=local
SERVER_PORT=8080
DATABASE_URL=postgresql://user:password@localhost:5432/health_tracking?sslmode=disable
CORS_ORIGINS=http://localhost:5173
```

## 啟動方式

在 backend 目錄下執行：

```bash
go run ./cmd/api
```

啟動流程包含：

1. 載入環境變數
2. 建立 PostgreSQL 連線
3. 自動執行 `db/migrations` 下的 migration
4. 初始化 sqlc queries
5. 啟動 Gin HTTP server

預設會監聽：

```text
:8080
```

## 目前已實作 API

Base path:

```text
/v1
```

### Health

- `GET /health`

### Body Metrics

- `POST /body-metrics`
- `GET /body-metrics`
- `PATCH /body-metrics/:id`
- `DELETE /body-metrics/:id`

目前 `body_metrics` 支援：

- 建立紀錄
- 日期區間查詢 `from` / `to`
- `limit` 分頁上限參數
- 部分欄位更新
- 刪除前先檢查資料是否存在
- 統一錯誤回傳格式

## Handler 行為說明

目前主要 handler 位於 [backend/internal/handler/body_metrics.go](backend/internal/handler/body_metrics.go)。

已實作內容：

- Gin request binding
- `validator` 輸入驗證
- `uuid` 路徑參數解析
- `sql.NullString` / `sql.NullInt16` 與 API response 轉換
- 統一 `error` JSON 格式

目前 body metrics 驗證規則包含：

- `weight_kg`: 30 到 300
- `body_fat_pct`: 1 到 70
- `muscle_pct`: 10 到 80
- `visceral_fat`: 1 到 30
- `recorded_at` 為必填

## 資料存取層

sqlc 設定檔位於 [backend/sqlc.yaml](backend/sqlc.yaml)，會根據：

- [backend/db/schema.sql](backend/db/schema.sql)
- [backend/db/queries/body_metrics.sql](backend/db/queries/body_metrics.sql)

生成 Go 型別與 query methods 到：

- [backend/db/sqlc](backend/db/sqlc)

目前已定義的 body metrics queries：

- `CreateBodyMetric`
- `GetBodyMetric`
- `ListBodyMetrics`
- `UpdateBodyMetric`
- `DeleteBodyMetric`

## Migration

Migration 檔案位於：

- [backend/db/migrations](backend/db/migrations)

目前 migration 流程由啟動程式自動執行，不需要另外手動跑 migrate 指令。

現有 migration 已涵蓋：

- 初始 schema 建立
- `muscle_kg` 欄位更名為 `muscle_pct`
- 非健康相關資料表清理

## 連線與效能設定

資料庫初始化位於 [backend/internal/db/db.go](backend/internal/db/db.go)。

目前設定：

- `MaxOpenConns = 10`
- `MaxIdleConns = 5`
- `ConnMaxLifetime = 5 分鐘`
- 啟動時使用 5 秒 timeout 做 `PingContext`

這些設定與 SRS 中的非功能性要求一致。

## CORS

Gin middleware 會依 `CORS_ORIGINS` 套用跨來源設定，供前端本機開發使用。相關實作位於：

- [backend/internal/middleware/cors.go](backend/internal/middleware/cors.go)

## 測試

目前已有 body metrics handler 測試：

- [backend/internal/handler/body_metrics_test.go](backend/internal/handler/body_metrics_test.go)

執行全部測試：

```bash
go test ./...
```

目前測試重點包含：

- 建立成功與驗證失敗
- 查詢成功與日期範圍錯誤
- 更新成功、空 payload、資料不存在
- 刪除成功與資料不存在

## 與需求文件的落差

以下功能已在規劃中，但目前尚未在 backend 落地：

- Google OAuth 2.0 登入流程
- JWT / Refresh Token
- `user_id` 資料隔離
- `sleep_logs` CRUD
- `daily_activities` CRUD
- 認證 middleware
- Rate limiting
- 覆蓋率門檻與 CI 整合

## 下一步建議

- 補上 `sleep_logs` 與 `daily_activities` schema / queries / handlers
- 將 `body_metrics` 表與查詢補上 `user_id`
- 加入 auth middleware 與 `/v1/auth/*` 路由
- 建立 repository / service 分層，降低 handler 與 sqlc 耦合
- 增加 integration tests 驗證 PostgreSQL trigger 與 migration 行為
