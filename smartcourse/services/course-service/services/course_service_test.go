package services

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	apperrors "smartcourse/libs/common/errors"
	"smartcourse/libs/common/outbox"
	"smartcourse/services/course-service/domain"
)

type fakeCourseStore struct {
	courses map[uuid.UUID]*domain.Course
	events  []outbox.Event
	created []*domain.Course
}

func newFakeCourseStore() *fakeCourseStore {
	return &fakeCourseStore{courses: map[uuid.UUID]*domain.Course{}}
}

func (f *fakeCourseStore) CreateCourse(_ context.Context, c *domain.Course, ev outbox.Event) error {
	f.events = append(f.events, ev)
	f.created = append(f.created, c)
	f.courses[c.ID] = c
	return nil
}

func (f *fakeCourseStore) GetCourseByID(_ context.Context, id uuid.UUID) (*domain.Course, error) {
	c, ok := f.courses[id]
	if !ok {
		return nil, &apperrors.CourseNotFoundError{CourseID: id}
	}
	copy := *c
	return &copy, nil
}

func (f *fakeCourseStore) ListAllCourses(context.Context) ([]domain.Course, error) { return nil, nil }

func (f *fakeCourseStore) UpdateCourseStatus(_ context.Context, id uuid.UUID, from, to string, ev outbox.Event) (bool, error) {
	c, ok := f.courses[id]
	if !ok || c.Status != from {
		return false, nil
	}
	c.Status = to
	f.events = append(f.events, ev)
	return true, nil
}

func intPtr(n int) *int { return &n }

func TestCreateCourse(t *testing.T) {
	instructor := uuid.New()
	store := newFakeCourseStore()
	svc := NewCourseService(store)

	course, err := svc.CreateCourse(context.Background(), instructor, "Go 101", "intro", intPtr(30))
	if err != nil {
		t.Fatal(err)
	}
	if course.InstructorID != instructor {
		t.Errorf("instructor = %v, want caller %v", course.InstructorID, instructor)
	}
	if course.Status != domain.CourseStatusDraft {
		t.Errorf("status = %q, want draft", course.Status)
	}
	if course.ID == uuid.Nil || course.EnrollmentLimit == nil || *course.EnrollmentLimit != 30 {
		t.Errorf("unexpected course: %+v", course)
	}
	if len(store.created) != 1 {
		t.Fatalf("want 1 course stored, got %d", len(store.created))
	}
	if len(store.events) != 1 || store.events[0].EventType != domain.EventCourseCreated ||
		store.events[0].AggregateID != course.ID || store.events[0].AggregateType != domain.AggregateTypeCourse {
		t.Fatalf("want one CourseCreated event for the course, got %+v", store.events)
	}

	unlimited, err := svc.CreateCourse(context.Background(), instructor, "Open", "", nil)
	if err != nil || unlimited.EnrollmentLimit != nil {
		t.Fatalf("nil limit should mean unlimited: %+v, %v", unlimited, err)
	}
}

func TestCreateCourseValidation(t *testing.T) {
	svc := NewCourseService(newFakeCourseStore())
	cases := []struct {
		name       string
		instructor uuid.UUID
		title      string
		limit      *int
	}{
		{"no instructor", uuid.Nil, "x", nil},
		{"blank name", uuid.New(), "   ", nil},
		{"zero limit", uuid.New(), "x", intPtr(0)},
		{"negative limit", uuid.New(), "x", intPtr(-5)},
	}
	for _, tc := range cases {
		if _, err := svc.CreateCourse(context.Background(), tc.instructor, tc.title, "", tc.limit); err == nil {
			t.Errorf("%s: want error", tc.name)
		}
	}
}

func TestPublishCourse(t *testing.T) {
	owner, other := uuid.New(), uuid.New()

	setup := func(status string) (*CourseService, *fakeCourseStore, uuid.UUID) {
		store := newFakeCourseStore()
		id := uuid.New()
		store.courses[id] = &domain.Course{ID: id, InstructorID: owner, Status: status}
		return NewCourseService(store), store, id
	}

	t.Run("owner publishes draft", func(t *testing.T) {
		svc, store, id := setup(domain.CourseStatusDraft)
		course, err := svc.PublishCourse(context.Background(), id, owner)
		if err != nil {
			t.Fatal(err)
		}
		if course.Status != domain.CourseStatusPublished {
			t.Fatalf("status = %q, want published", course.Status)
		}
		if len(store.events) != 1 || store.events[0].EventType != domain.EventCoursePublished || store.events[0].AggregateID != id {
			t.Fatalf("want one CoursePublished event, got %+v", store.events)
		}
	})

	t.Run("not the owner", func(t *testing.T) {
		svc, store, id := setup(domain.CourseStatusDraft)
		_, err := svc.PublishCourse(context.Background(), id, other)
		var notOwned *apperrors.CourseNotOwnedError
		if !errors.As(err, &notOwned) {
			t.Fatalf("want CourseNotOwnedError, got %v", err)
		}
		if len(store.events) != 0 {
			t.Fatalf("rejected publish must not emit events, got %d", len(store.events))
		}
	})

	for _, status := range []string{domain.CourseStatusPublished, domain.CourseStatusArchived} {
		t.Run("already "+status, func(t *testing.T) {
			svc, store, id := setup(status)
			_, err := svc.PublishCourse(context.Background(), id, owner)
			var conflict *apperrors.CourseStatusConflictError
			if !errors.As(err, &conflict) {
				t.Fatalf("want CourseStatusConflictError, got %v", err)
			}
			if len(store.events) != 0 {
				t.Fatalf("rejected publish must not emit events, got %d", len(store.events))
			}
		})
	}

	t.Run("unknown course", func(t *testing.T) {
		svc, _, _ := setup(domain.CourseStatusDraft)
		_, err := svc.PublishCourse(context.Background(), uuid.New(), owner)
		var notFound *apperrors.CourseNotFoundError
		if !errors.As(err, &notFound) {
			t.Fatalf("want CourseNotFoundError, got %v", err)
		}
	})
}
