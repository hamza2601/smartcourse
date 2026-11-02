package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func internalRouter(apiKey string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/internal/x", InternalAuth(apiKey), func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func callInternal(r *gin.Engine, token *string) int {
	req := httptest.NewRequest(http.MethodGet, "/internal/x", nil)
	if token != nil {
		req.Header.Set(InternalTokenHeader, *token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

func ptr(s string) *string { return &s }

func TestInternalAuth(t *testing.T) {
	r := internalRouter("secret-key")
	cases := []struct {
		name  string
		token *string
		want  int
	}{
		{"no header", nil, 401},
		{"empty header", ptr(""), 401},
		{"wrong key", ptr("wrong-key"), 401},
		{"prefix of key", ptr("secret"), 401},
		{"correct key", ptr("secret-key"), 200},
	}
	for _, tc := range cases {
		if got := callInternal(r, tc.token); got != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestInternalAuthFailsClosedWithoutKey(t *testing.T) {
	r := internalRouter("")
	if got := callInternal(r, ptr("")); got != 401 {
		t.Fatalf("unconfigured key must reject everything, got %d", got)
	}
}

func TestUserIDFrom(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	if _, ok := UserIDFrom(c); ok {
		t.Fatal("no identity set: want ok=false")
	}
	c.Set(ContextUserID, "not-a-uuid")
	if _, ok := UserIDFrom(c); ok {
		t.Fatal("wrong type: want ok=false")
	}
	id := uuid.New()
	c.Set(ContextUserID, id)
	if got, ok := UserIDFrom(c); !ok || got != id {
		t.Fatalf("got %v %v, want %v true", got, ok, id)
	}
}
