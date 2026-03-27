# Milestone 5 — 進階分析與匯出 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 實作體重 × 睡眠品質相關係數、通勤模式 × 步數統計，以及 CSV / PDF 匯出功能，並新增前端分析頁面。

**Architecture:**
後端新增 `/v1/analytics/*` 端點（相關係數與通勤統計）與 `/v1/export/csv` 端點（CSV 串流下載）。Pearson 相關係數在 Go service layer 計算（純函式）。前端新增 `/analytics` 路由，以 layerchart 呈現圖表；PDF 透過瀏覽器列印功能實現，不需後端。

**Tech Stack:** Go (Gin, sqlc, encoding/csv), SvelteKit, layerchart (d3), Tailwind CSS

---

## Task 1：Analytics SQL 查詢

**Files:**
- Create: `backend/db/queries/analytics.sql`
- Generate: `backend/db/sqlc/analytics.sql.go` （由 sqlc 生成，不手寫）

### Step 1：寫 SQL 查詢

```sql
-- backend/db/queries/analytics.sql

-- name: GetWeightSleepPoints :many
-- 取得體重與睡眠品質的配對資料（依日期 JOIN）
SELECT
    bm.recorded_at::DATE                AS date,
    CAST(bm.weight_kg AS FLOAT8)        AS weight_kg,
    sl.quality                          AS quality
FROM body_metrics bm
JOIN sleep_logs sl
    ON bm.recorded_at::DATE = sl.wake_at::DATE
   AND sl.user_id = bm.user_id
WHERE bm.user_id = sqlc.arg('user_id')
    AND bm.weight_kg IS NOT NULL
    AND sl.quality IS NOT NULL
    AND (sqlc.narg('from')::DATE IS NULL OR bm.recorded_at::DATE >= sqlc.narg('from')::DATE)
    AND (sqlc.narg('to')::DATE IS NULL OR bm.recorded_at::DATE <= sqlc.narg('to')::DATE)
ORDER BY date DESC
LIMIT 365;

-- name: GetCommuteStepStats :many
-- 依通勤模式統計步數（平均 / 最大 / 最小 / 筆數）
SELECT
    commute_mode,
    COUNT(*)::INT       AS count,
    AVG(steps)::FLOAT8  AS avg_steps,
    MAX(steps)          AS max_steps,
    MIN(steps)          AS min_steps
FROM daily_activities
WHERE user_id = sqlc.arg('user_id')
    AND commute_mode IS NOT NULL
    AND steps IS NOT NULL
    AND (sqlc.narg('from')::DATE IS NULL OR activity_date >= sqlc.narg('from')::DATE)
    AND (sqlc.narg('to')::DATE IS NULL OR activity_date <= sqlc.narg('to')::DATE)
GROUP BY commute_mode
ORDER BY avg_steps DESC
LIMIT 20;
```

### Step 1.5：確認時區一致性

執行前先確認 `body_metrics.recorded_at` 和 `sleep_logs.wake_at` 的時區儲存方式：

```sql
-- 檢查欄位型別（應為 TIMESTAMPTZ 而非 TIMESTAMP）
SELECT column_name, data_type
FROM information_schema.columns
WHERE table_name IN ('body_metrics', 'sleep_logs')
  AND column_name IN ('recorded_at', 'wake_at', 'sleep_at');
```

- 若兩者都是 `TIMESTAMPTZ`（帶時區），`::DATE` 轉換會用資料庫 session timezone，需確保 session timezone 設為使用者本地時區（台灣為 `Asia/Taipei`），否則接近午夜的記錄可能 JOIN 不到。
- 最安全的做法：在 SQL 加 `AT TIME ZONE 'Asia/Taipei'` 再轉 `DATE`：

```sql
-- 若需要修正，將 JOIN 條件改為：
ON bm.recorded_at AT TIME ZONE 'Asia/Taipei' = sl.wake_at AT TIME ZONE 'Asia/Taipei'
```

> **注意：** 若時區已在應用層處理（所有時間都已轉為本地時間儲存），則不需調整。執行前檢查一筆資料確認。

### Step 2：執行 sqlc generate

```bash
cd backend && sqlc generate
```

Expected：`backend/db/sqlc/analytics.sql.go` 被建立，含 `GetWeightSleepPoints`、`GetCommuteStepStats` 函式及對應的 Row struct。

### Step 3：Commit

```bash
git add backend/db/queries/analytics.sql backend/db/sqlc/analytics.sql.go
git commit -m "feat: add analytics SQL queries for weight-sleep correlation and commute stats"
```

---

## Task 2：Pearson 相關係數純函式

**Files:**
- Create: `backend/internal/analytics/pearson.go`
- Test: `backend/internal/analytics/pearson_test.go`

### Step 1：寫失敗測試

```go
// backend/internal/analytics/pearson_test.go
package analytics_test

import (
	"math"
	"testing"

	"health-tracking/backend/internal/analytics"
)

func TestPearsonCorrelation(t *testing.T) {
	tests := []struct {
		name    string
		xs, ys  []float64
		want    float64
		wantNil bool
	}{
		{
			name: "perfect positive correlation",
			xs:   []float64{1, 2, 3, 4, 5},
			ys:   []float64{2, 4, 6, 8, 10},
			want: 1.0,
		},
		{
			name: "perfect negative correlation",
			xs:   []float64{1, 2, 3, 4, 5},
			ys:   []float64{10, 8, 6, 4, 2},
			want: -1.0,
		},
		{
			name: "no correlation (constant y)",
			xs:   []float64{1, 2, 3},
			ys:   []float64{5, 5, 5},
			want: 0.0,
		},
		{
			name:    "insufficient data (< 2 points)",
			xs:      []float64{1},
			ys:      []float64{2},
			wantNil: true,
		},
		{
			name:    "empty slices",
			xs:      []float64{},
			ys:      []float64{},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := analytics.PearsonCorrelation(tt.xs, tt.ys)
			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %v", *got)
				}
				return
			}
			if got == nil {
				t.Fatalf("expected %v, got nil", tt.want)
			}
			if math.Abs(*got-tt.want) > 1e-9 {
				t.Errorf("expected %v, got %v", tt.want, *got)
			}
		})
	}
}
```

### Step 2：執行測試確認失敗

```bash
cd backend && go test ./internal/analytics/...
```

Expected: `FAIL` — `package analytics not found` 或 `cannot find package`

### Step 3：實作 PearsonCorrelation

```go
// backend/internal/analytics/pearson.go
package analytics

import "math"

// PearsonCorrelation computes the Pearson correlation coefficient between xs and ys.
// Returns nil if there are fewer than 2 data points or if the standard deviation of
// either slice is zero (e.g. all values are identical).
func PearsonCorrelation(xs, ys []float64) *float64 {
	n := len(xs)
	if n != len(ys) || n < 2 {
		return nil
	}

	var sumX, sumY, sumXY, sumX2, sumY2 float64
	fn := float64(n)
	for i := 0; i < n; i++ {
		sumX += xs[i]
		sumY += ys[i]
		sumXY += xs[i] * ys[i]
		sumX2 += xs[i] * xs[i]
		sumY2 += ys[i] * ys[i]
	}

	num := fn*sumXY - sumX*sumY
	den := math.Sqrt((fn*sumX2 - sumX*sumX) * (fn*sumY2 - sumY*sumY))
	if den == 0 {
		zero := 0.0
		return &zero
	}
	r := num / den
	return &r
}
```

### Step 4：執行測試確認通過

```bash
cd backend && go test ./internal/analytics/... -v
```

Expected: 全部 PASS

### Step 5：Commit

```bash
git add backend/internal/analytics/
git commit -m "feat: add Pearson correlation pure function with tests"
```

---

## Task 3：Analytics Handler

**Files:**
- Create: `backend/internal/handler/helpers.go`
- Test: `backend/internal/handler/helpers_test.go`
- Create: `backend/internal/handler/analytics.go`
- Test: `backend/internal/handler/analytics_test.go`

### Step 1：寫失敗測試

```go
// backend/internal/handler/analytics_test.go
package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlcdb "health-tracking/backend/db/sqlc"

	"github.com/google/uuid"
)

// mockAnalyticsStore implements AnalyticsStore
type mockAnalyticsStore struct {
	weightSleepFn   func(context.Context, *sqlcdb.GetWeightSleepPointsParams) ([]sqlcdb.GetWeightSleepPointsRow, error)
	commuteStatsFn  func(context.Context, *sqlcdb.GetCommuteStepStatsParams) ([]sqlcdb.GetCommuteStepStatsRow, error)
}

func (m *mockAnalyticsStore) GetWeightSleepPoints(ctx context.Context, arg *sqlcdb.GetWeightSleepPointsParams) ([]sqlcdb.GetWeightSleepPointsRow, error) {
	return m.weightSleepFn(ctx, arg)
}
func (m *mockAnalyticsStore) GetCommuteStepStats(ctx context.Context, arg *sqlcdb.GetCommuteStepStatsParams) ([]sqlcdb.GetCommuteStepStatsRow, error) {
	return m.commuteStatsFn(ctx, arg)
}

func TestGetWeightSleepCorrelation(t *testing.T) {
	tests := []struct {
		name            string
		rows            []sqlcdb.GetWeightSleepPointsRow
		storeErr        error
		wantStatus      int
		wantSampleSize  int
	}{
		{
			name: "returns correlation with data and correct sample_size",
			rows: []sqlcdb.GetWeightSleepPointsRow{
				{Date: time.Now(), WeightKg: sql.NullFloat64{Float64: 72.5, Valid: true}, Quality: sql.NullInt16{Int16: 3, Valid: true}},
				{Date: time.Now().AddDate(0, 0, -1), WeightKg: sql.NullFloat64{Float64: 73.0, Valid: true}, Quality: sql.NullInt16{Int16: 2, Valid: true}},
			},
			wantStatus:      http.StatusOK,
			wantSampleSize:  2,
		},
		{
			name:       "returns empty points when no data",
			rows:       []sqlcdb.GetWeightSleepPointsRow{},
			wantStatus: http.StatusOK,
		},
		{
			name:       "returns 500 on store error",
			storeErr:   sql.ErrConnDone,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockAnalyticsStore{
				weightSleepFn: func(_ context.Context, _ *sqlcdb.GetWeightSleepPointsParams) ([]sqlcdb.GetWeightSleepPointsRow, error) {
					return tt.rows, tt.storeErr
				},
				commuteStatsFn: func(_ context.Context, _ *sqlcdb.GetCommuteStepStatsParams) ([]sqlcdb.GetCommuteStepStatsRow, error) {
					return nil, nil
				},
			}
			r := newTestRouter(GetWeightSleepCorrelation(store), http.MethodGet, "/v1/analytics/weight-sleep")
			req := httptest.NewRequest(http.MethodGet, "/v1/analytics/weight-sleep", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected %d, got %d — body: %s", tt.wantStatus, w.Code, w.Body.String())
			}
			if tt.wantSampleSize > 0 {
				var body map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
					t.Fatalf("invalid JSON: %v", err)
				}
				data := body["data"].(map[string]interface{})
				if int(data["sample_size"].(float64)) != tt.wantSampleSize {
					t.Errorf("expected sample_size %d, got %v", tt.wantSampleSize, data["sample_size"])
				}
			}
		})
	}
}

func TestGetCommuteStepStats(t *testing.T) {
	tests := []struct {
		name       string
		rows       []sqlcdb.GetCommuteStepStatsRow
		storeErr   error
		wantStatus int
	}{
		{
			name: "returns stats",
			rows: []sqlcdb.GetCommuteStepStatsRow{
				{
					CommuteMode: sqlcdb.NullCommuteMode{CommuteMode: sqlcdb.CommuteModeTrain, Valid: true},
					Count:       sql.NullInt32{Int32: 10, Valid: true},
					AvgSteps:    sql.NullFloat64{Float64: 8500, Valid: true},
					MaxSteps:    sql.NullInt32{Int32: 12000, Valid: true},
					MinSteps:    sql.NullInt32{Int32: 5000, Valid: true},
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "db error returns 500",
			storeErr:   sql.ErrConnDone,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockAnalyticsStore{
				weightSleepFn: func(_ context.Context, _ *sqlcdb.GetWeightSleepPointsParams) ([]sqlcdb.GetWeightSleepPointsRow, error) {
					return nil, nil
				},
				commuteStatsFn: func(_ context.Context, _ *sqlcdb.GetCommuteStepStatsParams) ([]sqlcdb.GetCommuteStepStatsRow, error) {
					return tt.rows, tt.storeErr
				},
			}
			r := newTestRouter(GetCommuteStepStats(store), http.MethodGet, "/v1/analytics/commute-steps")
			req := httptest.NewRequest(http.MethodGet, "/v1/analytics/commute-steps", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected %d, got %d — body: %s", tt.wantStatus, w.Code, w.Body.String())
			}
			if tt.wantStatus == http.StatusOK {
				var body map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
					t.Fatalf("invalid JSON: %v", err)
				}
				if body["data"] == nil {
					t.Error("expected data field in response")
				}
			}
		})
	}
}
```

### Step 2：執行測試確認失敗

```bash
cd backend && go test ./internal/handler/... -run TestGetWeightSleep -v
```

Expected: `FAIL` — `GetWeightSleepCorrelation undefined`

### Step 3a：建立 helpers.go（含單元測試）

```go
// backend/internal/handler/helpers.go
package handler

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// parseDateRange extracts optional "from" and "to" query params (YYYY-MM-DD).
// Returns an error response via gin and false if parsing fails or to < from.
func parseDateRange(c *gin.Context) (from, to sql.NullTime, ok bool) {
	var fromTime, toTime time.Time
	if s := c.Query("from"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", "invalid from date, expected YYYY-MM-DD"))
			return from, to, false
		}
		fromTime = t
		from = sql.NullTime{Time: t, Valid: true}
	}
	if s := c.Query("to"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", "invalid to date, expected YYYY-MM-DD"))
			return from, to, false
		}
		toTime = t
		to = sql.NullTime{Time: t, Valid: true}
	}
	if !fromTime.IsZero() && !toTime.IsZero() && toTime.Before(fromTime) {
		c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", "to must not be earlier than from"))
		return from, to, false
	}
	return from, to, true
}
```

```go
// backend/internal/handler/helpers_test.go
package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParseDateRange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name    string
		query   string
		wantOK  bool
		wantStatus int
	}{
		{"both empty", "", true, 0},
		{"valid from and to", "from=2026-01-01&to=2026-03-01", true, 0},
		{"invalid from format", "from=01-01-2026", false, http.StatusBadRequest},
		{"invalid to format", "to=not-a-date", false, http.StatusBadRequest},
		{"to before from", "from=2026-03-01&to=2026-01-01", false, http.StatusBadRequest},
		{"only from", "from=2026-01-01", true, 0},
		{"only to", "to=2026-03-01", true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest(http.MethodGet, "/?"+tt.query, nil)
			_, _, ok := parseDateRange(c)
			if ok != tt.wantOK {
				t.Errorf("expected ok=%v, got ok=%v — body: %s", tt.wantOK, ok, w.Body.String())
			}
			if !tt.wantOK && w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}
```

### Step 3b：實作 Analytics Handler

```go
// backend/internal/handler/analytics.go
package handler

import (
	"context"
	"database/sql"
	"math"
	"net/http"

	sqlcdb "health-tracking/backend/db/sqlc"
	"health-tracking/backend/internal/analytics"
	"health-tracking/backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AnalyticsStore defines the DB queries used by analytics handlers.
// parseDateRange is defined in helpers.go (same package).
type AnalyticsStore interface {
	GetWeightSleepPoints(ctx context.Context, arg *sqlcdb.GetWeightSleepPointsParams) ([]sqlcdb.GetWeightSleepPointsRow, error)
	GetCommuteStepStats(ctx context.Context, arg *sqlcdb.GetCommuteStepStatsParams) ([]sqlcdb.GetCommuteStepStatsRow, error)
}

// --- Response types ---

type WeightSleepPoint struct {
	Date    string  `json:"date"`
	Weight  float64 `json:"weight_kg"`
	Quality int16   `json:"quality"`
}

type WeightSleepCorrelationResponse struct {
	Correlation *float64           `json:"correlation"`
	SampleSize  int                `json:"sample_size"`
	Points      []WeightSleepPoint `json:"points"`
}

type CommuteStepStatResponse struct {
	CommuteMode string   `json:"commute_mode"`
	Count       int32    `json:"count"`
	AvgSteps    float64  `json:"avg_steps"`
	MaxSteps    *int32   `json:"max_steps"`
	MinSteps    *int32   `json:"min_steps"`
}

// --- Handlers ---

func GetWeightSleepCorrelation(store AnalyticsStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		from, to, ok := parseDateRange(c)
		if !ok {
			return
		}
		userID := c.MustGet(middleware.UserIDKey).(uuid.UUID)
		ctx, cancel := withTimeout(c)
		defer cancel()

		rows, err := store.GetWeightSleepPoints(ctx, &sqlcdb.GetWeightSleepPointsParams{
			UserID: userID,
			From:   from,
			To:     to,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResponse("INTERNAL_ERROR", "failed to fetch analytics data"))
			return
		}

		points := make([]WeightSleepPoint, 0, len(rows))
		xs := make([]float64, 0, len(rows))
		ys := make([]float64, 0, len(rows))
		for _, r := range rows {
			if !r.WeightKg.Valid || !r.Quality.Valid {
				continue
			}
			points = append(points, WeightSleepPoint{
				Date:    r.Date.Format("2006-01-02"),
				Weight:  r.WeightKg.Float64,
				Quality: r.Quality.Int16,
			})
			xs = append(xs, r.WeightKg.Float64)
			ys = append(ys, float64(r.Quality.Int16))
		}

		c.JSON(http.StatusOK, gin.H{"data": WeightSleepCorrelationResponse{
			Correlation: analytics.PearsonCorrelation(xs, ys),
			SampleSize:  len(points),
			Points:      points,
		}})
	}
}

func GetCommuteStepStats(store AnalyticsStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		from, to, ok := parseDateRange(c)
		if !ok {
			return
		}
		userID := c.MustGet(middleware.UserIDKey).(uuid.UUID)
		ctx, cancel := withTimeout(c)
		defer cancel()

		rows, err := store.GetCommuteStepStats(ctx, &sqlcdb.GetCommuteStepStatsParams{
			UserID: userID,
			From:   from,
			To:     to,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResponse("INTERNAL_ERROR", "failed to fetch commute stats"))
			return
		}

		stats := make([]CommuteStepStatResponse, 0, len(rows))
		for _, r := range rows {
			if !r.CommuteMode.Valid {
				continue
			}
			s := CommuteStepStatResponse{
				CommuteMode: string(r.CommuteMode.CommuteMode),
				Count:       0,
				AvgSteps:    0,
			}
			if r.Count.Valid {
				s.Count = r.Count.Int32
			}
			if r.AvgSteps.Valid {
				s.AvgSteps = r.AvgSteps.Float64
				s.AvgSteps = math.Round(r.AvgSteps.Float64*10) / 10
			}
			if r.MaxSteps.Valid {
				v := r.MaxSteps.Int32
				s.MaxSteps = &v
			}
			if r.MinSteps.Valid {
				v := r.MinSteps.Int32
				s.MinSteps = &v
			}
			stats = append(stats, s)
		}

		c.JSON(http.StatusOK, gin.H{"data": stats})
	}
}
```

### Step 4：執行測試確認通過

```bash
cd backend && go test ./internal/handler/... -run "TestGetWeightSleep|TestGetCommute" -v
```

Expected: 全部 PASS

### Step 5：Commit

```bash
git add backend/internal/handler/analytics.go backend/internal/handler/analytics_test.go
git commit -m "feat: add analytics handlers for weight-sleep correlation and commute stats"
```

---

## Task 4：CSV Export Handler

**Files:**
- Create: `backend/internal/handler/export.go`
- Test: `backend/internal/handler/export_test.go`

### Step 1：寫失敗測試

```go
// backend/internal/handler/export_test.go
package handler

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sqlcdb "health-tracking/backend/db/sqlc"
)

type mockExportStore struct {
	listMetricsFn     func(context.Context, *sqlcdb.ListBodyMetricsParams) ([]sqlcdb.BodyMetric, error)
	listSleepFn       func(context.Context, *sqlcdb.ListSleepLogsParams) ([]sqlcdb.SleepLog, error)
	listActivitiesFn  func(context.Context, *sqlcdb.ListDailyActivitiesParams) ([]sqlcdb.DailyActivity, error)
}

func (m *mockExportStore) ListBodyMetrics(ctx context.Context, arg *sqlcdb.ListBodyMetricsParams) ([]sqlcdb.BodyMetric, error) {
	return m.listMetricsFn(ctx, arg)
}
func (m *mockExportStore) ListSleepLogs(ctx context.Context, arg *sqlcdb.ListSleepLogsParams) ([]sqlcdb.SleepLog, error) {
	return m.listSleepFn(ctx, arg)
}
func (m *mockExportStore) ListDailyActivities(ctx context.Context, arg *sqlcdb.ListDailyActivitiesParams) ([]sqlcdb.DailyActivity, error) {
	return m.listActivitiesFn(ctx, arg)
}

func TestExportCSV(t *testing.T) {
	sampleMetric := sqlcdb.BodyMetric{
		WeightKg:   sql.NullString{String: "72.50", Valid: true},
		RecordedAt: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name          string
		typeParam     string
		metricsErr    error
		sleepErr      error
		activitiesErr error
		wantStatus    int
		wantHeader    string
		wantBodyLine  string
	}{
		{
			name:         "body-metrics CSV",
			typeParam:    "body-metrics",
			wantStatus:   http.StatusOK,
			wantHeader:   "text/csv",
			wantBodyLine: "recorded_at",
		},
		{
			name:         "sleep-logs CSV",
			typeParam:    "sleep-logs",
			wantStatus:   http.StatusOK,
			wantHeader:   "text/csv",
			wantBodyLine: "sleep_at",
		},
		{
			name:         "daily-activities CSV",
			typeParam:    "daily-activities",
			wantStatus:   http.StatusOK,
			wantHeader:   "text/csv",
			wantBodyLine: "activity_date",
		},
		{
			name:       "unknown type returns 400",
			typeParam:  "unknown",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "body-metrics store error returns 500",
			typeParam:  "body-metrics",
			metricsErr: sql.ErrConnDone,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:      "sleep-logs store error returns 500",
			typeParam: "sleep-logs",
			sleepErr:  sql.ErrConnDone,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:          "daily-activities store error returns 500",
			typeParam:     "daily-activities",
			activitiesErr: sql.ErrConnDone,
			wantStatus:    http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockExportStore{
				listMetricsFn: func(_ context.Context, _ *sqlcdb.ListBodyMetricsParams) ([]sqlcdb.BodyMetric, error) {
					if tt.metricsErr != nil {
						return nil, tt.metricsErr
					}
					return []sqlcdb.BodyMetric{sampleMetric}, nil
				},
				listSleepFn: func(_ context.Context, _ *sqlcdb.ListSleepLogsParams) ([]sqlcdb.SleepLog, error) {
					if tt.sleepErr != nil {
						return nil, tt.sleepErr
					}
					return []sqlcdb.SleepLog{}, nil
				},
				listActivitiesFn: func(_ context.Context, _ *sqlcdb.ListDailyActivitiesParams) ([]sqlcdb.DailyActivity, error) {
					if tt.activitiesErr != nil {
						return nil, tt.activitiesErr
					}
					return []sqlcdb.DailyActivity{}, nil
				},
			}
			r := newTestRouter(ExportCSV(store), http.MethodGet, "/v1/export/csv")
			req := httptest.NewRequest(http.MethodGet, "/v1/export/csv?type="+tt.typeParam, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected %d, got %d — body: %s", tt.wantStatus, w.Code, w.Body.String())
			}
			if tt.wantHeader != "" && !strings.Contains(w.Header().Get("Content-Type"), tt.wantHeader) {
				t.Errorf("expected Content-Type %s, got %s", tt.wantHeader, w.Header().Get("Content-Type"))
			}
			if tt.wantBodyLine != "" && !strings.Contains(w.Body.String(), tt.wantBodyLine) {
				t.Errorf("expected body to contain %q, got: %s", tt.wantBodyLine, w.Body.String())
			}
		})
	}
}
```

### Step 2：執行測試確認失敗

```bash
cd backend && go test ./internal/handler/... -run TestExportCSV -v
```

Expected: `FAIL` — `ExportCSV undefined`

### Step 3：實作 Export Handler

```go
// backend/internal/handler/export.go
package handler

import (
	"context"
	"database/sql"
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	sqlcdb "health-tracking/backend/db/sqlc"
	"health-tracking/backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ExportStore defines the list queries needed for CSV export.
type ExportStore interface {
	ListBodyMetrics(ctx context.Context, arg *sqlcdb.ListBodyMetricsParams) ([]sqlcdb.BodyMetric, error)
	ListSleepLogs(ctx context.Context, arg *sqlcdb.ListSleepLogsParams) ([]sqlcdb.SleepLog, error)
	ListDailyActivities(ctx context.Context, arg *sqlcdb.ListDailyActivitiesParams) ([]sqlcdb.DailyActivity, error)
}

func ExportCSV(store ExportStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		dataType := c.Query("type")
		from, to, ok := parseDateRange(c)
		if !ok {
			return
		}
		userID := c.MustGet(middleware.UserIDKey).(uuid.UUID)
		ctx, cancel := withTimeout(c)
		defer cancel()

		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", "attachment; filename=\""+dataType+"-export.csv\"")
		w := csv.NewWriter(c.Writer)

		switch dataType {
		case "body-metrics":
			rows, err := store.ListBodyMetrics(ctx, &sqlcdb.ListBodyMetricsParams{
				UserID: userID,
				From:   from,
				To:     to,
				Limit:  10000,
			})
			if err != nil {
				c.Status(http.StatusInternalServerError)
				return
			}
			_ = w.Write([]string{"recorded_at", "weight_kg", "body_fat_pct", "muscle_pct", "visceral_fat", "note"})
			for _, r := range rows {
				_ = w.Write([]string{
					r.RecordedAt.Format(time.RFC3339),
					nullStringVal(r.WeightKg),
					nullStringVal(r.BodyFatPct),
					nullStringVal(r.MusclePct),
					nullInt16Val(r.VisceralFat),
					nullStringVal(r.Note),
				})
			}

		case "sleep-logs":
			rows, err := store.ListSleepLogs(ctx, &sqlcdb.ListSleepLogsParams{
				UserID:      userID,
				From:        from,
				To:          to,
				Limit:       10000,
				AbnormalOnly: sql.NullBool{},
			})
			if err != nil {
				c.Status(http.StatusInternalServerError)
				return
			}
			_ = w.Write([]string{"sleep_at", "wake_at", "duration_min", "abnormal_wake", "quality", "note"})
			for _, r := range rows {
				_ = w.Write([]string{
					r.SleepAt.Format(time.RFC3339),
					r.WakeAt.Format(time.RFC3339),
					nullInt32Val(r.DurationMin),
					strconv.FormatBool(r.AbnormalWake),
					nullInt16Val(r.Quality),
					nullStringVal(r.Note),
				})
			}

		case "daily-activities":
			rows, err := store.ListDailyActivities(ctx, &sqlcdb.ListDailyActivitiesParams{
				UserID: userID,
				From:   from,
				To:     to,
				Limit:  10000,
			})
			if err != nil {
				c.Status(http.StatusInternalServerError)
				return
			}
			_ = w.Write([]string{"activity_date", "steps", "commute_mode", "commute_minutes", "note"})
			for _, r := range rows {
				mode := ""
				if r.CommuteMode.Valid {
					mode = string(r.CommuteMode.CommuteMode)
				}
				_ = w.Write([]string{
					r.ActivityDate.Format("2006-01-02"),
					nullInt32Val(r.Steps),
					mode,
					nullInt32Val(r.CommuteMinutes),
					nullStringVal(r.Note),
				})
			}

		default:
			c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", "type must be body-metrics, sleep-logs, or daily-activities"))
			return
		}

		w.Flush()
	}
}

func nullStringVal(s sql.NullString) string {
	if !s.Valid {
		return ""
	}
	return s.String
}

func nullInt16Val(n sql.NullInt16) string {
	if !n.Valid {
		return ""
	}
	return strconv.Itoa(int(n.Int16))
}

func nullInt32Val(n sql.NullInt32) string {
	if !n.Valid {
		return ""
	}
	return strconv.Itoa(int(n.Int32))
}
```

> **注意：** `ListSleepLogsParams` 的 `AbnormalOnly` 欄位和 `ListDailyActivitiesParams` 的欄位需與現有 sqlc 生成的 struct 對齊。執行前先確認 `backend/db/sqlc/sleep_logs.sql.go` 和 `daily_activities.sql.go` 中 param struct 的欄位名稱。

### Step 4：執行測試確認通過

```bash
cd backend && go test ./internal/handler/... -run TestExportCSV -v
```

Expected: 全部 PASS

### Step 5：Commit

```bash
git add backend/internal/handler/export.go backend/internal/handler/export_test.go
git commit -m "feat: add CSV export handler for body-metrics, sleep-logs, daily-activities"
```

---

## Task 5：註冊新路由

**Files:**
- Modify: `backend/cmd/api/main.go`

### Step 1：在 `main.go` protected group 加入新路由

在 `protected.DELETE("/daily-activities/:id", ...)` 之後加入：

```go
// Analytics
analyticsGroup := protected.Group("/analytics")
{
    analyticsGroup.GET("/weight-sleep", handler.GetWeightSleepCorrelation(queries))
    analyticsGroup.GET("/commute-steps", handler.GetCommuteStepStats(queries))
}

// Export
protected.GET("/export/csv", handler.ExportCSV(queries))
```

### Step 2：確認編譯通過

```bash
cd backend && go build ./...
```

Expected: 無錯誤

### Step 3：Commit

```bash
git add backend/cmd/api/main.go
git commit -m "feat: register analytics and export routes"
```

---

## Task 6：前端型別定義

**Files:**
- Modify: `frontend/src/lib/types/index.ts`

### Step 1：在 `types/index.ts` 末尾加入新型別

```typescript
// Analytics
export interface WeightSleepPoint {
  date: string;
  weight_kg: number;
  quality: number;
}

export interface WeightSleepCorrelation {
  correlation: number | null;
  sample_size: number;
  points: WeightSleepPoint[];
}

export interface CommuteStepStat {
  commute_mode: string;
  count: number;
  avg_steps: number;
  max_steps: number | null;
  min_steps: number | null;
}
```

### Step 2：Commit

```bash
git add frontend/src/lib/types/index.ts
git commit -m "feat: add analytics types to frontend"
```

---

## Task 7：前端 Analytics API Client

**Files:**
- Create: `frontend/src/lib/api/analytics.ts`

### Step 1：建立 API 函式

```typescript
// frontend/src/lib/api/analytics.ts
import { PUBLIC_API_BASE_URL } from '$env/static/public';
import { api, createApi } from './client';
import type { WeightSleepCorrelation, CommuteStepStat } from '$lib/types';

export async function getWeightSleepCorrelation(
  params?: { from?: string; to?: string },
  fetchFn?: typeof fetch
): Promise<WeightSleepCorrelation> {
  const query = new URLSearchParams();
  if (params?.from) query.set('from', params.from);
  if (params?.to) query.set('to', params.to);
  const qs = query.toString() ? `?${query}` : '';
  const client = fetchFn ? createApi(fetchFn) : api;
  const res = await client.get<{ data: WeightSleepCorrelation }>(`/analytics/weight-sleep${qs}`);
  return res.data;
}

export async function getCommuteStepStats(
  params?: { from?: string; to?: string },
  fetchFn?: typeof fetch
): Promise<CommuteStepStat[]> {
  const query = new URLSearchParams();
  if (params?.from) query.set('from', params.from);
  if (params?.to) query.set('to', params.to);
  const qs = query.toString() ? `?${query}` : '';
  const client = fetchFn ? createApi(fetchFn) : api;
  const res = await client.get<{ data: CommuteStepStat[] }>(`/analytics/commute-steps${qs}`);
  return res.data;
}

export function buildExportUrl(type: 'body-metrics' | 'sleep-logs' | 'daily-activities', params?: { from?: string; to?: string }): string {
  const query = new URLSearchParams({ type });
  if (params?.from) query.set('from', params.from);
  if (params?.to) query.set('to', params.to);
  // 使用 SvelteKit server-side proxy（/export/csv），避免跨域 cookie 問題
  // 直接打 Go API 會因為 cross-origin 而無法帶 cookie，下載會失敗認證
  return `/export/csv?${query}`;
}
```

### Step 2：建立 SvelteKit CSV proxy route

跨網域的 `<a href download>` 不會帶 cookie，需要透過 SvelteKit server-side route 轉發。

```typescript
// frontend/src/routes/export/csv/+server.ts
import type { RequestHandler } from './$types';
import { PUBLIC_API_BASE_URL } from '$env/static/public';

export const GET: RequestHandler = async ({ url, request }) => {
  const params = new URLSearchParams();
  const type = url.searchParams.get('type');
  const from = url.searchParams.get('from');
  const to = url.searchParams.get('to');
  if (type) params.set('type', type);
  if (from) params.set('from', from);
  if (to) params.set('to', to);

  let res: Response;
  try {
    res = await fetch(`${PUBLIC_API_BASE_URL}/v1/export/csv?${params}`, {
      headers: {
        cookie: request.headers.get('cookie') ?? '',
      },
    });
  } catch {
    return new Response('Export service unavailable', { status: 502 });
  }

  if (!res.ok) {
    return new Response(await res.text(), { status: res.status });
  }

  return new Response(res.body, {
    status: res.status,
    headers: {
      'Content-Type': res.headers.get('Content-Type') ?? 'text/csv',
      'Content-Disposition': res.headers.get('Content-Disposition') ?? `attachment; filename="${type}-export.csv"`,
    },
  });
};
```

### Step 3：Commit

```bash
git add frontend/src/lib/api/analytics.ts frontend/src/routes/export/csv/+server.ts
git commit -m "feat: add analytics and export API clients with SvelteKit CSV proxy"
```

---

## Task 8：前端分析頁面

**Files:**
- Create: `frontend/src/routes/analytics/+page.ts`
- Create: `frontend/src/routes/analytics/+page.svelte`
- Modify: `frontend/src/routes/+layout.svelte`

### Step 1：建立 page loader

```typescript
// frontend/src/routes/analytics/+page.ts
import { getWeightSleepCorrelation, getCommuteStepStats } from '$lib/api/analytics';

export async function load({ fetch }) {
  const to = new Date().toISOString().slice(0, 10);
  const from = new Date(Date.now() - 90 * 24 * 60 * 60 * 1000).toISOString().slice(0, 10);

  // 用 allSettled 讓兩個端點互相獨立：任一失敗不影響另一個區塊
  const [correlationResult, commuteResult] = await Promise.allSettled([
    getWeightSleepCorrelation({ from, to }, fetch),
    getCommuteStepStats({ from, to }, fetch),
  ]);

  return {
    correlation: correlationResult.status === 'fulfilled' ? correlationResult.value : null,
    commuteStats: commuteResult.status === 'fulfilled' ? commuteResult.value : [],
    from,
    to,
  };
}
```

### Step 2：建立分析頁面

```svelte
<!-- frontend/src/routes/analytics/+page.svelte -->
<script lang="ts">
  import { browser } from '$app/environment';
  import { ScatterChart } from 'layerchart';
  import type { PageData } from './$types';
  import { buildExportUrl } from '$lib/api/analytics';

  let { data }: { data: PageData } = $props();

  const correlationLabel = $derived(() => {
    if (data.correlation === null) return '—';
    const r = data.correlation.correlation;
    if (r === null) return '資料不足';
    const abs = Math.abs(r);
    const direction = r > 0 ? '正相關' : '負相關';
    const strength = abs >= 0.7 ? '強' : abs >= 0.4 ? '中' : '弱';
    return `r = ${r.toFixed(3)}（${strength}${direction}）`;
  });

  const scatterPoints = $derived(
    data.correlation?.points.map(p => ({ x: p.quality, y: p.weight_kg, date: p.date })) ?? []
  );

  // Bar chart data：各通勤模式
  const commuteModeLabels: Record<string, string> = {
    train: '捷運/火車',
    scooter: '機車',
    walk: '步行',
    other: '其他',
  };
</script>

<div class="space-y-10">
  <header>
    <h1 class="text-2xl font-black tracking-widest text-white uppercase">Analytics</h1>
    <p class="text-gray-400 text-sm mt-1">近 90 天資料分析（{data.from} — {data.to}）</p>
  </header>

  <!-- 體重 × 睡眠品質 -->
  <section class="bg-[#16213e] rounded-xl p-6">
    <div class="flex items-center justify-between mb-4">
      <h2 class="text-white font-bold tracking-wide">體重 × 睡眠品質相關性</h2>
      <span class="text-sm text-gray-300 bg-[#0f3460] px-3 py-1 rounded-full">
        {correlationLabel()} · {data.correlation?.sample_size ?? 0} 筆
      </span>
    </div>

    {#if data.correlation === null}
      <p class="text-gray-500 text-sm py-10 text-center">載入失敗，請稍後再試</p>
    {:else if data.correlation.sample_size >= 2}
      {#if browser}
        <div class="h-64">
          <ScatterChart
            data={scatterPoints}
            x="quality"
            y="weight_kg"
            xDomain={[0.5, 5.5]}
            props={{
              points: { r: 5, fill: '#0EA5E9', fillOpacity: 0.7 },
              xAxis: { label: '睡眠品質（1–5）' },
            }}
          />
        </div>
      {/if}
    {:else}
      <p class="text-gray-500 text-sm py-10 text-center">
        需要至少 2 筆體重與睡眠品質同日的資料
      </p>
    {/if}
  </section>

  <!-- 通勤模式 × 步數 -->
  <section class="bg-[#16213e] rounded-xl p-6">
    <h2 class="text-white font-bold tracking-wide mb-4">通勤模式 × 平均步數</h2>

    {#if data.commuteStats.length > 0}
      <div class="space-y-3">
        {#each data.commuteStats as stat}
          {@const maxAvg = Math.max(...data.commuteStats.map(s => s.avg_steps))}
          <div class="flex items-center gap-3">
            <span class="text-gray-300 text-sm w-20 shrink-0">
              {commuteModeLabels[stat.commute_mode] ?? stat.commute_mode}
            </span>
            <div class="flex-1 bg-[#0f3460] rounded-full h-5 overflow-hidden">
              <div
                class="h-full bg-[#0EA5E9] rounded-full transition-all"
                style="width: {(stat.avg_steps / maxAvg) * 100}%"
              ></div>
            </div>
            <span class="text-white text-sm font-bold w-20 text-right">
              {Math.round(stat.avg_steps).toLocaleString()} 步
            </span>
            <span class="text-gray-500 text-xs w-12 text-right">
              ({stat.count} 筆)
            </span>
          </div>
        {/each}
      </div>
    {:else}
      <p class="text-gray-500 text-sm py-6 text-center">尚無通勤資料</p>
    {/if}
  </section>

  <!-- 匯出區塊 -->
  <section class="bg-[#16213e] rounded-xl p-6">
    <h2 class="text-white font-bold tracking-wide mb-4">匯出資料</h2>
    <div class="flex flex-wrap gap-3">
      <a
        href={buildExportUrl('body-metrics')}
        class="px-4 py-2 bg-[#0EA5E9] text-white text-sm font-bold rounded hover:opacity-80 transition-opacity"
        download
      >
        體位數據 CSV
      </a>
      <a
        href={buildExportUrl('sleep-logs')}
        class="px-4 py-2 bg-[#0EA5E9] text-white text-sm font-bold rounded hover:opacity-80 transition-opacity"
        download
      >
        睡眠紀錄 CSV
      </a>
      <a
        href={buildExportUrl('daily-activities')}
        class="px-4 py-2 bg-[#0EA5E9] text-white text-sm font-bold rounded hover:opacity-80 transition-opacity"
        download
      >
        每日活動 CSV
      </a>
      <button
        onclick={() => window.print()}
        class="px-4 py-2 bg-gray-600 text-white text-sm font-bold rounded hover:opacity-80 transition-opacity"
      >
        列印 / PDF
      </button>
    </div>
  </section>
</div>
```

### Step 3：加入列印樣式

在 `+page.svelte` 尾端加入 `<style>` 區塊：

```svelte
<style>
  @media print {
    :global(nav)               { display: none !important; }
    :global(body)              { background: white !important; }
    :global(.bg-\[#1a1a2e\])  { background: white !important; }
    :global(.bg-\[#16213e\])  { background: #f5f5f5 !important; border: 1px solid #ddd; }
    :global(.text-white)       { color: #111 !important; }
    :global(.text-gray-300),
    :global(.text-gray-400),
    :global(.text-gray-500)    { color: #555 !important; }
    /* 隱藏匯出按鈕（列印時不需要） */
    section:last-of-type       { display: none; }
  }
</style>
```

### Step 4：在 navbar 加入「分析」連結

在 `frontend/src/routes/+layout.svelte` 的 `navItems` 陣列加入：

```typescript
{ label: '分析', href: resolve('/analytics'), enabled: true },
```

加在 `{ label: '活動', ... }` 之後。

### Step 5：確認前端編譯

```bash
cd frontend && npm run check
```

Expected: 無 TypeScript 錯誤

### Step 6：Commit

```bash
git add frontend/src/routes/analytics/ frontend/src/routes/export/ frontend/src/routes/+layout.svelte
git commit -m "feat: add analytics page with correlation chart, commute stats, CSV export, and print styles"
```

---

## Task 9：整合驗收

### Step 1：執行全部後端測試

```bash
cd backend && go test ./... -v
```

Expected: 全部 PASS，覆蓋率 ≥ 80%

### Step 2：啟動本地環境手動驗收

```bash
# Terminal 1
cd backend && APP_ENV=local go run ./cmd/api

# Terminal 2
cd frontend && npm run dev
```

驗收清單：
- [ ] 訪問 `http://localhost:5173/analytics`，頁面正常載入
- [ ] 若有 90 天內同日的體重 + 睡眠品質資料，顯示散點圖與相關係數
- [ ] 通勤模式橫條圖顯示各模式平均步數
- [ ] 點擊「體位數據 CSV」觸發下載，檔案內容含 header 行
- [ ] 點擊「列印 / PDF」開啟瀏覽器列印視窗
- [ ] Navbar 顯示「分析」連結，點擊正確導覽

### Step 3：最終 Commit

```bash
git add .
git commit -m "feat: milestone 5 — advanced analytics and CSV/PDF export"
```

---

## 附錄：已知注意事項

1. **`ListSleepLogsParams` 欄位確認**：`ExportCSV` handler 中的 `AbnormalOnly sql.NullBool{}` 需與現有 sqlc 生成的 struct 完全一致，執行前先查看 `backend/db/sqlc/sleep_logs.sql.go`。

2. **CSV export 透過 SvelteKit proxy**：`buildExportUrl` 回傳的是 `/export/csv?...`（SvelteKit server route），由 server 帶 cookie 轉發給 Go API。直接打 `PUBLIC_API_BASE_URL` 會因跨域無法帶 cookie。

3. **layerchart `ScatterChart` component**：使用高階 API `ScatterChart`，需確認已從 `layerchart` 正確 export。若遇到問題，查閱當前版本文件。

4. **時區確認**：Task 1.5 有時區一致性的確認步驟，執行前必看。

## GSTACK REVIEW REPORT

| Review | Trigger | Why | Runs | Status | Findings |
|--------|---------|-----|------|--------|----------|
| CEO Review | `/plan-ceo-review` | Scope & strategy | 0 | — | — |
| Outside Voice | `/plan-eng-review` | Independent 2nd opinion | 1 | issues_found | 11 findings, all resolved |
| Eng Review | `/plan-eng-review` | Architecture & tests (required) | 1 | CLEAR | 11 issues, 0 critical gaps |
| Design Review | `/plan-design-review` | UI/UX gaps | 0 | — | — |

**VERDICT:** ENG CLEARED — ready to implement.
