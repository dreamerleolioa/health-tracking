package repository_test

import (
	"context"
	"testing"
	"time"

	sqlcdb "health-tracking/backend/db/sqlc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpsertUser_CreateAndUpdate(t *testing.T) {
	ctx := context.Background()

	// First insert
	user, err := queries.UpsertUser(ctx, &sqlcdb.UpsertUserParams{
		GoogleID: "google-123",
		Email:    "test@example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", user.Email)

	// Upsert again — should update last_login_at
	before := user.LastLoginAt
	time.Sleep(10 * time.Millisecond)
	updated, err := queries.UpsertUser(ctx, &sqlcdb.UpsertUserParams{
		GoogleID: "google-123",
		Email:    "test@example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, user.ID, updated.ID, "should return same user")
	assert.True(t, updated.LastLoginAt.After(before) || updated.LastLoginAt.Equal(before))
}

func TestRefreshToken_CreateAndRevoke(t *testing.T) {
	ctx := context.Background()

	user, err := queries.UpsertUser(ctx, &sqlcdb.UpsertUserParams{
		GoogleID: "google-rt", Email: "rt@test.com",
	})
	require.NoError(t, err)

	rt, err := queries.CreateRefreshToken(ctx, &sqlcdb.CreateRefreshTokenParams{
		UserID:    user.ID,
		TokenHash: "test-token-hash",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	})
	require.NoError(t, err)
	assert.False(t, rt.Revoked)

	// Lookup
	found, err := queries.GetRefreshToken(ctx, "test-token-hash")
	require.NoError(t, err)
	assert.Equal(t, rt.ID, found.ID)

	// Revoke
	err = queries.RevokeRefreshToken(ctx, "test-token-hash")
	require.NoError(t, err)

	// Should not be found anymore
	_, err = queries.GetRefreshToken(ctx, "test-token-hash")
	assert.Error(t, err, "revoked token should not be returned")
}
