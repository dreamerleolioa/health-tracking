# Go 後端學習指南

> 針對本專案的 Go 後端入門，說明專案結構、核心套件與開發流程。

---

## 1. 技術棧一覽

| 角色         | 套件                          | 說明                                     |
| ------------ | ----------------------------- | ---------------------------------------- |
| HTTP 框架    | `gin-gonic/gin`               | 路由、middleware、JSON binding           |
| 資料庫驅動   | `jackc/pgx/v5`                | PostgreSQL 驅動（透過 `stdlib` 介面）    |
| Migration    | `golang-migrate/migrate/v4`   | 管理 DB schema 版本                      |
| Query 生成   | `sqlc`                        | 從 SQL 自動生成 type-safe Go 程式碼      |
| 輸入驗證     | `go-playground/validator/v10` | struct tag 驗證                          |
| 環境變數     | `joho/godotenv`               | 載入 `.env` 檔                           |
| JWT          | `golang-jwt/jwt/v5`           | 簽發與驗證 Access Token / Refresh Token  |
| OAuth        | `golang.org/x/oauth2`         | Google OAuth 2.0 flow                    |
| UUID         | `google/uuid`                 | 生成與解析 UUID                          |
| 測試容器     | `testcontainers-go`           | 整合測試用真實 PostgreSQL 容器           |
| Rate Limit   | `ulule/limiter`               | 限流 middleware                          |

---

## 2. 專案結構

```
backend/
├── cmd/
│   └── api/
│       └── main.go                 ← 程式進入點
├── db/
│   ├── migrations/                 ← SQL migration 檔（up / down 成對）
│   ├── queries/                    ← sqlc 用的 SQL query 檔
│   └── sqlc/                       ← sqlc 生成的 Go 程式碼
├── internal/
│   ├── auth/
│   │   └── jwt.go                  ← JWT 簽發、驗證、Refresh Token 邏輯
│   ├── config/
│   │   └── config.go               ← 讀取環境變數、組成 Config struct
│   ├── db/
│   │   ├── db.go                   ← 建立 DB 連線、設定連線池
│   │   └── migrate.go              ← 執行 migration
│   ├── handler/
│   │   ├── health.go               ← 健康檢查
│   │   ├── auth.go                 ← Google OAuth callback、refresh、logout、me
│   │   ├── auth_test.go
│   │   ├── body_metrics.go         ← 體位數據 CRUD
│   │   ├── body_metrics_test.go
│   │   ├── sleep_logs.go           ← 睡眠紀錄 CRUD
│   │   ├── sleep_logs_test.go
│   │   ├── daily_activities.go     ← 每日活動 CRUD
│   │   └── daily_activities_test.go
│   ├── middleware/
│   │   ├── auth.go                 ← JWT 驗證 middleware，注入 user_id 到 context
│   │   ├── auth_test.go
│   │   ├── cors.go                 ← CORS middleware
│   │   └── rate_limit.go           ← 限流 middleware
│   └── repository/
│       ├── auth.go                 ← users / refresh_tokens 資料存取
│       ├── auth_test.go
│       ├── body_metrics.go         ← body_metrics repository
│       ├── body_metrics_test.go
│       ├── sleep_logs.go
│       ├── sleep_logs_test.go
│       ├── daily_activities.go
│       ├── daily_activities_test.go
│       └── isolation_test.go       ← 跨 user_id 資料隔離測試
├── .env.local                      ← 本地環境變數（不進 git）
├── .env.example                    ← 環境變數範本（進 git）
├── go.mod
└── go.sum
```

---

## 3. 程式進入點 `main.go`

啟動流程：

```
載入 Config → 連接 DB → 執行 Migration → 建立 Gin router → 掛 middleware → 註冊路由 → 啟動 HTTP server
```

`main.go` 是組裝點，不放業務邏輯。所有設定、middleware、handler 都在這裡接起來。

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
JWT_SECRET=your-secret-here
GOOGLE_CLIENT_ID=...
GOOGLE_CLIENT_SECRET=...
GOOGLE_REDIRECT_URL=http://localhost:8080/v1/auth/google/callback
```

在程式中透過 `config.Load()` 取得，不要直接呼叫 `os.Getenv()`。

---

## 5. DB 連線與連線池

`internal/db/db.go` 負責建立連線，內建連線池設定：

```go
db.SetMaxOpenConns(10)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

---

## 6. Migration

使用 `golang-migrate`，採用**檔案命名版本**管理 schema 變更：

```
db/migrations/
├── 001_init.up.sql         ← 建立初始資料表
├── 001_init.down.sql
├── 002_add_users.up.sql    ← 加入 users、refresh_tokens 資料表
├── 002_add_users.down.sql
└── ...
```

### 規則

- 每次 schema 變更新增一對 `NNN_描述.up.sql` / `NNN_描述.down.sql`
- **不要修改已執行過的 migration**，一律新增新檔
- 伺服器啟動時自動套用尚未執行的 migration

---

## 7. 目前已實作的 API

### 認證 `/v1/auth`

```go
v1.GET("/auth/google", handler.RedirectToGoogle)
v1.GET("/auth/google/callback", handler.GoogleCallback)
v1.POST("/auth/refresh", handler.RefreshToken)
v1.POST("/auth/logout", handler.Logout)
v1.GET("/auth/me", authMiddleware, handler.Me)
```

### 需要認證的資源（全部套用 `authMiddleware`）

```go
authed := v1.Group("/", authMiddleware)
{
    authed.POST("/body-metrics", handler.CreateBodyMetric)
    authed.GET("/body-metrics", handler.ListBodyMetrics)
    authed.PATCH("/body-metrics/:id", handler.UpdateBodyMetric)
    authed.DELETE("/body-metrics/:id", handler.DeleteBodyMetric)

    authed.POST("/sleep-logs", handler.CreateSleepLog)
    authed.GET("/sleep-logs", handler.ListSleepLogs)
    authed.PATCH("/sleep-logs/:id", handler.UpdateSleepLog)
    authed.DELETE("/sleep-logs/:id", handler.DeleteSleepLog)

    authed.POST("/daily-activities", handler.CreateDailyActivity)
    authed.GET("/daily-activities", handler.ListDailyActivities)
    authed.PATCH("/daily-activities/:id", handler.UpdateDailyActivity)
    authed.DELETE("/daily-activities/:id", handler.DeleteDailyActivity)
}
```

所有 `/v1/*` 資源（除了 auth 流程）未帶有效 JWT 一律回傳 `401 Unauthorized`。

---

## 8. sqlc

sqlc 從 `db/queries/` 的 SQL 自動生成 type-safe 的 Go 程式碼到 `db/sqlc/`。

目前已有 queries：

| 檔案                          | 包含的 query                                                     |
| ----------------------------- | ---------------------------------------------------------------- |
| `db/queries/users.sql`        | `UpsertUser`、`GetUserByGoogleID`、`GetUserByID`                 |
| `db/queries/refresh_tokens.sql` | `CreateRefreshToken`、`GetRefreshToken`、`RevokeRefreshToken`  |
| `db/queries/body_metrics.sql` | `CreateBodyMetric`、`GetBodyMetric`、`ListBodyMetrics`、`UpdateBodyMetric`、`DeleteBodyMetric` |
| `db/queries/sleep_logs.sql`   | `CreateSleepLog`、`GetSleepLog`、`ListSleepLogs`、`UpdateSleepLog`、`DeleteSleepLog` |
| `db/queries/daily_activities.sql` | `CreateDailyActivity`、`GetDailyActivity`、`ListDailyActivities`、`UpdateDailyActivity`、`DeleteDailyActivity` |

重新生成指令：

```bash
cd backend && sqlc generate
```

---

## 9. 認證與 JWT

### Google OAuth 流程

```
使用者點擊登入
→ GET /v1/auth/google（後端重導至 Google 授權頁）
→ 使用者同意授權
→ Google 回調 GET /v1/auth/google/callback?code=...
→ 後端交換 code 取得 Google profile
→ Upsert users 資料表
→ 簽發 Access Token（15 分鐘）+ Refresh Token（7 天）
→ 以 httpOnly Cookie 回傳給前端
```

### JWT 內容

Access Token 只存 `user_id`，不放敏感資訊：

```go
type Claims struct {
    UserID uuid.UUID `json:"user_id"`
    jwt.RegisteredClaims
}
```

### Auth Middleware

`internal/middleware/auth.go` 從 Cookie 或 `Authorization: Bearer` header 取出 JWT，驗證後把 `user_id` 注入 Gin context：

```go
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 取出 token → 驗證 → 取得 user_id
        c.Set("user_id", claims.UserID)
        c.Next()
    }
}
```

Handler 內用這個方式取得 user_id：

```go
userID := c.MustGet("user_id").(uuid.UUID)
```

### Refresh Token

- 儲存在 `refresh_tokens` 資料表，存 hash（不存明文）
- 前端呼叫 `POST /v1/auth/refresh` 換新的 Access Token
- 登出時 revoke（標記 `revoked = true`）

---

## 10. Repository 層

每個資料模組都有對應的 repository，負責把 sqlc 方法包起來，並注入 `user_id` 做資料隔離：

```go
// 所有查詢都帶 user_id，確保使用者 A 看不到使用者 B 的資料
func (r *BodyMetricsRepository) List(ctx context.Context, userID uuid.UUID, params ...) ([]BodyMetric, error) {
    return r.queries.ListBodyMetrics(ctx, &sqlcdb.ListBodyMetricsParams{
        UserID: userID,
        // ...
    })
}
```

`isolation_test.go` 專門測試跨使用者資料隔離：建兩個 user，確認 user A 的查詢拿不到 user B 的資料。

---

## 11. Handler 設計與 Store Interface

每個 handler 依賴 interface，而不是直接依賴 repository：

```go
type BodyMetricsStore interface {
    Create(ctx context.Context, userID uuid.UUID, params CreateParams) (BodyMetric, error)
    Get(ctx context.Context, userID uuid.UUID, id uuid.UUID) (BodyMetric, error)
    List(ctx context.Context, userID uuid.UUID, params ListParams) ([]BodyMetric, error)
    Update(ctx context.Context, userID uuid.UUID, id uuid.UUID, params UpdateParams) (BodyMetric, error)
    Delete(ctx context.Context, userID uuid.UUID, id uuid.UUID) error
}
```

好處：
- handler 與資料存取層解耦
- 單元測試可以注入 mock store
- handler 只處理 HTTP 邊界

分層口訣：
- **handler**：處理 HTTP（解析 request、回傳 response、錯誤格式）
- **repository**：處理資料存取（sqlc query、user_id 隔離）
- **auth**：處理 JWT 簽發與驗證
- **interface**：描述依賴邊界

---

## 12. 測試

測試分三層，全部位於 `backend/`：

| 層級          | 位置                                     | 說明                                              |
| ------------- | ---------------------------------------- | ------------------------------------------------- |
| Handler 單元測試 | `internal/handler/*_test.go`          | 使用 mock store，驗證 HTTP request/response 邊界  |
| Repository 整合測試 | `internal/repository/*_test.go`   | 使用 testcontainers 啟動真實 PostgreSQL，測試 DB 行為 |
| 資料隔離測試  | `internal/repository/isolation_test.go` | 確認 user_id 過濾正確，A 看不到 B 的資料          |

執行全部測試：

```bash
cd backend && go test ./...
```

執行單一檔案：

```bash
go test ./internal/handler/... -v
go test ./internal/repository/... -v
```

### Handler 測試模式（table-driven）

```go
func TestCreateBodyMetric(t *testing.T) {
    tests := []struct {
        name       string
        body       string
        wantStatus int
    }{
        {"valid input", `{"weight_kg":72.5,"recorded_at":"..."}`, 201},
        {"missing recorded_at", `{"weight_kg":72.5}`, 400},
        {"unauthenticated", `{"weight_kg":72.5,"recorded_at":"..."}`, 401},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) { /* ... */ })
    }
}
```

---

## 13. 開發前的思考順序

新增一個 endpoint 的標準順序：

1. request / response 長什麼樣子？
2. 驗證規則是什麼？
3. 需要哪些 DB 操作？（在 `db/queries/` 寫 SQL）
4. 跑 `sqlc generate` 生成 Go 程式碼
5. 在 `repository/` 包裝 query（注入 user_id）
6. 在 `handler/` 實作 HTTP 邊界
7. 先寫 handler 測試，再實作

---

## 14. 常見坑

### 1. 忘記帶 user_id

所有資料查詢都必須帶 `user_id`，否則所有使用者的資料會混在一起。repository 層負責確保這件事，不要跳過 repository 直接在 handler 呼叫 sqlc。

### 2. 把 SQL 邏輯寫在 handler

handler 只處理 HTTP。SQL 邏輯寫在 `db/queries/*.sql`，資料存取邏輯寫在 repository。

### 3. JWT 存在 localStorage

這個專案採用 httpOnly Cookie，前端 JS 無法讀取，防止 XSS 竊取 token。不要改成 localStorage。

### 4. 沒分清楚 handler 測試和 repository 測試

- Handler 測試用 mock store：驗證 HTTP 邊界（400/401/404 等）
- Repository 測試用真實 DB：驗證 SQL 正確性

這兩種測試目的不同，不要混用。

### 5. migration 執行後不能修改

已套用的 migration 不能改，要加新的。如果本地開發想重置，可以：

```bash
migrate -path ./db/migrations -database "..." down
```

---

## 15. 常用指令

| 指令                        | 用途                               |
| --------------------------- | ---------------------------------- |
| `go run ./cmd/api`          | 啟動伺服器                         |
| `go build ./cmd/api`        | 編譯 binary                        |
| `go mod tidy`               | 整理依賴                           |
| `go vet ./...`              | 靜態分析                           |
| `go test ./...`             | 執行所有測試                       |
| `go test ./... -coverprofile=coverage.out` | 產出覆蓋率報告     |
| `go tool cover -html=coverage.out` | 瀏覽器查看覆蓋率             |
| `sqlc generate`             | 從 SQL 生成 Go 程式碼              |

---

## 16. 專案閱讀順序

1. `cmd/api/main.go`：理解啟動流程與路由組裝
2. `internal/config/config.go`：理解設定結構
3. `internal/middleware/auth.go`：理解 JWT 驗證流程
4. `internal/handler/auth.go`：理解 Google OAuth callback
5. `internal/handler/body_metrics.go`：理解一個完整 CRUD handler
6. `db/queries/body_metrics.sql` + `db/sqlc/`：理解 sqlc 怎麼把 SQL 接進 Go
7. `internal/repository/body_metrics.go`：理解 repository 層的 user_id 隔離
8. `internal/handler/body_metrics_test.go`：理解 mock store + table-driven 測試

---

## 17. 學習資源

- [Go 官方 Tour](https://go.dev/tour/)（語法入門）
- [Gin 官方文件](https://gin-gonic.com/docs/)
- [sqlc 官方文件](https://docs.sqlc.dev/)
- [golang-migrate 說明](https://github.com/golang-migrate/migrate)
- [golang-jwt 文件](https://github.com/golang-jwt/jwt)
