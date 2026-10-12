package middleware

import (
	"slices"
	"strings"

	"github.com/gin-gonic/gin"

	"smartcourse/internal/services"
)

const (
	ContextUserID = "user_id"
	ContextEmail  = "email"
	ContextRole   = "role"
)

func AuthMiddleware(tokenService *services.TokenService, secretKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || token == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "missing or malformed authorization header"})
			return
		}

		claims, err := tokenService.VerifyAccessToken(token, secretKey)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextEmail, claims.Email)
		c.Set(ContextRole, claims.Role)
		c.Next()
	}
}

// RequireRole must run after AuthMiddleware.
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !slices.Contains(roles, c.GetString(ContextRole)) {
			c.AbortWithStatusJSON(403, gin.H{"error": "insufficient permissions"})
			return
		}
		c.Next()
	}
}
