package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	sqlcdb "health-tracking/backend/db/sqlc"
	"health-tracking/backend/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// AuthStore defines DB operations needed by auth handlers.
type AuthStore interface {
	UpsertUser(ctx context.Context, arg *sqlcdb.UpsertUserParams) (sqlcdb.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (sqlcdb.User, error)
	CreateRefreshToken(ctx context.Context, arg *sqlcdb.CreateRefreshTokenParams) (sqlcdb.RefreshToken, error)
	GetRefreshToken(ctx context.Context, tokenHash string) (sqlcdb.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error
}

type AuthHandler struct {
	store       AuthStore
	jwtSvc      *auth.JWTService
	oauth       *oauth2.Config
	frontendURL string
	refreshTTL  time.Duration
}

func NewAuthHandler(store AuthStore, jwtSvc *auth.JWTService, clientID, clientSecret, redirectURL, frontendURL string, refreshTTL time.Duration) *AuthHandler {
	oauthCfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
	return &AuthHandler{
		store:       store,
		jwtSvc:      jwtSvc,
		oauth:       oauthCfg,
		frontendURL: frontendURL,
		refreshTTL:  refreshTTL,
	}
}

// GET /v1/auth/google
func (h *AuthHandler) RedirectToGoogle(c *gin.Context) {
	state := generateState()
	c.SetCookie("oauth_state", state, 600, "/", "", false, true)
	url := h.oauth.AuthCodeURL(state, oauth2.AccessTypeOffline)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// GET /v1/auth/google/callback
func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	// Validate state
	stateCookie, err := c.Cookie("oauth_state")
	if err != nil || stateCookie != c.Query("state") {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_STATE", "OAuth state mismatch"))
		return
	}
	c.SetCookie("oauth_state", "", -1, "/", "", false, true) // clear

	ctx, cancel := withTimeout(c)
	defer cancel()

	// Exchange code for token
	oauthToken, err := h.oauth.Exchange(ctx, c.Query("code"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("OAUTH_EXCHANGE_FAILED", "failed to exchange code"))
		return
	}

	// Get user info from Google
	info, err := fetchGoogleUserInfo(ctx, h.oauth, oauthToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("USERINFO_FAILED", "failed to fetch user info"))
		return
	}

	// Upsert user
	user, err := h.store.UpsertUser(ctx, &sqlcdb.UpsertUserParams{
		GoogleID:    info.Sub,
		Email:       info.Email,
		DisplayName: sqlNullString(info.Name),
		AvatarUrl:   sqlNullString(info.Picture),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("DB_ERROR", "failed to upsert user"))
		return
	}

	// Sign access token
	accessToken, err := h.jwtSvc.SignAccess(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("TOKEN_ERROR", "failed to sign token"))
		return
	}

	// Create refresh token — store SHA-256(rawToken) for lookup
	rawRefresh := generateToken()
	tokenHash := sha256Hex(rawRefresh)
	_, err = h.store.CreateRefreshToken(ctx, &sqlcdb.CreateRefreshTokenParams{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(h.refreshTTL),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("DB_ERROR", "failed to create refresh token"))
		return
	}

	// Set httpOnly cookies
	c.SetCookie("access_token", accessToken, int(15*time.Minute/time.Second), "/", "", false, true)
	c.SetCookie("refresh_token", rawRefresh, int(h.refreshTTL/time.Second), "/", "", false, true)

	// Redirect to frontend
	c.Redirect(http.StatusTemporaryRedirect, h.frontendURL+"/auth/callback")
}

// POST /v1/auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	rawRefresh, err := c.Cookie("refresh_token")
	if err != nil {
		rawRefresh = c.PostForm("refresh_token")
	}
	if rawRefresh == "" {
		c.JSON(http.StatusUnauthorized, errorResponse("MISSING_REFRESH_TOKEN", "refresh token required"))
		return
	}

	ctx, cancel := withTimeout(c)
	defer cancel()

	// Look up by SHA-256(rawToken) — same hash stored at creation time
	rt, err := h.store.GetRefreshToken(ctx, sha256Hex(rawRefresh))
	if err != nil {
		c.JSON(http.StatusUnauthorized, errorResponse("INVALID_REFRESH_TOKEN", "token not found or expired"))
		return
	}

	user, err := h.store.GetUserByID(ctx, rt.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, errorResponse("USER_NOT_FOUND", "user not found"))
		return
	}

	// Rotate: create new token FIRST, only revoke old if creation succeeds
	accessToken, err := h.jwtSvc.SignAccess(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("TOKEN_ERROR", "failed to sign token"))
		return
	}
	newRawRefresh := generateToken()
	newHash := sha256Hex(newRawRefresh)
	if _, err := h.store.CreateRefreshToken(ctx, &sqlcdb.CreateRefreshTokenParams{
		UserID:    user.ID,
		TokenHash: newHash,
		ExpiresAt: time.Now().Add(h.refreshTTL),
	}); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("DB_ERROR", "failed to create refresh token"))
		return
	}
	// Only revoke old token after new one is safely stored
	_ = h.store.RevokeRefreshToken(ctx, sha256Hex(rawRefresh))

	c.SetCookie("access_token", accessToken, int(15*time.Minute/time.Second), "/", "", false, true)
	c.SetCookie("refresh_token", newRawRefresh, int(h.refreshTTL/time.Second), "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"access_token": accessToken}})
}

// POST /v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	rawRefresh, _ := c.Cookie("refresh_token")
	if rawRefresh != "" {
		ctx, cancel := withTimeout(c)
		defer cancel()
		_ = h.store.RevokeRefreshToken(ctx, sha256Hex(rawRefresh))
	}
	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)
	c.Status(http.StatusNoContent)
}

// GET /v1/auth/me  (requires JWT middleware)
func (h *AuthHandler) Me(c *gin.Context) {
	userID := mustUserID(c)
	ctx, cancel := withTimeout(c)
	defer cancel()

	user, err := h.store.GetUserByID(ctx, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("NOT_FOUND", "user not found"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"id":            user.ID,
		"email":         user.Email,
		"display_name":  user.DisplayName,
		"avatar_url":    user.AvatarUrl,
		"created_at":    user.CreatedAt,
		"last_login_at": user.LastLoginAt,
	}})
}

// ---- helpers ----

func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

type googleUserInfo struct {
	Sub     string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func fetchGoogleUserInfo(ctx context.Context, cfg *oauth2.Config, token *oauth2.Token) (*googleUserInfo, error) {
	client := cfg.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var info googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

func errorResponse(code, message string) gin.H {
	return gin.H{"error": gin.H{"code": code, "message": message}}
}

func mustUserID(c *gin.Context) uuid.UUID {
	return c.MustGet("userID").(uuid.UUID)
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func sqlNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
