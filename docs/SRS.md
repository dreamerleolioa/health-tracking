# 軟體需求規格書 (Software Requirements Specification)

**專案名稱：** 全方位生活與健康追蹤儀表板
**版本：** 1.0.0
**日期：** 2026-03-24
**作者：** Leo
**技術棧：**
- 前端：SvelteKit + Tailwind CSS
- 後端：Go (Gin)
- 資料庫：PostgreSQL

---

## 目錄

1. [核心功能模組](#1-核心功能模組)
2. [資料庫模型設計](#2-資料庫模型設計)
3. [API 接口設計](#3-api-接口設計)
4. [非功能性需求](#4-非功能性需求)
5. [開發進度里程碑](#5-開發進度里程碑)

---

## 1. 核心功能模組

### 1.1 體位數據管理 (Body Metrics)

| 欄位 | 類型 | 說明 |
|------|------|------|
| 體重 | float | 單位 kg，精度 0.1 |
| 體脂率 | float | 單位 %，精度 0.1 |
| 肌肉率 | float | 單位 %，精度 0.1 |
| 內臟脂肪 | int | 等級 1–30 |

**功能需求：**
- CRUD：新增、查詢（單筆 / 區間列表）、更新、刪除
- 查詢支援日期範圍篩選（`from` / `to` query param）
- 同一天多筆記錄以最後一筆為主顯示（可設定）

---

### 1.2 生活因子追蹤 (Lifestyle Factors)

#### 1.2.1 睡眠紀錄

| 欄位 | 說明 |
|------|------|
| 上床時間 | `sleep_at TIMESTAMPTZ` |
| 起床時間 | `wake_at TIMESTAMPTZ` |
| 異常喚醒 | `abnormal_wake BOOLEAN`，自動標記 03:00–04:00 醒來 |
| 睡眠品質 | `quality SMALLINT`，1–5 主觀評分 |
| 備註 | `note TEXT` |

**功能需求：**
- 自動計算睡眠時長
- 異常喚醒 (3–4 AM) 自動標記，並可在趨勢圖中以特殊顏色呈現
- 支援查詢「含異常喚醒」的歷史紀錄

#### 1.2.2 每日活動 / 通勤

| 欄位 | 說明 |
|------|------|
| 步數 | `steps INT` |
| 通勤模式 | `commute_mode ENUM('scooter', 'train', 'walk', 'other')` |
| 通勤時長 | `commute_minutes INT` |
| 備註 | `note TEXT` |

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
-- ENUM Types
-- =====================
CREATE TYPE commute_mode AS ENUM ('scooter', 'train', 'walk', 'other');

-- =====================
-- 體位數據
-- =====================
CREATE TABLE body_metrics (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
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

-- =====================
-- 睡眠紀錄
-- =====================
CREATE TABLE sleep_logs (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
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

-- =====================
-- 每日活動
-- =====================
CREATE TABLE daily_activities (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    activity_date    DATE         NOT NULL UNIQUE,
    steps            INT          CHECK (steps >= 0),
    commute_mode     commute_mode,
    commute_minutes  INT          CHECK (commute_minutes >= 0),
    note             TEXT,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_daily_activities_date ON daily_activities (activity_date DESC);

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

### 3.1 通用規範

**Base URL：** `https://api.health.local/v1`

**通用 Header：**
```
Content-Type: application/json
Authorization: Bearer <token>   // MVP 後加入
```

**統一錯誤格式：**
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "weight_kg must be between 30 and 300",
    "details": [
      { "field": "weight_kg", "issue": "out_of_range" }
    ]
  }
}
```

**HTTP 狀態碼慣例：**

| 狀態碼 | 情境 |
|--------|------|
| 200 | 查詢 / 更新成功 |
| 201 | 新增成功 |
| 204 | 刪除成功（無 body） |
| 400 | 請求格式或驗證錯誤 |
| 404 | 資源不存在 |
| 409 | 資料衝突（如同日 activity 重複） |
| 500 | 伺服器內部錯誤 |

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
  "data": [ /* BodyMetric[] */ ],
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
  "data": [ /* SleepLog[] */ ],
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
- **Rate Limiting（MVP 後）：** 使用 `golang.org/x/time/rate` 或 `ulule/limiter`

---

### 4.3 前端 PWA 離線緩存策略 (SvelteKit)

使用 `@vite-pwa/sveltekit` 實作 Service Worker：

| 資源類型 | 緩存策略 |
|----------|----------|
| App Shell (HTML/JS/CSS) | `CacheFirst`，版本更新時強制刷新 |
| API 資料 (GET) | `NetworkFirst`，fallback 至 staleWhileRevalidate |
| 圖表靜態資源 | `CacheFirst`，TTL 7 天 |
| POST/PATCH/DELETE | 不緩存，失敗時以 Toast 通知使用者 |

**離線提示：**
```svelte
// 偵測網路狀態，離線時顯示 banner 並禁用寫入操作
import { useOnline } from '@vueuse/core' // 或手動實作
```

---

## 5. 開發進度里程碑

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

### Milestone 2 — 生活因子（第 4–5 週）

- [ ] `sleep_logs` CRUD + 異常喚醒自動標記
- [ ] `daily_activities` CRUD
- [ ] 儀表板：睡眠異常標記疊加至體重趨勢圖
- [ ] 步數熱度背景

### Milestone 3 — 強化與部署（第 6–8 週）

- [ ] SvelteKit PWA 離線緩存
- [ ] JWT 認證（個人使用可簡化為 API Key）
- [ ] Rate Limiting
- [ ] CI/CD（GitHub Actions → Railway / Fly.io）
- [ ] 正式域名 + HTTPS

### Milestone 5 — 進階分析（未來規劃）

- [ ] 體重 × 睡眠品質相關係數計算
- [ ] 通勤模式 × 步數統計分析
- [ ] 匯出 CSV / PDF 報告

---

*本文件隨開發進度持續更新。*
