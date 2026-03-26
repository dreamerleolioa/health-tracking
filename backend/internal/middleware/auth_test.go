package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"health-tracking/backend/internal/auth"
	"health-tracking/backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func setupMiddlewareRouter(jwtSvc *auth.JWTService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", middleware.JWTAuth(jwtSvc), func(c *gin.Context) {
		uid := c.MustGet(middleware.UserIDKey).(uuid.UUID)
		c.JSON(http.StatusOK, gin.H{"user_id": uid.String()})
	})
	return r
}

func TestJWTAuth_CookiePath(t *testing.T) {
	jwtSvc := auth.NewJWTService("test-secret-32-chars-long-xxxxx", 15*time.Minute)
	userID := uuid.New()
	token, _ := jwtSvc.SignAccess(userID)

	r := setupMiddlewareRouter(jwtSvc)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestJWTAuth_BearerHeaderPath(t *testing.T) {
	jwtSvc := auth.NewJWTService("test-secret-32-chars-long-xxxxx", 15*time.Minute)
	userID := uuid.New()
	token, _ := jwtSvc.SignAccess(userID)

	r := setupMiddlewareRouter(jwtSvc)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestJWTAuth_NoToken_Returns401(t *testing.T) {
	jwtSvc := auth.NewJWTService("test-secret-32-chars-long-xxxxx", 15*time.Minute)

	r := setupMiddlewareRouter(jwtSvc)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_ExpiredToken_Returns401(t *testing.T) {
	jwtSvc := auth.NewJWTService("test-secret-32-chars-long-xxxxx", -1*time.Second) // already expired
	token, _ := jwtSvc.SignAccess(uuid.New())

	validSvc := auth.NewJWTService("test-secret-32-chars-long-xxxxx", 15*time.Minute)
	r := setupMiddlewareRouter(validSvc)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
