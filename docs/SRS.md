# 軟體需求規格書 (Software Requirements Specification)

**專案名稱：** 全方位生活與健康追蹤儀表板
**版本：** 1.1.0
**日期：** 2026-03-24
**作者：** Leo
**技術棧：**

- 前端：SvelteKit + Tailwind CSS
- 後端：Go (Gin)
- 資料庫：PostgreSQL

---

## 目錄

1. [核心功能模組](#1-核心功能模組)
   - [1.0 身份認證與會員管理](#10-身份認證與會員管理-authentication--member-management)
2. [資料庫模型設計](#2-資料庫模型設計)
3. [API 接口設計](#3-api-接口設計)
4. [非功能性需求](#4-非功能性需求)
5. [開發進度里程碑](#5-開發進度里程碑)

---

## 1. 核心功能模組

### 1.0 身份認證與會員管理 (Authentication & Member Management)

> 所有健康數據 API 均需通過認證才能存取，每位使用者只能讀寫自己的資料。

#### 登入方式

- **Google OAuth 2.0**：唯一登入方式，不提供帳號密碼註冊
- 登入後由後端簽發 JWT（Access Token + Refresh Token）
- Access Token 存於 `httpOnly Cookie` 或 `localStorage`（依安全評估決定）
- Refresh Token 存於資料庫，支援撤銷

#### 會員資料

| 欄位            | 說明                            |
| --------------- | ------------------------------- |
| `id`            | UUID，系統內部主鍵              |
| `google_id`     | Google 帳號的唯一識別碼         |
| `email`         | Google 帳號 email               |
| `display_name`  | 顯示名稱（來自 Google profile） |
| `avatar_url`    | 頭像 URL（來自 Google profile） |
| `created_at`    | 首次登入建立時間                |
| `last_login_at` | 最近登入時間                    |

#### 功能需求

- 使用者點擊「以 Google 登入」→ 導向 Google OAuth 授權頁 → 回調後建立或更新會員資料
- 首次登入自動建立帳號（無需事先註冊）
- 所有資料表加入 `user_id` 外鍵，查詢時自動以登入者 ID 過濾
- 提供登出功能（清除 Token / Cookie）
- 未登入時存取任何 `/v1/*` 資源一律回傳 `401 Unauthorized`

---

### 1.1 體位數據管理 (Body Metrics)

| 欄位     | 類型  | 說明              |
| -------- | ----- | ----------------- |
| 體重     | float | 單位 kg，精度 0.1 |
| 體脂率   | float | 單位 %，精度 0.1  |
| 肌肉率   | float | 單位 %，精度 0.1  |
| 內臟脂肪 | int   | 等級 1–30         |

**功能需求：**

- CRUD：新增、查詢（單筆 / 區間列表）、更新、刪除
- 查詢支援日期範圍篩選（`from` / `to` query param）
- 同一天多筆記錄以最後一筆為主顯示（可設定）

---

### 1.2 生活因子追蹤 (Lifestyle Factors)

#### 1.2.1 睡眠紀錄

| 欄位     | 說明                                               |
| -------- | -------------------------------------------------- |
| 上床時間 | `sleep_at TIMESTAMPTZ`                             |
| 起床時間 | `wake_at TIMESTAMPTZ`                              |
| 異常喚醒 | `abnormal_wake BOOLEAN`，自動標記 03:00–04:00 醒來 |
| 睡眠品質 | `quality SMALLINT`，1–5 主觀評分                   |
| 備註     | `note TEXT`                                        |

**功能需求：**

- 自動計算睡眠時長
- 異常喚醒 (3–4 AM) 自動標記，並可在趨勢圖中以特殊顏色呈現
- 支援查詢「含異常喚醒」的歷史紀錄

#### 1.2.2 每日活動 / 通勤

| 欄位     | 說明                                                     |
| -------- | -------------------------------------------------------- |
| 步數     | `steps INT`                                              |
| 通勤模式 | `commute_mode ENUM('scooter', 'train', 'walk', 'other')` |
| 通勤時長 | `commute_minutes INT`                                    |
| 備註     | `note TEXT`                                              |

---

### 1.3 數據視覺化 (Dashboard)

- **體重趨勢折線圖**：X 軸為日期，Y 軸為體重/體脂/肌肉率，三線疊加
- **關聯標記**：
  - 異常睡眠日期以橘色三角形標記於 X 軸
  - 步數以背景色深淺呈現（熱度圖概念）
- **圖表庫：** 使用 `layerchart`（基於 d3，SvelteKit 友善）或 `chart.js` + `svelte-chartjs`

---

## 2. 資料庫模型設計

> 所有時間欄位統一使用 `TIMESTAMPTZ`，儲存 UTC，應用層轉換為 Asia/Taipei (UTC+8) 顯示。

```sql
-- =====================
-- Extension
-- =====================
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =====================
-- 使用者（Google OAuth）
-- =====================
CREATE TABLE users (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    google_id       TEXT        NOT NULL UNIQUE,
    email           TEXT        NOT NULL UNIQUE,
    display_name    TEXT,
    avatar_url      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_google_id ON users (google_id);

-- =====================
-- Refresh Token
-- =====================
CREATE TABLE refresh_tokens (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT        NOT NULL UNIQUE,  -- bcrypt hash，不儲存明文
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked     BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens (user_id);

-- =====================
-- ENUM Types（需在依賴此 ENUM 的資料表之前建立）
-- =====================
CREATE TYPE commute_mode AS ENUM ('scooter', 'train', 'walk', 'other');

-- =====================
-- 體位數據
-- =====================
CREATE TABLE body_metrics (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    weight_kg     NUMERIC(5,2),
    body_fat_pct  NUMERIC(5,2),
    muscle_pct    NUMERIC(5,2),
    visceral_fat  SMALLINT    CHECK (visceral_fat BETWEEN 1 AND 30),
    recorded_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    note          TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_body_metrics_recorded_at ON body_metrics (recorded_at DESC);
CREATE INDEX idx_body_metrics_user_id ON body_metrics (user_id);

-- =====================
-- 睡眠紀錄
-- =====================
CREATE TABLE sleep_logs (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sleep_at        TIMESTAMPTZ NOT NULL,
    wake_at         TIMESTAMPTZ NOT NULL,
    duration_min    INT         GENERATED ALWAYS AS (
                        EXTRACT(EPOCH FROM (wake_at - sleep_at)) / 60
                    )::INT STORED,
    abnormal_wake   BOOLEAN     NOT NULL DEFAULT FALSE,
    quality         SMALLINT    CHECK (quality BETWEEN 1 AND 5),
    note            TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 自動標記 abnormal_wake：於 INSERT/UPDATE 時檢查 wake_at 是否介於 03:00–04:00 (local time)
CREATE OR REPLACE FUNCTION set_abnormal_wake()
RETURNS TRIGGER AS $$
BEGIN
    NEW.abnormal_wake := (
        EXTRACT(HOUR FROM NEW.wake_at AT TIME ZONE 'Asia/Taipei') = 3
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_abnormal_wake
BEFORE INSERT OR UPDATE ON sleep_logs
FOR EACH ROW EXECUTE FUNCTION set_abnormal_wake();

CREATE INDEX idx_sleep_logs_sleep_at     ON sleep_logs (sleep_at DESC);
CREATE INDEX idx_sleep_logs_abnormal     ON sleep_logs (abnormal_wake) WHERE abnormal_wake = TRUE;
CREATE INDEX idx_sleep_logs_user_id      ON sleep_logs (user_id);

-- =====================
-- 每日活動
-- =====================
CREATE TABLE daily_activities (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    activity_date    DATE         NOT NULL,
    UNIQUE (user_id, activity_date),
    steps            INT          CHECK (steps >= 0),
    commute_mode     commute_mode,
    commute_minutes  INT          CHECK (commute_minutes >= 0),
    note             TEXT,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_daily_activities_date ON daily_activities (activity_date DESC);
CREATE INDEX idx_daily_activities_user_id ON daily_activities (user_id);

-- =====================
-- updated_at 自動更新觸發器（通用）
-- =====================
CREATE OR REPLACE FUNCTION touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_body_metrics_updated_at
    BEFORE UPDATE ON body_metrics
    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

CREATE TRIGGER trg_sleep_logs_updated_at
    BEFORE UPDATE ON sleep_logs
    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

CREATE TRIGGER trg_daily_activities_updated_at
    BEFORE UPDATE ON daily_activities
    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

```

---

## 3. API 接口設計

### 3.0 認證 `/v1/auth`

#### GET `/v1/auth/google`

> 重定向至 Google OAuth 授權頁面。

#### GET `/v1/auth/google/callback`

> Google OAuth 回調，交換 code 取得 tokens，建立或更新 user，簽發 JWT。

```json
// Response 200（設置 httpOnly Cookie: access_token, refresh_token）
{
  "data": {
    "user": {
      "id": "uuid",
      "email": "leo@gmail.com",
      "display_name": "Leo",
      "avatar_url": "https://..."
    }
  }
}
```

#### POST `/v1/auth/refresh`

> 使用 Refresh Token 換取新 Access Token。

```json
// Request（從 Cookie 自動帶入，或 body 傳入）
{ "refresh_token": "<token>" }

// Response 200
{ "data": { "access_token": "<jwt>" } }
```

#### POST `/v1/auth/logout`

> 撤銷 Refresh Token，清除 Cookie。Response 204 No Content。

#### GET `/v1/auth/me`

> 取得當前登入使用者資訊。

```json
// Response 200
{
  "data": {
    "id": "uuid",
    "email": "leo@gmail.com",
    "display_name": "Leo",
    "avatar_url": "https://...",
    "created_at": "2026-03-24T00:00:00Z",
    "last_login_at": "2026-03-24T08:00:00Z"
  }
}
```

---

### 3.1 通用規範

**Base URL：** `https://api.health.local/v1`

**通用 Header：**

```
Content-Type: application/json
Authorization: Bearer <access_token>
```

**統一錯誤格式：**

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "weight_kg must be between 30 and 300",
    "details": [{ "field": "weight_kg", "issue": "out_of_range" }]
  }
}
```

**HTTP 狀態碼慣例：**

| 狀態碼 | 情境                             |
| ------ | -------------------------------- |
| 200    | 查詢 / 更新成功                  |
| 201    | 新增成功                         |
| 204    | 刪除成功（無 body）              |
| 400    | 請求格式或驗證錯誤               |
| 404    | 資源不存在                       |
| 409    | 資料衝突（如同日 activity 重複） |
| 500    | 伺服器內部錯誤                   |

---

### 3.2 體位數據 `/v1/body-metrics`

#### POST `/v1/body-metrics`

```json
// Request
{
  "weight_kg": 72.5,
  "body_fat_pct": 18.2,
  "muscle_pct": 35.2,
  "visceral_fat": 8,
  "recorded_at": "2026-03-24T08:00:00+08:00",
  "note": "早上空腹測量"
}

// Response 201
{
  "data": {
    "id": "uuid",
    "weight_kg": 72.5,
    "body_fat_pct": 18.2,
    "muscle_pct": 35.2,
    "visceral_fat": 8,
    "recorded_at": "2026-03-24T00:00:00Z",
    "note": "早上空腹測量",
    "created_at": "2026-03-24T00:01:00Z"
  }
}
```

#### GET `/v1/body-metrics?from=2026-01-01&to=2026-03-24&limit=90`

```json
// Response 200
{
  "data": [
    /* BodyMetric[] */
  ],
  "meta": {
    "total": 45,
    "from": "2026-01-01",
    "to": "2026-03-24"
  }
}
```

#### PATCH `/v1/body-metrics/:id`

> 僅更新提供的欄位（partial update）

#### DELETE `/v1/body-metrics/:id`

> Response 204 No Content

---

### 3.3 睡眠紀錄 `/v1/sleep-logs`

#### POST `/v1/sleep-logs`

```json
// Request
{
  "sleep_at": "2026-03-23T23:30:00+08:00",
  "wake_at": "2026-03-24T07:00:00+08:00",
  "quality": 3,
  "note": "半夜三點多醒來一次"
}

// Response 201
{
  "data": {
    "id": "uuid",
    "sleep_at": "2026-03-23T15:30:00Z",
    "wake_at": "2026-03-23T23:00:00Z",
    "duration_min": 450,
    "abnormal_wake": false,
    "quality": 3,
    "note": "半夜三點多醒來一次"
  }
}
```

#### GET `/v1/sleep-logs?from=2026-03-01&abnormal_only=true`

```json
{
  "data": [
    /* SleepLog[] */
  ],
  "meta": { "total": 5 }
}
```

---

### 3.4 每日活動 `/v1/daily-activities`

#### POST `/v1/daily-activities`

```json
// Request
{
  "activity_date": "2026-03-24",
  "steps": 8500,
  "commute_mode": "train",
  "commute_minutes": 45,
  "note": ""
}
```

#### GET `/v1/daily-activities?from=2026-03-01&to=2026-03-24`

---

## 4. 非功能性需求

### 4.1 環境隔離

專案根目錄使用 `.env` 管理設定，透過 `godotenv` 載入。

```
# .env.local
APP_ENV=local
SERVER_PORT=8080
DATABASE_URL=postgres://postgres:password@localhost:5432/health_tracking?sslmode=disable
CORS_ORIGINS=http://localhost:5173

# .env.production
APP_ENV=production
SERVER_PORT=8080
DATABASE_URL=postgres://user:pass@cloud-db-host:5432/health_tracking?sslmode=require
CORS_ORIGINS=https://health.yourdomain.com
```

**載入策略（Go）：**

```go
// main.go
env := os.Getenv("APP_ENV")
if env == "" {
    env = "local"
}
godotenv.Load(".env." + env)
godotenv.Load(".env") // fallback
```

**.gitignore 必加：**

```
.env
.env.local
.env.production
```

---

### 4.2 Go 後端效能與安全性

#### 併發處理

- 使用 Gin 預設的 goroutine-per-request 模型，足應個人專案規模
- DB 連線池設定：
  ```go
  db.SetMaxOpenConns(10)
  db.SetMaxIdleConns(5)
  db.SetConnMaxLifetime(5 * time.Minute)
  ```
- 所有 DB 查詢使用 `context.WithTimeout(ctx, 5*time.Second)` 避免慢查詢阻塞

#### 安全性

- **SQL Injection：** 全面使用 `sqlc` 生成的 type-safe query，禁止手動拼接 SQL
- **輸入驗證：** 使用 `go-playground/validator` 搭配 Gin binding
- **CORS：** 使用 `gin-contrib/cors`，限定 `CORS_ORIGINS` 白名單
- **JWT：** 使用 `golang-jwt/jwt/v5`，Access Token 有效期 15 分鐘，Refresh Token 7 天；Token 僅存 `user_id`，不含敏感資訊
- **OAuth State：** Google OAuth 流程使用隨機 `state` 參數防止 CSRF
- **Rate Limiting（MVP 後）：** 使用 `golang.org/x/time/rate` 或 `ulule/limiter`

---

### 4.3 後端測試規範

所有後端業務邏輯與 API 端點均需撰寫測試，確保功能正確性與回歸安全性。

#### 測試分層

| 層級            | 工具                                 | 說明                                                                      |
| --------------- | ------------------------------------ | ------------------------------------------------------------------------- |
| 單元測試        | Go 標準 `testing` 套件               | 測試純函式、驗證邏輯、時間計算（如 `duration_min`、`abnormal_wake` 判斷） |
| 整合測試        | `testcontainers-go`                  | 啟動真實 PostgreSQL 容器，測試 sqlc query 與 DB 觸發器行為                |
| API 測試（E2E） | `net/http/httptest` + `gin.TestMode` | 測試完整 HTTP 請求 / 回應流程，含 middleware、認證、錯誤格式              |

#### 測試覆蓋範圍要求

**Handler 層**

- 每個 API 端點至少涵蓋：正常回應（2xx）、驗證錯誤（400）、資源不存在（404）、未授權（401）
- Auth 流程：OAuth callback 成功、state 不符拒絕、token 過期 / 撤銷

**Service / 業務邏輯層**

- 輸入驗證規則（邊界值、格式錯誤）
- `abnormal_wake` 自動標記邏輯（含時區轉換）
- JWT 簽發與驗證（過期、無效簽章）

**Repository / DB 層**

- CRUD 操作正確性
- `user_id` 隔離：確保使用者 A 無法存取使用者 B 的資料
- UNIQUE 約束衝突（如同 user 同日 activity 重複新增）

#### 測試慣例

```go
// 測試檔案命名：與被測檔案同目錄，加 _test.go 後綴
// 例：handler/health.go → handler/health_test.go

// Table-driven tests 為標準寫法
func TestCreateBodyMetric(t *testing.T) {
    tests := []struct {
        name       string
        input      CreateBodyMetricRequest
        wantStatus int
    }{
        {"valid input", validInput, http.StatusCreated},
        {"missing weight", noWeightInput, http.StatusBadRequest},
        {"unauthenticated", validInput, http.StatusUnauthorized}, // no token
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) { /* ... */ })
    }
}
```

#### CI 整合

- Pull Request 觸發 `go test ./...`，任一測試失敗則阻擋合併
- 產出覆蓋率報告（`-coverprofile`），目標覆蓋率 ≥ 80%

#### ⚠️ PostgreSQL 時區資料健康檢查

`ListSleepLogs` 查詢使用 `AT TIME ZONE 'Asia/Taipei'`。`postgres:16-alpine` 目前包含時區資料，但未來 base image 升版可能靜默移除，導致查詢結果錯誤卻不報錯。

**規範：** 在所有使用 `AT TIME ZONE` 查詢的整合測試 `TestMain` 中，於 migration 執行後加入以下健康檢查：

```go
var tzCheck string
if err := db.QueryRowContext(ctx, "SELECT (NOW() AT TIME ZONE 'Asia/Taipei')::TEXT").Scan(&tzCheck); err != nil {
    panic("PostgreSQL timezone data unavailable: " + err.Error())
}
```

---

### 4.4 前端 PWA 離線緩存策略 (SvelteKit)

使用 `@vite-pwa/sveltekit` 實作 Service Worker：

| 資源類型                | 緩存策略                                         |
| ----------------------- | ------------------------------------------------ |
| App Shell (HTML/JS/CSS) | `CacheFirst`，版本更新時強制刷新                 |
| API 資料 (GET)          | `NetworkFirst`，fallback 至 staleWhileRevalidate |
| 圖表靜態資源            | `CacheFirst`，TTL 7 天                           |
| POST/PATCH/DELETE       | 不緩存，失敗時以 Toast 通知使用者                |

**離線提示：**

```svelte
// 偵測網路狀態，離線時顯示 banner 並禁用寫入操作
import { useOnline } from '@vueuse/core' // 或手動實作
```

---

## 5. 開發進度里程碑

> **完成標準（Definition of Done）：** 所有 checklist 項目，包含後端 API 開發，必須在對應的測試程式碼全數通過（`go test ./...` 無失敗）後，才能標記為完成（`[x]`）。計劃文件同此規範。

### Milestone 0 — 專案初始化（第 1 週）

- [x] 建立 `backend/` Go 專案結構（Gin + sqlc + godotenv）
- [x] 建立 `frontend/` SvelteKit 專案
- [x] Docker Compose 啟動 PostgreSQL
- [x] 執行 DB migration（`golang-migrate`）

### Milestone 1 — MVP：體位數據（第 2–3 週）

- [ ] `body_metrics` CRUD API 完成
- [ ] sqlc query 生成
- [ ] SvelteKit 新增/列表頁面
- [ ] 體重趨勢折線圖（基礎版，無關聯標記）
- [ ] `.env` 環境切換驗證
- [ ] `body_metrics` Handler / Repository 單元測試與整合測試

### Milestone 2 — 生活因子（第 4–5 週）

- [ ] `sleep_logs` CRUD + 異常喚醒自動標記
- [ ] `daily_activities` CRUD
- [ ] 儀表板：睡眠異常標記疊加至體重趨勢圖
- [ ] 步數熱度背景
- [ ] `sleep_logs`、`daily_activities` Handler / Repository 測試（含 `abnormal_wake` 時區邊界測試）

### Milestone 3 — 認證與強化（第 6–8 週）

> ⚠️ **Migration 注意事項：** M2 的 `daily_activities` 採 `UNIQUE(activity_date)`（無 `user_id`）。M3 加入 `user_id` 時，必須以單一 transaction 依序執行以下步驟，否則對有既有資料的資料表執行 migration 會失敗：
>
> 1. 新增 `user_id UUID REFERENCES users(id)` — 先設為 nullable
> 2. 以系統/預設 `user_id` 填充既有資料列
> 3. `ALTER COLUMN user_id SET NOT NULL`
> 4. `DROP CONSTRAINT UNIQUE(activity_date)`
> 5. `ADD CONSTRAINT UNIQUE(user_id, activity_date)`

- [x] Google OAuth 2.0 登入（後端 `/v1/auth/*` 端點）
- [x] JWT 簽發 / Refresh Token 機制
- [x] SvelteKit 登入頁與 Auth 狀態管理（`+layout.svelte` guard）
- [x] 所有資料表加入 `user_id` 外鍵並套用 migration（含上方 `daily_activities` 安全遷移序列）
- [x] Auth 流程測試（OAuth callback、token 刷新、登出、401 防護、`user_id` 資料隔離）
- [x] CI/CD（GitHub Actions → Railway / Fly.io）；Pipeline 加入 `go test ./... -coverprofile` 覆蓋率門檻 ≥ 80%
- [x] 補全 handler 層 table-driven 測試（含 400 驗證錯誤、404 資源不存在、401 未授權案例），使後端覆蓋率從目前 51.5% 達到 ≥ 80%（參考 §4.3 測試覆蓋範圍要求）
- [x] SvelteKit PWA 離線緩存
- [x] Rate Limiting
- [x] 正式域名 + HTTPS

### Milestone 5 — 進階分析（未來規劃）

- [ ] 體重 × 睡眠品質相關係數計算
- [ ] 通勤模式 × 步數統計分析
- [ ] 匯出 CSV / PDF 報告

---

_本文件隨開發進度持續更新。_
