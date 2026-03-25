package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlcdb "health-tracking/backend/db/sqlc"

	"github.com/google/uuid"
)

// --- Mock store ---

type mockSleepLogStore struct {
	createFn func(context.Context, *sqlcdb.CreateSleepLogParams) (sqlcdb.SleepLog, error)
	getFn    func(context.Context, uuid.UUID) (sqlcdb.SleepLog, error)
	listFn   func(context.Context, *sqlcdb.ListSleepLogsParams) ([]sqlcdb.SleepLog, error)
	updateFn func(context.Context, *sqlcdb.UpdateSleepLogParams) (sqlcdb.SleepLog, error)
	deleteFn func(context.Context, uuid.UUID) error
}

func (m *mockSleepLogStore) CreateSleepLog(ctx context.Context, arg *sqlcdb.CreateSleepLogParams) (sqlcdb.SleepLog, error) {
	return m.createFn(ctx, arg)
}
func (m *mockSleepLogStore) GetSleepLog(ctx context.Context, id uuid.UUID) (sqlcdb.SleepLog, error) {
	return m.getFn(ctx, id)
}
func (m *mockSleepLogStore) ListSleepLogs(ctx context.Context, arg *sqlcdb.ListSleepLogsParams) ([]sqlcdb.SleepLog, error) {
	return m.listFn(ctx, arg)
}
func (m *mockSleepLogStore) UpdateSleepLog(ctx context.Context, arg *sqlcdb.UpdateSleepLogParams) (sqlcdb.SleepLog, error) {
	return m.updateFn(ctx, arg)
}
func (m *mockSleepLogStore) DeleteSleepLog(ctx context.Context, id uuid.UUID) error {
	return m.deleteFn(ctx, id)
}

func sampleSleepLog() sqlcdb.SleepLog {
	now := time.Now()
	return sqlcdb.SleepLog{
		ID:           uuid.New(),
		SleepAt:      now.Add(-8 * time.Hour),
		WakeAt:       now,
		AbnormalWake: false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// --- Tests ---

func TestCreateSleepLog(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		setupStore func(*mockSleepLogStore)
		wantStatus int
	}{
		{
			name: "success 201",
			body: map[string]any{
				"sleep_at": "2026-03-23T23:30:00+08:00",
				"wake_at":  "2026-03-24T07:00:00+08:00",
				"quality":  3,
			},
			setupStore: func(ms *mockSleepLogStore) {
				ms.createFn = func(_ context.Context, _ *sqlcdb.CreateSleepLogParams) (sqlcdb.SleepLog, error) {
					return sampleSleepLog(), nil
				}
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "missing sleep_at returns 400",
			body:       map[string]any{"wake_at": "2026-03-24T07:00:00+08:00"},
			setupStore: func(_ *mockSleepLogStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "wake_at before sleep_at returns 400",
			body: map[string]any{
				"sleep_at": "2026-03-24T07:00:00+08:00",
				"wake_at":  "2026-03-23T23:30:00+08:00",
			},
			setupStore: func(_ *mockSleepLogStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "quality out of range returns 400",
			body: map[string]any{
				"sleep_at": "2026-03-23T23:30:00+08:00",
				"wake_at":  "2026-03-24T07:00:00+08:00",
				"quality":  6,
			},
			setupStore: func(_ *mockSleepLogStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "store returns 500",
			body: map[string]any{
				"sleep_at": "2026-03-23T23:30:00+08:00",
				"wake_at":  "2026-03-24T07:00:00+08:00",
			},
			setupStore: func(ms *mockSleepLogStore) {
				ms.createFn = func(_ context.Context, _ *sqlcdb.CreateSleepLogParams) (sqlcdb.SleepLog, error) {
					return sqlcdb.SleepLog{}, errors.New("db error")
				}
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "invalid JSON body",
			body:       "not-json",
			setupStore: func(_ *mockSleepLogStore) {},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &mockSleepLogStore{}
			tt.setupStore(ms)

			b, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/v1/sleep-logs", bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			newTestRouter(CreateSleepLog(ms), http.MethodPost, "/v1/sleep-logs").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got %d, want %d; body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestListSleepLogs(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		setupStore func(*mockSleepLogStore)
		wantStatus int
	}{
		{
			name:  "no params returns list",
			query: "",
			setupStore: func(ms *mockSleepLogStore) {
				ms.listFn = func(_ context.Context, _ *sqlcdb.ListSleepLogsParams) ([]sqlcdb.SleepLog, error) {
					return []sqlcdb.SleepLog{sampleSleepLog()}, nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:  "abnormal_only=true",
			query: "?abnormal_only=true",
			setupStore: func(ms *mockSleepLogStore) {
				ms.listFn = func(_ context.Context, _ *sqlcdb.ListSleepLogsParams) ([]sqlcdb.SleepLog, error) {
					return []sqlcdb.SleepLog{}, nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "to earlier than from returns 400",
			query:      "?from=2026-03-31&to=2026-01-01",
			setupStore: func(_ *mockSleepLogStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid from date format",
			query:      "?from=bad-date",
			setupStore: func(_ *mockSleepLogStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:  "store returns 500",
			query: "",
			setupStore: func(ms *mockSleepLogStore) {
				ms.listFn = func(_ context.Context, _ *sqlcdb.ListSleepLogsParams) ([]sqlcdb.SleepLog, error) {
					return nil, errors.New("db error")
				}
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &mockSleepLogStore{}
			tt.setupStore(ms)

			req := httptest.NewRequest(http.MethodGet, "/v1/sleep-logs"+tt.query, nil)
			w := httptest.NewRecorder()

			newTestRouter(ListSleepLogs(ms), http.MethodGet, "/v1/sleep-logs").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got %d, want %d; body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestUpdateSleepLog(t *testing.T) {
	validID := uuid.New().String()

	tests := []struct {
		name       string
		id         string
		body       any
		setupStore func(*mockSleepLogStore)
		wantStatus int
	}{
		{
			name: "success 200",
			id:   validID,
			body: map[string]any{"quality": 4},
			setupStore: func(ms *mockSleepLogStore) {
				ms.getFn = func(_ context.Context, _ uuid.UUID) (sqlcdb.SleepLog, error) {
					return sampleSleepLog(), nil
				}
				ms.updateFn = func(_ context.Context, _ *sqlcdb.UpdateSleepLogParams) (sqlcdb.SleepLog, error) {
					return sampleSleepLog(), nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "id not found returns 404",
			id:   validID,
			body: map[string]any{"quality": 4},
			setupStore: func(ms *mockSleepLogStore) {
				ms.getFn = func(_ context.Context, _ uuid.UUID) (sqlcdb.SleepLog, error) {
					return sqlcdb.SleepLog{}, sql.ErrNoRows
				}
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "empty body returns 400",
			id:         validID,
			body:       map[string]any{},
			setupStore: func(_ *mockSleepLogStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid UUID param",
			id:         "not-a-uuid",
			body:       map[string]any{"quality": 4},
			setupStore: func(_ *mockSleepLogStore) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "store GetSleepLog returns 500",
			id:   validID,
			body: map[string]any{"quality": 4},
			setupStore: func(ms *mockSleepLogStore) {
				ms.getFn = func(_ context.Context, _ uuid.UUID) (sqlcdb.SleepLog, error) {
					return sqlcdb.SleepLog{}, errors.New("db error")
				}
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "only wake_at provided before existing sleep_at returns 400",
			id:   validID,
			body: map[string]any{"wake_at": "2026-01-01T00:00:00Z"},
			setupStore: func(ms *mockSleepLogStore) {
				ms.getFn = func(_ context.Context, _ uuid.UUID) (sqlcdb.SleepLog, error) {
					s := sampleSleepLog()
					// Set sleep_at to a time after the patch wake_at
					s.SleepAt = time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
					s.WakeAt = time.Date(2026, 1, 2, 8, 0, 0, 0, time.UTC)
					return s, nil
				}
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &mockSleepLogStore{}
			tt.setupStore(ms)

			b, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPatch, "/v1/sleep-logs/"+tt.id, bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			newTestRouter(UpdateSleepLog(ms), http.MethodPatch, "/v1/sleep-logs/:id").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got %d, want %d; body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestDeleteSleepLog(t *testing.T) {
	validID := uuid.New().String()

	tests := []struct {
		name       string
		id         string
		setupStore func(*mockSleepLogStore)
		wantStatus int
	}{
		{
			name: "success 204",
			id:   validID,
			setupStore: func(ms *mockSleepLogStore) {
				ms.getFn = func(_ context.Context, _ uuid.UUID) (sqlcdb.SleepLog, error) {
					return sampleSleepLog(), nil
				}
				ms.deleteFn = func(_ context.Context, _ uuid.UUID) error { return nil }
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name: "id not found returns 404",
			id:   validID,
			setupStore: func(ms *mockSleepLogStore) {
				ms.getFn = func(_ context.Context, _ uuid.UUID) (sqlcdb.SleepLog, error) {
					return sqlcdb.SleepLog{}, sql.ErrNoRows
				}
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid UUID param",
			id:         "not-a-uuid",
			setupStore: func(_ *mockSleepLogStore) {},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &mockSleepLogStore{}
			tt.setupStore(ms)

			req := httptest.NewRequest(http.MethodDelete, "/v1/sleep-logs/"+tt.id, nil)
			w := httptest.NewRecorder()

			newTestRouter(DeleteSleepLog(ms), http.MethodDelete, "/v1/sleep-logs/:id").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got %d, want %d; body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}
