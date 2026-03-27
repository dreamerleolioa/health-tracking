package handler_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlcdb "health-tracking/backend/db/sqlc"
	"health-tracking/backend/internal/auth"
	"health-tracking/backend/internal/handler"
	"health-tracking/backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sha256HexTest(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// mockAuthStore implements handler.AuthStore
type mockAuthStore struct {
	user         sqlcdb.User
	refreshToken sqlcdb.RefreshToken
	upsertErr    error
}

func (m *mockAuthStore) UpsertUser(_ context.Context, _ *sqlcdb.UpsertUserParams) (sqlcdb.User, error) {
	return m.user, m.upsertErr
}
func (m *mockAuthStore) GetUserByID(_ context.Context, _ uuid.UUID) (sqlcdb.User, error) {
	return m.user, nil
}
func (m *mockAuthStore) GetUserByGoogleID(_ context.Context, _ string) (sqlcdb.User, error) {
	return m.user, nil
}
func (m *mockAuthStore) CreateRefreshToken(_ context.Context, _ *sqlcdb.CreateRefreshTokenParams) (sqlcdb.RefreshToken, error) {
	return m.refreshToken, nil
}
func (m *mockAuthStore) GetRefreshToken(_ context.Context, hash string) (sqlcdb.RefreshToken, error) {
	if hash == m.refreshToken.TokenHash {
		return m.refreshToken, nil
	}
	return sqlcdb.RefreshToken{}, sql.ErrNoRows
}
func (m *mockAuthStore) RevokeRefreshToken(_ context.Context, _ string) error       { return nil }
func (m *mockAuthStore) RevokeAllUserRefreshTokens(_ context.Context, _ uuid.UUID) error { return nil }

func setupAuthRouter(store handler.AuthStore, jwtSvc *auth.JWTService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.NewAuthHandler(store, jwtSvc, "cid", "csec", "http://localhost/callback", "http://localhost:5173", 7*24*time.Hour, false)
	v1 := r.Group("/v1/auth")
	v1.POST("/logout", h.Logout)
	v1.POST("/refresh", h.RefreshToken)
	v1.GET("/me", middleware.JWTAuth(jwtSvc), h.Me)
	return r
}

func TestAuthHandler_Me_Authenticated(t *testing.T) {
	userID := uuid.New()
	store := &mockAuthStore{user: sqlcdb.User{ID: userID, Email: "test@example.com"}}
	jwtSvc := auth.NewJWTService("test-secret-32-chars-long-xxxxx", 15*time.Minute)

	token, err := jwtSvc.SignAccess(userID)
	require.NoError(t, err)

	r := setupAuthRouter(store, jwtSvc)
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp["data"])
}

func TestAuthHandler_Me_Unauthenticated(t *testing.T) {
	store := &mockAuthStore{}
	jwtSvc := auth.NewJWTService("test-secret-32-chars-long-xxxxx", 15*time.Minute)

	r := setupAuthRouter(store, jwtSvc)
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil) // no cookie/header
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_Logout(t *testing.T) {
	store := &mockAuthStore{
		refreshToken: sqlcdb.RefreshToken{TokenHash: "mytoken"},
	}
	jwtSvc := auth.NewJWTService("test-secret-32-chars-long-xxxxx", 15*time.Minute)

	r := setupAuthRouter(store, jwtSvc)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "mytoken"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestAuthHandler_Refresh_ValidToken(t *testing.T) {
	userID := uuid.New()
	// The refresh token stored has SHA-256("validtoken")
	rawToken := "validtoken"
	tokenHash := sha256HexTest(rawToken)
	store := &mockAuthStore{
		user: sqlcdb.User{ID: userID, Email: "test@example.com"},
		refreshToken: sqlcdb.RefreshToken{
			TokenHash: tokenHash,
			UserID:    userID,
			ExpiresAt: time.Now().Add(time.Hour),
		},
	}
	jwtSvc := auth.NewJWTService("test-secret-32-chars-long-xxxxx", 15*time.Minute)

	r := setupAuthRouter(store, jwtSvc)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: rawToken})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_Refresh_InvalidToken(t *testing.T) {
	store := &mockAuthStore{
		refreshToken: sqlcdb.RefreshToken{TokenHash: sha256HexTest("validtoken")},
	}
	jwtSvc := auth.NewJWTService("test-secret-32-chars-long-xxxxx", 15*time.Minute)

	r := setupAuthRouter(store, jwtSvc)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "wrongtoken"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
