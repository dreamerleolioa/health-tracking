package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	sqlcdb "health-tracking/backend/db/sqlc"
)

// abnormalWakeTime returns a time that is 03:30 Asia/Taipei (UTC+8), i.e. 19:30 UTC previous day.
func abnormalWakeTime(date time.Time) time.Time {
	loc, _ := time.LoadLocation("Asia/Taipei")
	local := time.Date(date.Year(), date.Month(), date.Day(), 3, 30, 0, 0, loc)
	return local.UTC()
}

// normalWakeTime returns a time that is 07:00 Asia/Taipei.
func normalWakeTime(date time.Time) time.Time {
	loc, _ := time.LoadLocation("Asia/Taipei")
	local := time.Date(date.Year(), date.Month(), date.Day(), 7, 0, 0, 0, loc)
	return local.UTC()
}

func TestCreateAndGetSleepLog(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()

	sleepAt := now.Add(-8 * time.Hour)
	wakeAt := now

	created, err := queries.CreateSleepLog(ctx, &sqlcdb.CreateSleepLogParams{
		UserID:  testUserID,
		SleepAt: sleepAt,
		WakeAt:  wakeAt,
	})
	if err != nil {
		t.Fatalf("CreateSleepLog: %v", err)
	}
	if created.ID.String() == "" {
		t.Fatal("expected non-empty ID")
	}

	fetched, err := queries.GetSleepLog(ctx, &sqlcdb.GetSleepLogParams{ID: created.ID, UserID: testUserID})
	if err != nil {
		t.Fatalf("GetSleepLog: %v", err)
	}
	if fetched.ID != created.ID {
		t.Errorf("ID mismatch: got %v, want %v", fetched.ID, created.ID)
	}
}

func TestSleepLogDurationMinComputed(t *testing.T) {
	ctx := context.Background()
	loc, _ := time.LoadLocation("Asia/Taipei")
	// Sleep 23:00 Taipei, Wake 07:00 Taipei = 480 min
	sleepAt := time.Date(2026, 3, 23, 23, 0, 0, 0, loc).UTC()
	wakeAt := time.Date(2026, 3, 24, 7, 0, 0, 0, loc).UTC()

	created, err := queries.CreateSleepLog(ctx, &sqlcdb.CreateSleepLogParams{
		UserID:  testUserID,
		SleepAt: sleepAt,
		WakeAt:  wakeAt,
	})
	if err != nil {
		t.Fatalf("CreateSleepLog: %v", err)
	}
	if !created.DurationMin.Valid {
		t.Fatal("expected duration_min to be computed")
	}
	if created.DurationMin.Int32 != 480 {
		t.Errorf("duration_min: got %d, want 480", created.DurationMin.Int32)
	}
}

// TestAbnormalWakeTrigger verifies the DB trigger sets abnormal_wake = true
// when wake_at is in 03:00–03:59 Asia/Taipei.
func TestAbnormalWakeTrigger(t *testing.T) {
	ctx := context.Background()
	ref := time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name             string
		wakeAt           time.Time
		wantAbnormalWake bool
	}{
		{
			name:             "wake at 03:30 Taipei → abnormal",
			wakeAt:           abnormalWakeTime(ref),
			wantAbnormalWake: true,
		},
		{
			name:             "wake at 07:00 Taipei → normal",
			wakeAt:           normalWakeTime(ref),
			wantAbnormalWake: false,
		},
		{
			name: "wake at 03:00 Taipei exactly → abnormal",
			wakeAt: func() time.Time {
				loc, _ := time.LoadLocation("Asia/Taipei")
				return time.Date(2026, 3, 24, 3, 0, 0, 0, loc).UTC()
			}(),
			wantAbnormalWake: true,
		},
		{
			name: "wake at 04:00 Taipei → normal (boundary)",
			wakeAt: func() time.Time {
				loc, _ := time.LoadLocation("Asia/Taipei")
				return time.Date(2026, 3, 24, 4, 0, 0, 0, loc).UTC()
			}(),
			wantAbnormalWake: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sleepAt := tt.wakeAt.Add(-6 * time.Hour)
			created, err := queries.CreateSleepLog(ctx, &sqlcdb.CreateSleepLogParams{
				UserID:  testUserID,
				SleepAt: sleepAt,
				WakeAt:  tt.wakeAt,
			})
			if err != nil {
				t.Fatalf("CreateSleepLog: %v", err)
			}
			if created.AbnormalWake != tt.wantAbnormalWake {
				t.Errorf("abnormal_wake: got %v, want %v", created.AbnormalWake, tt.wantAbnormalWake)
			}
		})
	}
}

func TestAbnormalWakeUpdatedByTriggerOnUpdate(t *testing.T) {
	ctx := context.Background()
	loc, _ := time.LoadLocation("Asia/Taipei")

	// Insert with normal wake time
	normalWake := time.Date(2026, 3, 25, 7, 0, 0, 0, loc).UTC()
	sleepAt := normalWake.Add(-8 * time.Hour)
	created, err := queries.CreateSleepLog(ctx, &sqlcdb.CreateSleepLogParams{
		UserID:  testUserID,
		SleepAt: sleepAt,
		WakeAt:  normalWake,
	})
	if err != nil {
		t.Fatalf("CreateSleepLog: %v", err)
	}
	if created.AbnormalWake {
		t.Fatal("expected abnormal_wake = false initially")
	}

	// Update wake_at to abnormal hour
	abnormalWake := time.Date(2026, 3, 25, 3, 30, 0, 0, loc).UTC()
	updated, err := queries.UpdateSleepLog(ctx, &sqlcdb.UpdateSleepLogParams{
		ID:     created.ID,
		UserID: testUserID,
		WakeAt: sql.NullTime{Time: abnormalWake, Valid: true},
	})
	if err != nil {
		t.Fatalf("UpdateSleepLog: %v", err)
	}
	if !updated.AbnormalWake {
		t.Error("expected abnormal_wake = true after updating wake_at to 03:30 Taipei")
	}
}

func TestDeleteSleepLogErrNoRows(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()

	created, err := queries.CreateSleepLog(ctx, &sqlcdb.CreateSleepLogParams{
		UserID:  testUserID,
		SleepAt: now.Add(-8 * time.Hour),
		WakeAt:  now,
	})
	if err != nil {
		t.Fatalf("CreateSleepLog: %v", err)
	}

	if err := queries.DeleteSleepLog(ctx, &sqlcdb.DeleteSleepLogParams{ID: created.ID, UserID: testUserID}); err != nil {
		t.Fatalf("DeleteSleepLog: %v", err)
	}

	_, err = queries.GetSleepLog(ctx, &sqlcdb.GetSleepLogParams{ID: created.ID, UserID: testUserID})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows after delete, got %v", err)
	}
}
