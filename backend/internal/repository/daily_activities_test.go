package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"math/rand"
	"strings"
	"testing"
	"time"

	sqlcdb "health-tracking/backend/db/sqlc"
)

// uniqueDate returns a random future date to prevent UNIQUE constraint collisions across test runs.
func uniqueDate(offsetDays int) time.Time {
	return time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, offsetDays+rand.Intn(10000))
}

func TestCreateAndGetDailyActivity(t *testing.T) {
	ctx := context.Background()
	date := uniqueDate(0)

	steps := sql.NullInt32{Int32: 8500, Valid: true}
	created, err := queries.CreateDailyActivity(ctx, &sqlcdb.CreateDailyActivityParams{
		UserID:       testUserID,
		ActivityDate: date,
		Steps:        steps,
	})
	if err != nil {
		t.Fatalf("CreateDailyActivity: %v", err)
	}
	if created.ID.String() == "" {
		t.Fatal("expected non-empty ID")
	}
	if !created.Steps.Valid || created.Steps.Int32 != 8500 {
		t.Errorf("steps mismatch: %+v", created.Steps)
	}

	fetched, err := queries.GetDailyActivity(ctx, &sqlcdb.GetDailyActivityParams{ID: created.ID, UserID: testUserID})
	if err != nil {
		t.Fatalf("GetDailyActivity: %v", err)
	}
	if fetched.ID != created.ID {
		t.Errorf("ID mismatch: got %v, want %v", fetched.ID, created.ID)
	}
}

func TestDailyActivityUniqueConstraint(t *testing.T) {
	ctx := context.Background()
	date := uniqueDate(10000) // distinct range from other tests

	_, err := queries.CreateDailyActivity(ctx, &sqlcdb.CreateDailyActivityParams{
		UserID:       testUserID,
		ActivityDate: date,
	})
	if err != nil {
		t.Fatalf("first CreateDailyActivity: %v", err)
	}

	_, err = queries.CreateDailyActivity(ctx, &sqlcdb.CreateDailyActivityParams{
		UserID:       testUserID,
		ActivityDate: date,
	})
	if err == nil {
		t.Fatal("expected unique violation error on duplicate date, got nil")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "23505") && !strings.Contains(errStr, "unique") {
		t.Errorf("expected unique constraint error, got: %v", err)
	}
}

func TestUpdateDailyActivityCOALESCE(t *testing.T) {
	ctx := context.Background()
	date := uniqueDate(20000) // distinct range
	mode := sqlcdb.CommuteModeTrain

	created, err := queries.CreateDailyActivity(ctx, &sqlcdb.CreateDailyActivityParams{
		UserID:       testUserID,
		ActivityDate: date,
		Steps:        sql.NullInt32{Int32: 5000, Valid: true},
		CommuteMode:  sqlcdb.NullCommuteMode{CommuteMode: mode, Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateDailyActivity: %v", err)
	}

	// Update only steps; commute_mode should remain
	updated, err := queries.UpdateDailyActivity(ctx, &sqlcdb.UpdateDailyActivityParams{
		ID:     created.ID,
		UserID: testUserID,
		Steps:  sql.NullInt32{Int32: 10000, Valid: true},
	})
	if err != nil {
		t.Fatalf("UpdateDailyActivity: %v", err)
	}
	if !updated.Steps.Valid || updated.Steps.Int32 != 10000 {
		t.Errorf("steps should be 10000, got %+v", updated.Steps)
	}
	if !updated.CommuteMode.Valid || updated.CommuteMode.CommuteMode != mode {
		t.Errorf("commute_mode should remain %v, got %+v", mode, updated.CommuteMode)
	}
}

func TestDeleteDailyActivityErrNoRows(t *testing.T) {
	ctx := context.Background()
	date := uniqueDate(30000) // distinct range

	created, err := queries.CreateDailyActivity(ctx, &sqlcdb.CreateDailyActivityParams{
		UserID:       testUserID,
		ActivityDate: date,
	})
	if err != nil {
		t.Fatalf("CreateDailyActivity: %v", err)
	}

	if err := queries.DeleteDailyActivity(ctx, &sqlcdb.DeleteDailyActivityParams{ID: created.ID, UserID: testUserID}); err != nil {
		t.Fatalf("DeleteDailyActivity: %v", err)
	}

	_, err = queries.GetDailyActivity(ctx, &sqlcdb.GetDailyActivityParams{ID: created.ID, UserID: testUserID})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows after delete, got %v", err)
	}
}
