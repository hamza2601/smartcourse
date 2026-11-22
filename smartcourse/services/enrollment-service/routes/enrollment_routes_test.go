package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"smartcourse/services/enrollment-service/config"
	"smartcourse/services/enrollment-service/domain"
	"smartcourse/services/enrollment-service/services"
)

const testSecret = "route-test-secret-key-that-is-long-enough"

// The router is built with a nil database: every request below is answered by
// the middleware or by request validation, before anything would touch it.
func newTestRouter(t *testing.T) (*gin.Engine, func(role string) string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		InternalAPIKey:   "route-test-internal-key",
		CourseServiceURL: "http://unused",
		JWT: config.JWTConfig{
			SecretKey:          testSecret,
			AccessTokenExpiry:  15 * time.Minute,
			RefreshTokenExpiry: time.Hour,
		},
	}
	public, _ := SetupEnrollmentServiceRoutes(nil, cfg)

	tokens := services.NewTokenService(cfg.JWT.AccessTokenExpiry, cfg.JWT.RefreshTokenExpiry)
	tokenFor := func(role string) string {
		token, err := tokens.GenerateAccessToken(&domain.User{ID: uuid.New(), Email: role + "@test.local", Role: role}, testSecret)
		if err != nil {
			t.Fatal(err)
		}
		return token
	}
	return public, tokenFor
}

func do(r *gin.Engine, method, path, token, body string) int {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

func TestOnlyStudentsMayEnroll(t *testing.T) {
	r, tokenFor := newTestRouter(t)
	body := `{"course_id":"` + uuid.NewString() + `"}`

	if got := do(r, http.MethodPost, "/api/v1/enrollments", "", body); got != 401 {
		t.Errorf("no token: got %d, want 401", got)
	}
	for _, role := range []string{domain.RoleInstructor, domain.RoleAdmin} {
		if got := do(r, http.MethodPost, "/api/v1/enrollments", tokenFor(role), body); got != 403 {
			t.Errorf("%s: got %d, want 403", role, got)
		}
	}
	// A student gets past the role check: the handler then rejects the empty
	// body (400) before any database access.
	if got := do(r, http.MethodPost, "/api/v1/enrollments", tokenFor(domain.RoleStudent), `{}`); got != 400 {
		t.Errorf("student: got %d, want 400 (past the role gate, rejected by validation)", got)
	}
}
