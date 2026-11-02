package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"smartcourse/libs/common/auth"
	apperrors "smartcourse/libs/common/errors"
	"smartcourse/libs/common/outbox"
	"smartcourse/services/enrollment-service/domain"
)

const testInternalKey = "test-internal-key"

type fakeEnrollmentStore struct {
	enrollErr error
	enrolled  []*domain.Enrollment
	capacity  *int
	events    []outbox.Event
}

func (f *fakeEnrollmentStore) GetEnrollmentByID(context.Context, uuid.UUID) (*domain.Enrollment, error) {
	return nil, nil
}

func (f *fakeEnrollmentStore) Enroll(_ context.Context, e *domain.Enrollment, capacity *int, ev outbox.Event) error {
	if f.enrollErr != nil {
		return f.enrollErr
	}
	f.enrolled = append(f.enrolled, e)
	f.capacity = capacity
	f.events = append(f.events, ev)
	return nil
}

// courseServer fakes course-service's /internal/courses/:id. It rejects calls
// without the internal key, like the real InternalAuth middleware does.
func courseServer(t *testing.T, status int, course *domain.Course) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(auth.InternalTokenHeader) != testInternalKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if course != nil && r.URL.Path != "/internal/courses/"+course.ID.String() {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.WriteHeader(status)
		if status == http.StatusOK {
			json.NewEncoder(w).Encode(course)
		}
	}))
}

func intPtr(n int) *int { return &n }

func TestEnrollStudent(t *testing.T) {
	studentID, courseID := uuid.New(), uuid.New()
	published := &domain.Course{ID: courseID, Status: domain.CourseStatusPublished, EnrollmentLimit: intPtr(25)}
	draft := &domain.Course{ID: courseID, Status: domain.CourseStatusDraft}

	tests := []struct {
		name       string
		status     int
		course     *domain.Course
		courseDown bool
		storeErr   error
		check      func(t *testing.T, e *domain.Enrollment, err error, store *fakeEnrollmentStore)
	}{
		{
			name: "success passes the course limit to the store", status: 200, course: published,
			check: func(t *testing.T, e *domain.Enrollment, err error, store *fakeEnrollmentStore) {
				if err != nil {
					t.Fatal(err)
				}
				if e.StudentID != studentID || e.CourseID != courseID || e.Status != domain.EnrollmentStatusActive {
					t.Fatalf("bad enrollment: %+v", e)
				}
				if len(store.enrolled) != 1 || store.capacity == nil || *store.capacity != 25 {
					t.Fatalf("store got %d enrollments, capacity %v", len(store.enrolled), store.capacity)
				}
				if len(store.events) != 1 {
					t.Fatalf("want 1 event, got %d", len(store.events))
				}
				ev := store.events[0]
				var p domain.EnrollmentCreatedPayload
				if err := json.Unmarshal(ev.Payload, &p); err != nil {
					t.Fatal(err)
				}
				if ev.EventType != domain.EventEnrollmentCreated || ev.AggregateID != e.ID ||
					p.StudentID != studentID || p.CourseID != courseID || p.EnrollmentID != e.ID {
					t.Fatalf("bad EnrollmentCreated event: %+v payload %+v", ev, p)
				}
			},
		},
		{
			name: "course not found", status: 404, course: published,
			check: func(t *testing.T, _ *domain.Enrollment, err error, _ *fakeEnrollmentStore) {
				var upstream *apperrors.UpstreamError
				if err == nil || err.Error() != "course not found" || errors.As(err, &upstream) {
					t.Fatalf("want plain 'course not found', got %v", err)
				}
			},
		},
		{
			name: "course not published", status: 200, course: draft,
			check: func(t *testing.T, _ *domain.Enrollment, err error, _ *fakeEnrollmentStore) {
				var notPublished *apperrors.CourseNotPublishedError
				if !errors.As(err, &notPublished) {
					t.Fatalf("want CourseNotPublishedError, got %v", err)
				}
			},
		},
		{
			name: "course-service returns 500", status: 500, course: published,
			check: wantUpstream,
		},
		{
			name: "course-service unreachable", status: 200, course: published, courseDown: true,
			check: wantUpstream,
		},
		{
			name: "course full", status: 200, course: published, storeErr: &apperrors.CourseFullError{CourseID: courseID},
			check: func(t *testing.T, _ *domain.Enrollment, err error, _ *fakeEnrollmentStore) {
				var full *apperrors.CourseFullError
				if !errors.As(err, &full) {
					t.Fatalf("want CourseFullError, got %v", err)
				}
			},
		},
		{
			name: "already enrolled", status: 200, course: published,
			storeErr: &apperrors.EnrollmentAlreadyExistsError{StudentID: studentID, CourseID: courseID},
			check: func(t *testing.T, _ *domain.Enrollment, err error, _ *fakeEnrollmentStore) {
				var exists *apperrors.EnrollmentAlreadyExistsError
				if !errors.As(err, &exists) {
					t.Fatalf("want EnrollmentAlreadyExistsError, got %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := courseServer(t, tt.status, tt.course)
			if tt.courseDown {
				srv.Close()
			} else {
				defer srv.Close()
			}
			store := &fakeEnrollmentStore{enrollErr: tt.storeErr}
			svc := NewEnrollmentService(store, srv.URL, testInternalKey)

			e, err := svc.EnrollStudent(context.Background(), studentID, courseID)
			tt.check(t, e, err, store)
			if err != nil && tt.storeErr == nil && len(store.enrolled) != 0 {
				t.Fatal("nothing should reach the store when the course check fails")
			}
		})
	}
}

func wantUpstream(t *testing.T, _ *domain.Enrollment, err error, _ *fakeEnrollmentStore) {
	var upstream *apperrors.UpstreamError
	if !errors.As(err, &upstream) {
		t.Fatalf("want UpstreamError, got %v", err)
	}
}

func TestEnrollStudentWrongInternalKeyIsUpstreamError(t *testing.T) {
	courseID := uuid.New()
	srv := courseServer(t, 200, &domain.Course{ID: courseID, Status: domain.CourseStatusPublished})
	defer srv.Close()

	svc := NewEnrollmentService(&fakeEnrollmentStore{}, srv.URL, "wrong-key")
	_, err := svc.EnrollStudent(context.Background(), uuid.New(), courseID)
	wantUpstream(t, nil, err, nil)
}

func TestEnrollStudentValidation(t *testing.T) {
	svc := NewEnrollmentService(&fakeEnrollmentStore{}, "http://unused", testInternalKey)
	if _, err := svc.EnrollStudent(context.Background(), uuid.Nil, uuid.New()); err == nil {
		t.Error("nil student: want error")
	}
	if _, err := svc.EnrollStudent(context.Background(), uuid.New(), uuid.Nil); err == nil {
		t.Error("nil course: want error")
	}
}
