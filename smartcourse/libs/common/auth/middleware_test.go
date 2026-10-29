package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func newRouter(roles ...string) (*gin.Engine, *Identity) {
	gin.SetMode(gin.TestMode)
	seen := &Identity{}
	verify := func(token string) (*Identity, error) {
		if token != "good" {
			return nil, errors.New("bad token")
		}
		return &Identity{UserID: uuid.New(), Email: "a@x.com", Role: "instructor"}, nil
	}

	r := gin.New()
	handlers := []gin.HandlerFunc{AuthMiddleware(verify)}
	if len(roles) > 0 {
		handlers = append(handlers, RequireRole(roles...))
	}
	handlers = append(handlers, func(c *gin.Context) {
		seen.Email = c.GetString(ContextEmail)
		seen.Role = c.GetString(ContextRole)
		c.Status(http.StatusOK)
	})
	r.GET("/x", handlers...)
	return r, seen
}

func call(r *gin.Engine, authHeader string) int {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

func TestAuthMiddleware(t *testing.T) {
	r, seen := newRouter()
	if got := call(r, "Bearer good"); got != 200 {
		t.Fatalf("valid token: got %d, want 200", got)
	}
	if seen.Email != "a@x.com" || seen.Role != "instructor" {
		t.Fatalf("identity not placed in context: %+v", seen)
	}
	for name, header := range map[string]string{
		"no header":    "",
		"not bearer":   "Basic good",
		"empty bearer": "Bearer ",
		"bad token":    "Bearer nope",
	} {
		if got := call(r, header); got != 401 {
			t.Errorf("%s: got %d, want 401", name, got)
		}
	}
}

func TestRequireRole(t *testing.T) {
	allowed, _ := newRouter("instructor", "admin")
	if got := call(allowed, "Bearer good"); got != 200 {
		t.Errorf("allowed role: got %d, want 200", got)
	}
	denied, _ := newRouter("admin")
	if got := call(denied, "Bearer good"); got != 403 {
		t.Errorf("denied role: got %d, want 403", got)
	}
}
