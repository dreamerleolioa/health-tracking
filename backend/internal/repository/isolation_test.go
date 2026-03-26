package repository_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	sqlcdb "health-tracking/backend/db/sqlc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBodyMetrics_UserIsolation(t *testing.T) {
	ctx := context.Background()

	userA, err := queries.UpsertUser(ctx, &sqlcdb.UpsertUserParams{
		GoogleID: "google-a", Email: "a@test.com",
	})
	require.NoError(t, err)
	userB, err := queries.UpsertUser(ctx, &sqlcdb.UpsertUserParams{
		GoogleID: "google-b", Email: "b@test.com",
	})
	require.NoError(t, err)

	_, err = queries.CreateBodyMetric(ctx, &sqlcdb.CreateBodyMetricParams{
		UserID:     userA.ID,
		WeightKg:   sql.NullString{String: "72.5", Valid: true},
		RecordedAt: time.Now(),
	})
	require.NoError(t, err)

	metricsB, err := queries.ListBodyMetrics(ctx, &sqlcdb.ListBodyMetricsParams{
		UserID: userB.ID,
		Limit:  10,
	})
	require.NoError(t, err)
	assert.Empty(t, metricsB, "user B should not see user A's data")

	metricsA, err := queries.ListBodyMetrics(ctx, &sqlcdb.ListBodyMetricsParams{
		UserID: userA.ID,
		Limit:  10,
	})
	require.NoError(t, err)
	assert.Len(t, metricsA, 1)
}

func TestSleepLogs_UserIsolation(t *testing.T) {
	ctx := context.Background()

	userA, err := queries.UpsertUser(ctx, &sqlcdb.UpsertUserParams{GoogleID: "google-sl-a", Email: "sla@test.com"})
	require.NoError(t, err)
	userB, err := queries.UpsertUser(ctx, &sqlcdb.UpsertUserParams{GoogleID: "google-sl-b", Email: "slb@test.com"})
	require.NoError(t, err)

	_, err = queries.CreateSleepLog(ctx, &sqlcdb.CreateSleepLogParams{
		UserID:  userA.ID,
		SleepAt: time.Now().Add(-8 * time.Hour),
		WakeAt:  time.Now(),
	})
	require.NoError(t, err)

	logsB, err := queries.ListSleepLogs(ctx, &sqlcdb.ListSleepLogsParams{UserID: userB.ID, Limit: 10})
	require.NoError(t, err)
	assert.Empty(t, logsB, "user B should not see user A's sleep logs")
}

func TestDailyActivities_UserIsolation(t *testing.T) {
	ctx := context.Background()

	userA, err := queries.UpsertUser(ctx, &sqlcdb.UpsertUserParams{GoogleID: "google-da-a", Email: "daa@test.com"})
	require.NoError(t, err)
	userB, err := queries.UpsertUser(ctx, &sqlcdb.UpsertUserParams{GoogleID: "google-da-b", Email: "dab@test.com"})
	require.NoError(t, err)

	_, err = queries.CreateDailyActivity(ctx, &sqlcdb.CreateDailyActivityParams{
		UserID:       userA.ID,
		ActivityDate: time.Now(),
	})
	require.NoError(t, err)

	activitiesB, err := queries.ListDailyActivities(ctx, &sqlcdb.ListDailyActivitiesParams{UserID: userB.ID, Limit: 10})
	require.NoError(t, err)
	assert.Empty(t, activitiesB, "user B should not see user A's daily activities")
}
