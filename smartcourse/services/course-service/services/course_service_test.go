package services

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	apperrors "smartcourse/libs/common/errors"
)

func TestFetchUser(t *testing.T) {
	id := uuid.New()

	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantErr    string
		wantUpstrm bool
	}{
		{
			name: "ok",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/internal/users/"+id.String() {
					t.Errorf("unexpected path %s", r.URL.Path)
				}
				w.Write([]byte(`{"id":"` + id.String() + `","name":"Ivy","email":"ivy@x.com","role":"instructor"}`))
			},
		},
		{
			name:    "not found",
			handler: func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) },
			wantErr: "instructor not found",
		},
		{
			name:       "server error",
			handler:    func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
			wantUpstrm: true,
		},
		{
			name:       "bad body",
			handler:    func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("not json")) },
			wantUpstrm: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			user, err := NewCourseService(nil, srv.URL).fetchUser(context.Background(), id)

			var upstream *apperrors.UpstreamError
			switch {
			case tt.wantUpstrm:
				if !errors.As(err, &upstream) {
					t.Fatalf("want UpstreamError, got %v", err)
				}
			case tt.wantErr != "":
				if err == nil || err.Error() != tt.wantErr || errors.As(err, &upstream) {
					t.Fatalf("want plain error %q, got %v", tt.wantErr, err)
				}
			default:
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if user.ID != id || user.Role != "instructor" || user.Email != "ivy@x.com" {
					t.Fatalf("bad user parsed: %+v", user)
				}
			}
		})
	}
}

func TestFetchUserUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()

	_, err := NewCourseService(nil, url).fetchUser(context.Background(), uuid.New())

	var upstream *apperrors.UpstreamError
	if !errors.As(err, &upstream) {
		t.Fatalf("want UpstreamError for unreachable service, got %v", err)
	}
}
