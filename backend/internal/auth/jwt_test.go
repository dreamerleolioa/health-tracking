package auth_test

import (
	"testing"
	"time"

	"health-tracking/backend/internal/auth"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTService_SignAndValidate(t *testing.T) {
	svc := auth.NewJWTService("test-secret-32-chars-long-xxxxx", 15*time.Minute)
	userID := uuid.New()

	token, err := svc.SignAccess(userID)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := svc.Validate(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
}

func TestJWTService_ExpiredToken(t *testing.T) {
	svc := auth.NewJWTService("test-secret-32-chars-long-xxxxx", -1*time.Second) // 已過期
	userID := uuid.New()

	token, err := svc.SignAccess(userID)
	require.NoError(t, err)

	_, err = svc.Validate(token)
	assert.Error(t, err, "should reject expired token")
}

func TestJWTService_InvalidSignature(t *testing.T) {
	svc1 := auth.NewJWTService("secret-one-32-chars-long-xxxxxx", 15*time.Minute)
	svc2 := auth.NewJWTService("secret-two-32-chars-long-xxxxxx", 15*time.Minute)

	token, err := svc1.SignAccess(uuid.New())
	require.NoError(t, err)

	_, err = svc2.Validate(token)
	assert.Error(t, err, "should reject token signed with different secret")
}
