package auth

import (
	"crypto/sha256"
	"crypto/subtle"

	"github.com/gin-gonic/gin"
)

// InternalTokenHeader carries the shared service-to-service key.
const InternalTokenHeader = "X-Internal-Token"

// InternalAuth guards /internal/* routes: the request must carry
// X-Internal-Token equal to apiKey. An empty apiKey rejects everything.
func InternalAuth(apiKey string) gin.HandlerFunc {
	// Compare fixed-length digests so the check is constant-time regardless
	// of how long the presented token is.
	want := sha256.Sum256([]byte(apiKey))
	return func(c *gin.Context) {
		got := sha256.Sum256([]byte(c.GetHeader(InternalTokenHeader)))
		if apiKey == "" || subtle.ConstantTimeCompare(got[:], want[:]) != 1 {
			c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}
