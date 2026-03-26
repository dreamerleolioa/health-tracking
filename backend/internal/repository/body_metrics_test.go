package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strconv"
	"testing"
	"time"

	sqlcdb "health-tracking/backend/db/sqlc"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

var (
	queries    *sqlcdb.Queries
	testUserID uuid.UUID
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	pgContainer, connStr := setupPostgres(ctx)
	defer pgContainer.Terminate(ctx) //nolint:errcheck

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		panic("open db: " + err.Error())
	}
	defer db.Close()

	if err := runMigrations(connStr); err != nil {
		panic("migrations: " + err.Error())
	}

	queries = sqlcdb.New(db)

	// Create a test user for all repository tests
	user, err := queries.UpsertUser(ctx, &sqlcdb.UpsertUserParams{
		GoogleID: "test-google-id",
		Email:    "test@repository.test",
	})
	if err != nil {
		panic("create test user: " + err.Error())
	}
	testUserID = user.ID

	os.Exit(m.Run())
}

func setupPostgres(ctx context.Context) (testcontainers.Container, string) {
	pgC, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		panic("start postgres container: " + err.Error())
	}

	connStr, err := pgC.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic("get connection string: " + err.Error())
	}
	return pgC, connStr
}

func runMigrations(connStr string) error {
	// Path is relative to where `go test` is run (backend/ directory)
	m, err := migrate.New("file://../../db/migrations", connStr)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func mustParseFloat(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic("mustParseFloat: " + err.Error())
	}
	return f
}

// --- Tests ---

func TestCreateAndGetBodyMetric(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Millisecond)

	wkg := sql.NullString{String: "72.5", Valid: true}
	created, err := queries.CreateBodyMetric(ctx, &sqlcdb.CreateBodyMetricParams{
		UserID:     testUserID,
		WeightKg:   wkg,
		RecordedAt: now,
	})
	if err != nil {
		t.Fatalf("CreateBodyMetric: %v", err)
	}
	if created.ID.String() == "" {
		t.Fatal("expected non-empty ID")
	}
	if !created.WeightKg.Valid || mustParseFloat(created.WeightKg.String) != 72.5 {
		t.Errorf("weight_kg mismatch: %+v", created.WeightKg)
	}

	fetched, err := queries.GetBodyMetric(ctx, &sqlcdb.GetBodyMetricParams{ID: created.ID, UserID: testUserID})
	if err != nil {
		t.Fatalf("GetBodyMetric: %v", err)
	}
	if fetched.ID != created.ID {
		t.Errorf("ID mismatch: got %v, want %v", fetched.ID, created.ID)
	}
}

func TestListBodyMetricsDateRange(t *testing.T) {
	ctx := context.Background()

	// Insert records in different months
	dates := []time.Time{
		time.Date(2026, 1, 15, 8, 0, 0, 0, time.UTC),
		time.Date(2026, 2, 15, 8, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 15, 8, 0, 0, 0, time.UTC),
	}
	for _, d := range dates {
		if _, err := queries.CreateBodyMetric(ctx, &sqlcdb.CreateBodyMetricParams{
			UserID:     testUserID,
			RecordedAt: d,
		}); err != nil {
			t.Fatalf("CreateBodyMetric: %v", err)
		}
	}

	// Filter to February only
	from := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)
	results, err := queries.ListBodyMetrics(ctx, &sqlcdb.ListBodyMetricsParams{
		UserID: testUserID,
		From:   sql.NullTime{Time: from, Valid: true},
		To:     sql.NullTime{Time: to, Valid: true},
		Limit:  100,
	})
	if err != nil {
		t.Fatalf("ListBodyMetrics: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for February, got %d", len(results))
	}
}

func TestUpdateBodyMetricCOALESCE(t *testing.T) {
	ctx := context.Background()

	fat := sql.NullString{String: "18.5", Valid: true}
	created, err := queries.CreateBodyMetric(ctx, &sqlcdb.CreateBodyMetricParams{
		UserID:     testUserID,
		WeightKg:   sql.NullString{String: "70.0", Valid: true},
		BodyFatPct: fat,
		RecordedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateBodyMetric: %v", err)
	}

	// Update only weight_kg, body_fat_pct should remain unchanged
	updated, err := queries.UpdateBodyMetric(ctx, &sqlcdb.UpdateBodyMetricParams{
		ID:       created.ID,
		UserID:   testUserID,
		WeightKg: sql.NullString{String: "75.0", Valid: true},
	})
	if err != nil {
		t.Fatalf("UpdateBodyMetric: %v", err)
	}
	if !updated.WeightKg.Valid || mustParseFloat(updated.WeightKg.String) != 75.0 {
		t.Errorf("weight_kg should be 75.0, got %+v", updated.WeightKg)
	}
	if !updated.BodyFatPct.Valid || mustParseFloat(updated.BodyFatPct.String) != 18.5 {
		t.Errorf("body_fat_pct should remain 18.5, got %+v", updated.BodyFatPct)
	}
}

func TestDeleteBodyMetricErrNoRows(t *testing.T) {
	ctx := context.Background()

	created, err := queries.CreateBodyMetric(ctx, &sqlcdb.CreateBodyMetricParams{
		UserID:     testUserID,
		RecordedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateBodyMetric: %v", err)
	}

	if err := queries.DeleteBodyMetric(ctx, &sqlcdb.DeleteBodyMetricParams{ID: created.ID, UserID: testUserID}); err != nil {
		t.Fatalf("DeleteBodyMetric: %v", err)
	}

	_, err = queries.GetBodyMetric(ctx, &sqlcdb.GetBodyMetricParams{ID: created.ID, UserID: testUserID})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows after delete, got %v", err)
	}
}
