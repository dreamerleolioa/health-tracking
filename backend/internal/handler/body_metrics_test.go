package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlcdb "health-tracking/backend/db/sqlc"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// --- Mock store ---

type mockStore struct {
	createFn func(context.Context, *sqlcdb.CreateBodyMetricParams) (sqlcdb.BodyMetric, error)
	getFn    func(context.Context, uuid.UUID) (sqlcdb.BodyMetric, error)
	listFn   func(context.Context, *sqlcdb.ListBodyMetricsParams) ([]sqlcdb.BodyMetric, error)
	updateFn func(context.Context, *sqlcdb.UpdateBodyMetricParams) (sqlcdb.BodyMetric, error)
	deleteFn func(context.Context, uuid.UUID) error
}

func (m *mockStore) CreateBodyMetric(ctx context.Context, arg *sqlcdb.CreateBodyMetricParams) (sqlcdb.BodyMetric, error) {
	return m.createFn(ctx, arg)
}
func (m *mockStore) GetBodyMetric(ctx context.Context, id uuid.UUID) (sqlcdb.BodyMetric, error) {
	return m.getFn(ctx, id)
}
func (m *mockStore) ListBodyMetrics(ctx context.Context, arg *sqlcdb.ListBodyMetricsParams) ([]sqlcdb.BodyMetric, error) {
	return m.listFn(ctx, arg)
}
func (m *mockStore) UpdateBodyMetric(ctx context.Context, arg *sqlcdb.UpdateBodyMetricParams) (sqlcdb.BodyMetric, error) {
	return m.updateFn(ctx, arg)
}
func (m *mockStore) DeleteBodyMetric(ctx context.Context, id uuid.UUID) error {
	return m.deleteFn(ctx, id)
}

// --- Helpers ---

func newTestRouter(h gin.HandlerFunc, method, path string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Handle(method, path, h)
	return r
}

func sampleMetric() sqlcdb.BodyMetric {
	now := time.Now()
	return sqlcdb.BodyMetric{
		ID:         uuid.New(),
		WeightKg:   sql.NullString{String: "72.5", Valid: true},
		RecordedAt: now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func mustMarshal(v any) *bytes.Reader {
	b, _ := json.Marshal(v)
	return bytes.NewReader(b)
}

// --- TestCreateBodyMetric ---

func TestCreateBodyMetric(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		setupStore func(*mockStore)
		wantStatus int
	}{
		{
			name: "success 201",
			body: map[string]any{
				"weight_kg":   72.5,
				"recorded_at": "2026-03-24T08:00:00+08:00",
			},
			setupStore: func(ms *mockStore) {
				ms.createFn = func(_ context.Context, _ *sqlcdb.CreateBodyMetricParams) (sqlcdb.BodyMetric, error) {
					return sampleMetric(), nil
				}
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "missing recorded_at returns 400",
			body:       map[string]any{"weight_kg": 72.5},
			setupStore: func(_ *mockStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "weight_kg below min returns 400",
			body: map[string]any{
				"weight_kg":   10.0,
				"recorded_at": "2026-03-24T08:00:00+08:00",
			},
			setupStore: func(_ *mockStore) {},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &mockStore{}
			tt.setupStore(ms)

			req := httptest.NewRequest(http.MethodPost, "/v1/body-metrics", mustMarshal(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			newTestRouter(CreateBodyMetric(ms), http.MethodPost, "/v1/body-metrics").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got %d, want %d; body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

// --- TestListBodyMetrics ---

func TestListBodyMetrics(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		setupStore func(*mockStore)
		wantStatus int
	}{
		{
			name:  "no params returns list",
			query: "",
			setupStore: func(ms *mockStore) {
				ms.listFn = func(_ context.Context, _ *sqlcdb.ListBodyMetricsParams) ([]sqlcdb.BodyMetric, error) {
					return []sqlcdb.BodyMetric{sampleMetric()}, nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:  "with from/to filter",
			query: "?from=2026-01-01&to=2026-03-31",
			setupStore: func(ms *mockStore) {
				ms.listFn = func(_ context.Context, _ *sqlcdb.ListBodyMetricsParams) ([]sqlcdb.BodyMetric, error) {
					return []sqlcdb.BodyMetric{}, nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "to earlier than from returns 400",
			query:      "?from=2026-03-31&to=2026-01-01",
			setupStore: func(_ *mockStore) {},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &mockStore{}
			tt.setupStore(ms)

			req := httptest.NewRequest(http.MethodGet, "/v1/body-metrics"+tt.query, nil)
			w := httptest.NewRecorder()

			newTestRouter(ListBodyMetrics(ms), http.MethodGet, "/v1/body-metrics").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got %d, want %d; body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

// --- TestUpdateBodyMetric ---

func TestUpdateBodyMetric(t *testing.T) {
	validID := uuid.New().String()

	tests := []struct {
		name       string
		id         string
		body       any
		setupStore func(*mockStore)
		wantStatus int
	}{
		{
			name: "success 200",
			id:   validID,
			body: map[string]any{"weight_kg": 75.0},
			setupStore: func(ms *mockStore) {
				ms.updateFn = func(_ context.Context, _ *sqlcdb.UpdateBodyMetricParams) (sqlcdb.BodyMetric, error) {
					return sampleMetric(), nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "id not found returns 404",
			id:   validID,
			body: map[string]any{"weight_kg": 75.0},
			setupStore: func(ms *mockStore) {
				ms.updateFn = func(_ context.Context, _ *sqlcdb.UpdateBodyMetricParams) (sqlcdb.BodyMetric, error) {
					return sqlcdb.BodyMetric{}, sql.ErrNoRows
				}
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "empty body returns 400",
			id:         validID,
			body:       map[string]any{},
			setupStore: func(_ *mockStore) {},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &mockStore{}
			tt.setupStore(ms)

			req := httptest.NewRequest(http.MethodPatch, "/v1/body-metrics/"+tt.id, mustMarshal(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			newTestRouter(UpdateBodyMetric(ms), http.MethodPatch, "/v1/body-metrics/:id").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got %d, want %d; body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

// --- TestDeleteBodyMetric ---

func TestDeleteBodyMetric(t *testing.T) {
	validID := uuid.New().String()

	tests := []struct {
		name       string
		id         string
		setupStore func(*mockStore)
		wantStatus int
	}{
		{
			name: "success 204",
			id:   validID,
			setupStore: func(ms *mockStore) {
				ms.getFn = func(_ context.Context, _ uuid.UUID) (sqlcdb.BodyMetric, error) {
					return sampleMetric(), nil
				}
				ms.deleteFn = func(_ context.Context, _ uuid.UUID) error {
					return nil
				}
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name: "id not found returns 404",
			id:   validID,
			setupStore: func(ms *mockStore) {
				ms.getFn = func(_ context.Context, _ uuid.UUID) (sqlcdb.BodyMetric, error) {
					return sqlcdb.BodyMetric{}, sql.ErrNoRows
				}
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &mockStore{}
			tt.setupStore(ms)

			req := httptest.NewRequest(http.MethodDelete, "/v1/body-metrics/"+tt.id, nil)
			w := httptest.NewRecorder()

			newTestRouter(DeleteBodyMetric(ms), http.MethodDelete, "/v1/body-metrics/:id").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got %d, want %d; body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}
