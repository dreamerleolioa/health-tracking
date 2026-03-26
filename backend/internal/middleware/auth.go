package middleware

import (
	"net/http"
	"strings"

	"health-tracking/backend/internal/auth"

	"github.com/gin-gonic/gin"
)

const UserIDKey = "userID"

func JWTAuth(jwtSvc *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try httpOnly cookie first
		tokenStr, err := c.Cookie("access_token")
		if err != nil {
			// Fallback to Authorization header
			header := c.GetHeader("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": gin.H{"code": "UNAUTHORIZED", "message": "missing access token"},
				})
				return
			}
			tokenStr = strings.TrimPrefix(header, "Bearer ")
		}

		claims, err := jwtSvc.Validate(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "invalid or expired token"},
			})
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Next()
	}
}
