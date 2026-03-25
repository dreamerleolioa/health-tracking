package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlcdb "health-tracking/backend/db/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

// --- Mock store ---

type mockDailyActivityStore struct {
	createFn func(context.Context, *sqlcdb.CreateDailyActivityParams) (sqlcdb.DailyActivity, error)
	getFn    func(context.Context, uuid.UUID) (sqlcdb.DailyActivity, error)
	listFn   func(context.Context, *sqlcdb.ListDailyActivitiesParams) ([]sqlcdb.DailyActivity, error)
	updateFn func(context.Context, *sqlcdb.UpdateDailyActivityParams) (sqlcdb.DailyActivity, error)
	deleteFn func(context.Context, uuid.UUID) error
}

func (m *mockDailyActivityStore) CreateDailyActivity(ctx context.Context, arg *sqlcdb.CreateDailyActivityParams) (sqlcdb.DailyActivity, error) {
	return m.createFn(ctx, arg)
}
func (m *mockDailyActivityStore) GetDailyActivity(ctx context.Context, id uuid.UUID) (sqlcdb.DailyActivity, error) {
	return m.getFn(ctx, id)
}
func (m *mockDailyActivityStore) ListDailyActivities(ctx context.Context, arg *sqlcdb.ListDailyActivitiesParams) ([]sqlcdb.DailyActivity, error) {
	return m.listFn(ctx, arg)
}
func (m *mockDailyActivityStore) UpdateDailyActivity(ctx context.Context, arg *sqlcdb.UpdateDailyActivityParams) (sqlcdb.DailyActivity, error) {
	return m.updateFn(ctx, arg)
}
func (m *mockDailyActivityStore) DeleteDailyActivity(ctx context.Context, id uuid.UUID) error {
	return m.deleteFn(ctx, id)
}

func sampleActivity() sqlcdb.DailyActivity {
	now := time.Now()
	return sqlcdb.DailyActivity{
		ID:           uuid.New(),
		ActivityDate: now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// uniqueViolationError simulates pgx unique violation error.
func uniqueViolationError() error {
	return &pgconn.PgError{Code: "23505", Message: "duplicate key value"}
}

// --- Tests ---

func TestCreateDailyActivity(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		setupStore func(*mockDailyActivityStore)
		wantStatus int
	}{
		{
			name: "success 201",
			body: map[string]any{
				"activity_date": "2026-03-24",
				"steps":         8500,
				"commute_mode":  "train",
			},
			setupStore: func(ms *mockDailyActivityStore) {
				ms.createFn = func(_ context.Context, _ *sqlcdb.CreateDailyActivityParams) (sqlcdb.DailyActivity, error) {
					return sampleActivity(), nil
				}
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "missing activity_date returns 400",
			body:       map[string]any{"steps": 5000},
			setupStore: func(_ *mockDailyActivityStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "duplicate date returns 409",
			body: map[string]any{"activity_date": "2026-03-24"},
			setupStore: func(ms *mockDailyActivityStore) {
				ms.createFn = func(_ context.Context, _ *sqlcdb.CreateDailyActivityParams) (sqlcdb.DailyActivity, error) {
					return sqlcdb.DailyActivity{}, uniqueViolationError()
				}
			},
			wantStatus: http.StatusConflict,
		},
		{
			name: "invalid commute_mode returns 400",
			body: map[string]any{
				"activity_date": "2026-03-24",
				"commute_mode":  "bicycle",
			},
			setupStore: func(_ *mockDailyActivityStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid date format returns 400",
			body:       map[string]any{"activity_date": "not-a-date"},
			setupStore: func(_ *mockDailyActivityStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "store returns 500",
			body: map[string]any{"activity_date": "2026-03-24"},
			setupStore: func(ms *mockDailyActivityStore) {
				ms.createFn = func(_ context.Context, _ *sqlcdb.CreateDailyActivityParams) (sqlcdb.DailyActivity, error) {
					return sqlcdb.DailyActivity{}, errors.New("db error")
				}
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &mockDailyActivityStore{}
			tt.setupStore(ms)

			b, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/v1/daily-activities", bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			newTestRouter(CreateDailyActivity(ms), http.MethodPost, "/v1/daily-activities").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got %d, want %d; body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestListDailyActivities(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		setupStore func(*mockDailyActivityStore)
		wantStatus int
	}{
		{
			name:  "no params returns list",
			query: "",
			setupStore: func(ms *mockDailyActivityStore) {
				ms.listFn = func(_ context.Context, _ *sqlcdb.ListDailyActivitiesParams) ([]sqlcdb.DailyActivity, error) {
					return []sqlcdb.DailyActivity{sampleActivity()}, nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "to earlier than from returns 400",
			query:      "?from=2026-03-31&to=2026-01-01",
			setupStore: func(_ *mockDailyActivityStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid from date format returns 400",
			query:      "?from=bad",
			setupStore: func(_ *mockDailyActivityStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:  "store returns 500",
			query: "",
			setupStore: func(ms *mockDailyActivityStore) {
				ms.listFn = func(_ context.Context, _ *sqlcdb.ListDailyActivitiesParams) ([]sqlcdb.DailyActivity, error) {
					return nil, errors.New("db error")
				}
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &mockDailyActivityStore{}
			tt.setupStore(ms)

			req := httptest.NewRequest(http.MethodGet, "/v1/daily-activities"+tt.query, nil)
			w := httptest.NewRecorder()

			newTestRouter(ListDailyActivities(ms), http.MethodGet, "/v1/daily-activities").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got %d, want %d; body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestUpdateDailyActivity(t *testing.T) {
	validID := uuid.New().String()

	tests := []struct {
		name       string
		id         string
		body       any
		setupStore func(*mockDailyActivityStore)
		wantStatus int
	}{
		{
			name: "success 200",
			id:   validID,
			body: map[string]any{"steps": 10000},
			setupStore: func(ms *mockDailyActivityStore) {
				ms.updateFn = func(_ context.Context, _ *sqlcdb.UpdateDailyActivityParams) (sqlcdb.DailyActivity, error) {
					return sampleActivity(), nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "id not found returns 404",
			id:   validID,
			body: map[string]any{"steps": 10000},
			setupStore: func(ms *mockDailyActivityStore) {
				ms.updateFn = func(_ context.Context, _ *sqlcdb.UpdateDailyActivityParams) (sqlcdb.DailyActivity, error) {
					return sqlcdb.DailyActivity{}, sql.ErrNoRows
				}
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "empty body returns 400",
			id:         validID,
			body:       map[string]any{},
			setupStore: func(_ *mockDailyActivityStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid UUID param returns 400",
			id:         "not-uuid",
			body:       map[string]any{"steps": 10000},
			setupStore: func(_ *mockDailyActivityStore) {},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &mockDailyActivityStore{}
			tt.setupStore(ms)

			b, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/v1/daily-activities/%s", tt.id), bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			newTestRouter(UpdateDailyActivity(ms), http.MethodPatch, "/v1/daily-activities/:id").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got %d, want %d; body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestDeleteDailyActivity(t *testing.T) {
	validID := uuid.New().String()

	tests := []struct {
		name       string
		id         string
		setupStore func(*mockDailyActivityStore)
		wantStatus int
	}{
		{
			name: "success 204",
			id:   validID,
			setupStore: func(ms *mockDailyActivityStore) {
				ms.getFn = func(_ context.Context, _ uuid.UUID) (sqlcdb.DailyActivity, error) {
					return sampleActivity(), nil
				}
				ms.deleteFn = func(_ context.Context, _ uuid.UUID) error { return nil }
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name: "id not found returns 404",
			id:   validID,
			setupStore: func(ms *mockDailyActivityStore) {
				ms.getFn = func(_ context.Context, _ uuid.UUID) (sqlcdb.DailyActivity, error) {
					return sqlcdb.DailyActivity{}, sql.ErrNoRows
				}
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid UUID param returns 400",
			id:         "not-uuid",
			setupStore: func(_ *mockDailyActivityStore) {},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &mockDailyActivityStore{}
			tt.setupStore(ms)

			req := httptest.NewRequest(http.MethodDelete, "/v1/daily-activities/"+tt.id, nil)
			w := httptest.NewRecorder()

			newTestRouter(DeleteDailyActivity(ms), http.MethodDelete, "/v1/daily-activities/:id").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got %d, want %d; body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}
