package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"smartcourse/libs/common/auth"
)

func TestClientSendsInternalKey(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get(auth.InternalTokenHeader)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	var out map[string]any
	if _, err := FetchInternal(context.Background(), NewClient("k-123"), "svc", srv.URL, &out); err != nil {
		t.Fatal(err)
	}
	if got != "k-123" {
		t.Fatalf("X-Internal-Token = %q, want %q", got, "k-123")
	}
}

func TestClientDoesNotMutateCallerRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err := NewClient("k").Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if req.Header.Get(auth.InternalTokenHeader) != "" {
		t.Fatal("transport modified the caller's request headers")
	}
}
