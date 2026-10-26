package services

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"

	"smartcourse/internal/domain"
	apperrors "smartcourse/internal/errors"
)

type fakeEnrollmentStore struct {
	existing *domain.Enrollment
	created  []*domain.Enrollment
}

func (f *fakeEnrollmentStore) CreateEnrollment(_ context.Context, e *domain.Enrollment) error {
	f.created = append(f.created, e)
	return nil
}

func (f *fakeEnrollmentStore) GetEnrollmentByID(context.Context, uuid.UUID) (*domain.Enrollment, error) {
	return nil, nil
}

func (f *fakeEnrollmentStore) GetEnrollmentByStudentCourse(context.Context, uuid.UUID, uuid.UUID) (*domain.Enrollment, error) {
	return f.existing, nil
}

// fakeServer answers every request with the given status; a 200 returns body.
func fakeServer(status int, body string, hits *atomic.Int32) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(status)
		if status == http.StatusOK {
			w.Write([]byte(body))
		}
	}))
}

func TestEnrollStudent(t *testing.T) {
	studentID, courseID := uuid.New(), uuid.New()
	courseBody := `{"id":"` + courseID.String() + `","name":"Go","status":"draft"}`
	userBody := `{"id":"` + studentID.String() + `","name":"Stu","email":"stu@x.com","role":"student"}`

	tests := []struct {
		name         string
		courseStatus int
		userStatus   int
		existing     *domain.Enrollment
		courseDown   bool

		wantEnrolled bool
		wantMsg      string // plain business error
		wantUpstream bool
		wantExists   bool
		wantUserHits int32 // user-service must not be called if the course check already failed
	}{
		{
			name: "success", courseStatus: 200, userStatus: 200,
			wantEnrolled: true, wantUserHits: 1,
		},
		{
			name: "course not found", courseStatus: 404, userStatus: 200,
			wantMsg: "course not found", wantUserHits: 0,
		},
		{
			name: "student not found", courseStatus: 200, userStatus: 404,
			wantMsg: "student not found", wantUserHits: 1,
		},
		{
			name: "course-service returns 500", courseStatus: 500, userStatus: 200,
			wantUpstream: true, wantUserHits: 0,
		},
		{
			name: "course-service unreachable", courseStatus: 200, userStatus: 200, courseDown: true,
			wantUpstream: true, wantUserHits: 0,
		},
		{
			name: "user-service returns 500", courseStatus: 200, userStatus: 500,
			wantUpstream: true, wantUserHits: 1,
		},
		{
			name: "already enrolled", courseStatus: 200, userStatus: 200,
			existing:   &domain.Enrollment{Status: "active"},
			wantExists: true, wantUserHits: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var courseHits, userHits atomic.Int32
			courseSrv := fakeServer(tt.courseStatus, courseBody, &courseHits)
			userSrv := fakeServer(tt.userStatus, userBody, &userHits)
			defer userSrv.Close()
			if tt.courseDown {
				courseSrv.Close()
			} else {
				defer courseSrv.Close()
			}

			store := &fakeEnrollmentStore{existing: tt.existing}
			svc := NewEnrollmentService(store, courseSrv.URL, userSrv.URL)

			enrollment, err := svc.EnrollStudent(context.Background(), studentID, courseID)

			var upstream *apperrors.UpstreamError
			var exists *apperrors.EnrollmentAlreadyExistsError
			switch {
			case tt.wantEnrolled:
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if enrollment.StudentID != studentID || enrollment.CourseID != courseID || enrollment.Status != "active" {
					t.Fatalf("bad enrollment: %+v", enrollment)
				}
				if len(store.created) != 1 {
					t.Fatalf("want 1 enrollment saved, got %d", len(store.created))
				}
			case tt.wantUpstream:
				if !errors.As(err, &upstream) {
					t.Fatalf("want UpstreamError, got %v", err)
				}
			case tt.wantExists:
				if !errors.As(err, &exists) {
					t.Fatalf("want EnrollmentAlreadyExistsError, got %v", err)
				}
			default:
				if err == nil || err.Error() != tt.wantMsg || errors.As(err, &upstream) {
					t.Fatalf("want plain business error %q, got %v", tt.wantMsg, err)
				}
			}

			if !tt.wantEnrolled && len(store.created) != 0 {
				t.Fatalf("nothing should be saved on failure, got %d", len(store.created))
			}
			if got := userHits.Load(); got != tt.wantUserHits {
				t.Fatalf("user-service hits = %d, want %d", got, tt.wantUserHits)
			}
		})
	}
}

func TestEnrollStudentCallsInternalPaths(t *testing.T) {
	studentID, courseID := uuid.New(), uuid.New()
	var coursePath, userPath string

	courseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		coursePath = r.URL.Path
		w.Write([]byte(`{"id":"` + courseID.String() + `"}`))
	}))
	defer courseSrv.Close()
	userSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userPath = r.URL.Path
		w.Write([]byte(`{"id":"` + studentID.String() + `","role":"student"}`))
	}))
	defer userSrv.Close()

	svc := NewEnrollmentService(&fakeEnrollmentStore{}, courseSrv.URL, userSrv.URL)
	if _, err := svc.EnrollStudent(context.Background(), studentID, courseID); err != nil {
		t.Fatal(err)
	}

	if want := "/internal/courses/" + courseID.String(); coursePath != want {
		t.Errorf("course path = %q, want %q", coursePath, want)
	}
	if want := "/internal/users/" + studentID.String(); userPath != want {
		t.Errorf("user path = %q, want %q", userPath, want)
	}
}
