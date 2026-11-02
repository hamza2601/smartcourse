package auth

import (
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	ContextUserID = "user_id"
	ContextEmail  = "email"
	ContextRole   = "role"
)

// Identity is what a verified access token says about the caller.
type Identity struct {
	UserID uuid.UUID
	Email  string
	Role   string
}

// VerifyFunc validates a raw access token. It keeps this library independent
// of any one service's token implementation.
type VerifyFunc func(token string) (*Identity, error)

func AuthMiddleware(verify VerifyFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || token == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "missing or malformed authorization header"})
			return
		}

		identity, err := verify(token)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set(ContextUserID, identity.UserID)
		c.Set(ContextEmail, identity.Email)
		c.Set(ContextRole, identity.Role)
		c.Next()
	}
}

// UserIDFrom returns the caller's user ID as verified by AuthMiddleware. This
// is the only trusted source of "who is calling"; IDs in request bodies are not.
func UserIDFrom(c *gin.Context) (uuid.UUID, bool) {
	v, ok := c.Get(ContextUserID)
	if !ok {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok && id != uuid.Nil
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
