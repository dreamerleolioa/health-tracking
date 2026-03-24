# Go 後端學習指南

> 針對本專案的 Go 後端入門，說明專案結構、核心套件與開發流程。

---

## 1. 技術棧一覽

| 角色 | 套件 | 說明 |
|------|------|------|
| HTTP 框架 | `gin-gonic/gin` | 路由、middleware、JSON binding |
| 資料庫驅動 | `jackc/pgx/v5` | PostgreSQL 驅動（透過 `stdlib` 介面） |
| Migration | `golang-migrate/migrate/v4` | 管理 DB schema 版本 |
| Query 生成 | `sqlc`（待導入） | 從 SQL 自動生成 type-safe Go 程式碼 |
| 輸入驗證 | `go-playground/validator/v10` | struct tag 驗證 |
| 環境變數 | `joho/godotenv` | 載入 `.env` 檔 |

---

## 2. 專案結構

```
backend/
├── cmd/
│   └── api/
│       └── main.go           ← 程式進入點
├── db/
│   ├── migrations/           ← SQL migration 檔（up / down 成對）
│   ├── queries/              ← sqlc 用的 SQL query 檔（待新增）
│   └── sqlc/                 ← sqlc 生成的 Go 程式碼（待生成）
├── internal/
│   ├── config/
│   │   └── config.go         ← 讀取環境變數、組成 Config struct
│   ├── db/
│   │   ├── db.go             ← 建立 DB 連線、設定連線池
│   │   └── migrate.go        ← 執行 migration
│   ├── handler/
│   │   └── health.go         ← HTTP handler（每個功能一個檔案）
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

---

## 4. 環境變數

透過 `godotenv` 依 `APP_ENV` 載入對應的 `.env` 檔：

| `APP_ENV` | 載入的檔案 |
|-----------|-----------|
| `local`（預設） | `.env.local` |
| `production` | `.env.production` |

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

## 7. 新增一個 API Handler

以新增 `GET /v1/body-metrics` 為例：

**1. 新增 handler 檔案** `internal/handler/body_metrics.go`
```go
package handler

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

func ListBodyMetrics(c *gin.Context) {
    // 之後接 sqlc query
    c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
}
```

**2. 在 `main.go` 註冊路由**
```go
v1 := r.Group("/v1")
{
    v1.GET("/health", handler.HealthCheck)
    v1.GET("/body-metrics", handler.ListBodyMetrics)  // 新增這行
}
```

---

## 8. sqlc（待導入）

sqlc 讓你寫 SQL，它自動生成對應的 Go struct 與 function，避免手動拼接 SQL。

流程：
1. 在 `db/queries/` 寫 SQL query（加上 sqlc 註解）
2. 執行 `sqlc generate`
3. `db/sqlc/` 自動產生 type-safe 的 Go 程式碼
4. 在 handler 中直接呼叫生成的 function

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

---

## 10. 常用指令

| 指令 | 用途 |
|------|------|
| `go run ./cmd/api` | 啟動伺服器 |
| `go build ./cmd/api` | 編譯 binary |
| `go mod tidy` | 整理依賴（移除未用的、補上缺少的） |
| `go vet ./...` | 靜態分析 |
| `sqlc generate` | 從 SQL 生成 Go 程式碼（待導入） |

---

## 11. 學習資源

- [Go 官方 Tour](https://go.dev/tour/)（語法入門）
- [Gin 官方文件](https://gin-gonic.com/docs/)
- [sqlc 官方文件](https://docs.sqlc.dev/)
- [golang-migrate 說明](https://github.com/golang-migrate/migrate)
