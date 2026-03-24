# Go 後端學習指南

> 針對本專案的 Go 後端入門，說明專案結構、核心套件與開發流程。

---

## 1. 技術棧一覽

| 角色       | 套件                          | 說明                                  |
| ---------- | ----------------------------- | ------------------------------------- |
| HTTP 框架  | `gin-gonic/gin`               | 路由、middleware、JSON binding        |
| 資料庫驅動 | `jackc/pgx/v5`                | PostgreSQL 驅動（透過 `stdlib` 介面） |
| Migration  | `golang-migrate/migrate/v4`   | 管理 DB schema 版本                   |
| Query 生成 | `sqlc`                        | 從 SQL 自動生成 type-safe Go 程式碼   |
| 輸入驗證   | `go-playground/validator/v10` | struct tag 驗證                       |
| 環境變數   | `joho/godotenv`               | 載入 `.env` 檔                        |

---

## 2. 專案結構

```
backend/
├── cmd/
│   └── api/
│       └── main.go           ← 程式進入點
├── db/
│   ├── migrations/           ← SQL migration 檔（up / down 成對）
│   ├── queries/              ← sqlc 用的 SQL query 檔
│   └── sqlc/                 ← sqlc 生成的 Go 程式碼
├── internal/
│   ├── config/
│   │   └── config.go         ← 讀取環境變數、組成 Config struct
│   ├── db/
│   │   ├── db.go             ← 建立 DB 連線、設定連線池
│   │   └── migrate.go        ← 執行 migration
│   ├── handler/
│   │   ├── health.go         ← 健康檢查 handler
│   │   ├── body_metrics.go   ← 體位數據 CRUD handler
│   │   └── body_metrics_test.go ← body metrics handler 測試
│   └── middleware/
│       └── cors.go           ← CORS middleware
├── .env.local                ← 本地開發環境變數（不進 git）
├── .env.example              ← 環境變數範本（進 git）
├── go.mod                    ← 模組宣告與依賴清單
└── go.sum                    ← 依賴 checksum（自動維護）
```

---

## 3. 程式進入點 `main.go`

啟動流程：

```
載入 Config → 連接 DB → 執行 Migration → 建立 Gin router → 啟動 HTTP server
```

```go
cfg := config.Load()           // 1. 讀取環境變數
database, _ := db.New(cfg.DatabaseURL)  // 2. 連接 DB
db.RunMigrations(database, "db/migrations") // 3. 自動跑 migration
r := gin.Default()             // 4. 建立 router
r.Use(middleware.CORS(...))    // 5. 掛 middleware
v1 := r.Group("/v1")           // 6. 設定路由群組
http.ListenAndServe(":8080", r) // 7. 啟動 server
```

### 總結

`main.go` 可以視為後端的組裝點，而不是放業務邏輯的地方。

也就是說，`main.go` 主要負責：

- 把設定讀進來
- 把資料庫接起來
- 把 middleware、queries、handlers 組起來
- 啟動整個應用程式

如果某段程式開始出現很多驗證、轉換或商業規則，通常就不應該繼續塞在 `main.go`。

---

## 4. 環境變數

透過 `godotenv` 依 `APP_ENV` 載入對應的 `.env` 檔：

| `APP_ENV`       | 載入的檔案        |
| --------------- | ----------------- |
| `local`（預設） | `.env.local`      |
| `production`    | `.env.production` |

```bash
# .env.local
APP_ENV=local
SERVER_PORT=8080
DATABASE_URL=postgresql://myuser:mypassword@localhost:5432/health_tracking?sslmode=disable
CORS_ORIGINS=http://localhost:5173
```

在程式中透過 `config.Load()` 取得，不要直接呼叫 `os.Getenv()`。

---

## 5. DB 連線與連線池

`internal/db/db.go` 負責建立連線，內建連線池設定：

```go
db.SetMaxOpenConns(10)              // 最多 10 條連線
db.SetMaxIdleConns(5)               // 閒置最多保留 5 條
db.SetConnMaxLifetime(5 * time.Minute) // 連線最長存活 5 分鐘
```

個人專案規模下這個設定已足夠，不需調整。

---

## 6. Migration

使用 `golang-migrate`，採用**檔案命名版本**管理 schema 變更：

```
db/migrations/
├── 001_init.up.sql       ← 建立所有初始資料表
├── 001_init.down.sql     ← 刪除所有資料表（rollback）
├── 002_rename_muscle_kg_to_pct.up.sql
└── 002_rename_muscle_kg_to_pct.down.sql
```

### 規則

- 每次 schema 變更新增一對 `NNN_描述.up.sql` / `NNN_描述.down.sql`
- **不要修改已執行過的 migration**，一律新增新檔
- `up.sql` 是正向變更，`down.sql` 是還原

### Migration 在 `main.go` 自動執行

伺服器啟動時會自動套用尚未執行的 migration，不需手動跑指令。

### 手動執行（需要時）

```bash
migrate -path ./db/migrations \
  -database "postgresql://myuser:mypassword@localhost:5432/health_tracking?sslmode=disable" \
  up       # 套用所有待執行的 migration
  down 1   # 回滾最近一筆
  version  # 查看目前版本
```

---

## 7. 目前已實作的 API Handler

目前後端已經不只有健康檢查，`body_metrics` 的 CRUD handler 也已完成。

已註冊路由：

```go
v1 := r.Group("/v1")
{
        v1.GET("/health", handler.HealthCheck)
        v1.POST("/body-metrics", handler.CreateBodyMetric(queries))
        v1.GET("/body-metrics", handler.ListBodyMetrics(queries))
        v1.PATCH("/body-metrics/:id", handler.UpdateBodyMetric(queries))
        v1.DELETE("/body-metrics/:id", handler.DeleteBodyMetric(queries))
}
```

這些 handler 集中在 `internal/handler/body_metrics.go`，目前做了幾件事：

- 解析 JSON request body
- 驗證欄位範圍與必要欄位
- 解析 `from` / `to` / `limit` query param
- 解析 `:id` UUID 路徑參數
- 呼叫 sqlc 產生的查詢方法
- 將 `sql.NullString` / `sql.NullInt16` 轉成 API response
- 統一錯誤格式

---

## 8. sqlc

sqlc 已經導入到專案中，後端目前的 `body_metrics` CRUD 就是透過 sqlc 生成的 type-safe 查詢在運作。

目前設定檔是 `backend/sqlc.yaml`，重點如下：

```yaml
version: "2"
sql:
    - engine: "postgresql"
        queries: "db/queries/"
        schema: "db/schema.sql"
        gen:
            go:
                package: "db"
                out: "db/sqlc"
                emit_json_tags: true
                emit_params_struct_pointers: true
```

目前流程：

1. 在 `db/queries/` 撰寫 SQL query 並加上 sqlc 註解
2. 執行 `sqlc generate`
3. `db/sqlc/` 產生對應的 Go 型別與方法
4. 在 handler 中透過 `queries := db.New(database)` 呼叫生成方法

目前 `db/queries/body_metrics.sql` 已定義：

- `CreateBodyMetric`
- `GetBodyMetric`
- `ListBodyMetrics`
- `UpdateBodyMetric`
- `DeleteBodyMetric`

### 重點理解

sqlc 最有價值的地方，不只是「少寫一點樣板」，而是它迫使資料存取邏輯清楚地寫回 SQL。

這樣的好處是：

- SQL 還是 SQL，不會被過度抽象藏起來
- Go 端拿到的是型別安全的方法
- query 集中在 `db/queries/`，比較容易檢查與維護

對自學來說，這很有幫助，因為你會同時練到：

- schema 設計
- SQL 查詢
- Go 型別與資料轉換

---

## 9. 啟動開發伺服器

```bash
cd backend
APP_ENV=local go run ./cmd/api
```

伺服器啟動於 `http://localhost:8080`，可用以下指令確認：

```bash
curl http://localhost:8080/v1/health
# {"status":"ok"}
```

也可以直接測試目前已完成的 body metrics API：

```bash
curl -X POST http://localhost:8080/v1/body-metrics \
    -H "Content-Type: application/json" \
    -d '{"weight_kg":72.5,"body_fat_pct":18.2,"muscle_pct":35.2,"visceral_fat":8,"recorded_at":"2026-03-24T08:00:00+08:00"}'

curl "http://localhost:8080/v1/body-metrics?from=2026-01-01&to=2026-03-31&limit=90"
```

---

## 10. 測試

目前已經有 `body_metrics` handler 的 table-driven tests，位置在：

- `internal/handler/body_metrics_test.go`

測試覆蓋的情境包含：

- `CreateBodyMetric` 成功與驗證失敗
- `ListBodyMetrics` 日期範圍查詢與錯誤輸入
- `UpdateBodyMetric` 成功、空 payload、資料不存在
- `DeleteBodyMetric` 成功與資料不存在

執行方式：

```bash
cd backend
go test ./...
```

這份測試目前以 mock store 為主，適合先驗證 handler 的 request/response 與錯誤處理邏輯。

### 測試重點

這些測試的重點不是證明資料庫真的有寫入，而是驗證 handler 這層的邊界行為，例如：

- 驗證錯誤有沒有回 400
- 找不到資料有沒有回 404
- 成功時 response 格式對不對

所以它比較像 handler 層的單元測試，而不是完整整合測試。這個分層觀念很重要。

---

## 11. Store interface 與 handler 設計

`body_metrics` handler 沒有直接依賴具體資料庫實作，而是依賴 `Store interface`：

```go
type Store interface {
        CreateBodyMetric(ctx context.Context, arg *sqlcdb.CreateBodyMetricParams) (sqlcdb.BodyMetric, error)
        GetBodyMetric(ctx context.Context, id uuid.UUID) (sqlcdb.BodyMetric, error)
        ListBodyMetrics(ctx context.Context, arg *sqlcdb.ListBodyMetricsParams) ([]sqlcdb.BodyMetric, error)
        UpdateBodyMetric(ctx context.Context, arg *sqlcdb.UpdateBodyMetricParams) (sqlcdb.BodyMetric, error)
        DeleteBodyMetric(ctx context.Context, id uuid.UUID) error
}
```

這樣設計的好處：

- handler 與資料存取層解耦
- 單元測試可以注入 mock store
- 未來若要加入 service / repository 分層，比較容易擴充

### 檔案拆分與介面設計

這一節可以把它當成後端版本的「元件拆分與 props 設計」。

前端會拆 component，後端常見的對應則是拆：

- handler
- service
- repository / queries
- interface

### 什麼時候該拆？

下面幾種情況通常就該考慮拆分：

1. 某個 handler 同時做驗證、商業規則、資料轉換、資料庫操作
2. 同一份邏輯被多個 endpoint 重複使用
3. 測試很難寫，因為依賴太多具體實作
4. 程式碼已經開始像一大段流程，沒有清楚邊界

### interface 的角色是什麼

`Store interface` 的重點不是為了「看起來有架構」，而是把 handler 依賴的能力講清楚。

對目前的 `body_metrics` handler 來說，它真正需要的能力就是：

- 建立 body metric
- 取得 body metric
- 列出 body metrics
- 更新 body metric
- 刪除 body metric

所以 interface 只定義這些方法，而不是把整個資料庫物件直接塞進 handler。

### interface 大小的判斷方式

原則是：

- 只放呼叫者真的需要的方法
- 不要把所有未來可能會用到的方法先塞進去
- interface 應該貼近使用者需求，而不是貼近底層實作細節

### 拆分原則

可以先記這四句：

- handler 處理 HTTP
- service 處理業務規則
- repository / sqlc 處理資料存取
- interface 描述依賴邊界

目前這個專案還沒有完整 service 層，但 `Store interface` 已經是往這個方向整理的第一步。

---

## 12. 開發前的思考順序

如果今天要新增一個 endpoint，通常可以照這個順序思考：

1. request / response 長什麼樣子？
2. 驗證規則是什麼？
3. 需要哪些資料庫操作？
4. 這些操作能不能用一個清楚的 interface 表示？
5. 哪些案例要測？

用這個順序做，通常比較不會變成先亂寫 handler，後面再補救。

---

## 13. 常見坑

### 1. 把所有邏輯都寫進 handler

短期看起來很快，但很快就會變難測、難讀、難改。

### 2. 看到 sqlc 就以為不用理解 SQL

剛好相反。用了 sqlc，更應該把 SQL 寫清楚，因為它就是資料存取層的核心。

### 3. 用 interface 只是為了抽象而抽象

如果 interface 只是把整個實作原封不動包起來，價值不大。真正有價值的是它能描述依賴邊界，並讓測試更容易。

### 4. 沒有分清單元測試和整合測試

mock store 的測試主要驗證 handler 行為，不是驗證資料庫。這兩種測試目的不同。

### 5. 在 `main.go` 放太多東西

`main.go` 應該像組裝工廠，不應該變成最大的業務檔案。

---

## 14. 常用指令

| 指令                 | 用途                               |
| -------------------- | ---------------------------------- |
| `go run ./cmd/api`   | 啟動伺服器                         |
| `go build ./cmd/api` | 編譯 binary                        |
| `go mod tidy`        | 整理依賴（移除未用的、補上缺少的） |
| `go vet ./...`       | 靜態分析                           |
| `go test ./...`      | 執行所有測試                       |
| `sqlc generate`      | 從 SQL 生成 Go 程式碼              |

---

## 15. 專案閱讀順序

閱讀順序：

1. 先讀 `cmd/api/main.go`，理解啟動流程
2. 再讀 `internal/config/config.go` 與 `internal/db/db.go`，理解設定與 DB 連線
3. 接著讀 `internal/handler/body_metrics.go`，理解 handler 邊界
4. 再讀 `db/queries/body_metrics.sql` 與 `db/sqlc/` 生成結果，理解 sqlc 怎麼把 SQL 接進 Go
5. 最後看 `body_metrics_test.go`，理解為什麼要用 interface + mock

重點：先看整體流程，再看每層責任，不要一開始就鑽進單一函式細節。

---

## 16. 目前尚未完成的後端項目

雖然 Milestone 1 的 `body_metrics` 基礎後端已落地，但整體後端還有幾塊尚未完成：

- Google OAuth 2.0 登入
- JWT / Refresh Token
- `user_id` 多使用者資料隔離
- `sleep_logs` CRUD
- `daily_activities` CRUD
- 認證 middleware
- rate limiting
- repository 整合測試

因此目前這份後端可以視為「MVP 的 body metrics API 已就位」，但還不是完整的最終版本。

---

## 17. 學習資源

- [Go 官方 Tour](https://go.dev/tour/)（語法入門）
- [Gin 官方文件](https://gin-gonic.com/docs/)
- [sqlc 官方文件](https://docs.sqlc.dev/)
- [golang-migrate 說明](https://github.com/golang-migrate/migrate)
