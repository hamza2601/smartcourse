package services

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	apperrors "smartcourse/libs/common/errors"
	"smartcourse/libs/common/outbox"
	"smartcourse/services/course-service/domain"
	"smartcourse/services/course-service/workflows"
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

// UpdateCourse mimics the repository: it only changes draft or ready courses.
func (f *fakeCourseStore) UpdateCourse(_ context.Context, id uuid.UUID, name, description *string, limit *int) (*domain.Course, error) {
	c, ok := f.courses[id]
	if !ok {
		return nil, &apperrors.CourseNotFoundError{CourseID: id}
	}
	if c.Status != domain.CourseStatusDraft && c.Status != domain.CourseStatusReady {
		return nil, &apperrors.CourseNotEditableError{CourseID: id, Status: c.Status}
	}
	if name != nil {
		c.Name = *name
	}
	if description != nil {
		c.Description = *description
	}
	if limit != nil {
		c.EnrollmentLimit = limit
	}
	copy := *c
	return &copy, nil
}

type fakeStarter struct {
	calls []startCall
	err   error
}

type startCall struct {
	courseID uuid.UUID
	previous string
}

func (f *fakeStarter) StartPublish(_ context.Context, id uuid.UUID, previous string) error {
	f.calls = append(f.calls, startCall{id, previous})
	return f.err
}

func intPtr(n int) *int { return &n }

func TestCreateCourse(t *testing.T) {
	instructor := uuid.New()
	store := newFakeCourseStore()
	svc := NewCourseService(store, nil)

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
	svc := NewCourseService(newFakeCourseStore(), nil)
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

	setup := func(status string) (*CourseService, *fakeStarter, uuid.UUID) {
		store := newFakeCourseStore()
		starter := &fakeStarter{}
		id := uuid.New()
		store.courses[id] = &domain.Course{ID: id, InstructorID: owner, Status: status}
		return NewCourseService(store, starter), starter, id
	}

	for _, status := range []string{domain.CourseStatusDraft, domain.CourseStatusReady} {
		t.Run("owner starts publishing a "+status+" course", func(t *testing.T) {
			svc, starter, id := setup(status)
			if err := svc.PublishCourse(context.Background(), id, owner); err != nil {
				t.Fatal(err)
			}
			if len(starter.calls) != 1 || starter.calls[0] != (startCall{id, status}) {
				t.Fatalf("starter calls = %+v, want one for %s with previous status %q", starter.calls, id, status)
			}
		})
	}

	t.Run("not the owner", func(t *testing.T) {
		svc, starter, id := setup(domain.CourseStatusDraft)
		err := svc.PublishCourse(context.Background(), id, other)
		var notOwned *apperrors.CourseNotOwnedError
		if !errors.As(err, &notOwned) {
			t.Fatalf("want CourseNotOwnedError, got %v", err)
		}
		if len(starter.calls) != 0 {
			t.Fatal("starter must not be called")
		}
	})

	for _, status := range []string{domain.CourseStatusPublishing, domain.CourseStatusArchived} {
		t.Run("course is "+status, func(t *testing.T) {
			svc, starter, id := setup(status)
			err := svc.PublishCourse(context.Background(), id, owner)
			var conflict *apperrors.CourseStatusConflictError
			if !errors.As(err, &conflict) || conflict.Status != status {
				t.Fatalf("want CourseStatusConflictError for %s, got %v", status, err)
			}
			if len(starter.calls) != 0 {
				t.Fatal("starter must not be called")
			}
		})
	}

	t.Run("unknown course", func(t *testing.T) {
		svc, starter, _ := setup(domain.CourseStatusDraft)
		err := svc.PublishCourse(context.Background(), uuid.New(), owner)
		var notFound *apperrors.CourseNotFoundError
		if !errors.As(err, &notFound) {
			t.Fatalf("want CourseNotFoundError, got %v", err)
		}
		if len(starter.calls) != 0 {
			t.Fatal("starter must not be called")
		}
	})

	t.Run("starter errors propagate", func(t *testing.T) {
		svc, starter, id := setup(domain.CourseStatusDraft)
		for _, want := range []error{workflows.ErrPublishInProgress, workflows.ErrPublishUnavailable, errors.New("boom")} {
			starter.err = want
			if err := svc.PublishCourse(context.Background(), id, owner); !errors.Is(err, want) {
				t.Fatalf("err = %v, want %v", err, want)
			}
		}
	})
}

func TestUpdateCourse(t *testing.T) {
	owner, other := uuid.New(), uuid.New()
	str := func(s string) *string { return &s }

	setup := func(status string) (*CourseService, uuid.UUID) {
		store := newFakeCourseStore()
		id := uuid.New()
		store.courses[id] = &domain.Course{
			ID: id, Name: "Go", Description: "intro", InstructorID: owner, Status: status, EnrollmentLimit: intPtr(10),
		}
		return NewCourseService(store, &fakeStarter{}), id
	}

	for _, status := range []string{domain.CourseStatusDraft, domain.CourseStatusReady} {
		t.Run("owner updates all fields of a "+status+" course", func(t *testing.T) {
			svc, id := setup(status)
			got, err := svc.UpdateCourse(context.Background(), id, owner, str("  Go 102  "), str(""), intPtr(50))
			if err != nil {
				t.Fatal(err)
			}
			if got.Name != "Go 102" || got.Description != "" || got.EnrollmentLimit == nil || *got.EnrollmentLimit != 50 || got.Status != status {
				t.Fatalf("unexpected course: %+v", got)
			}
		})
	}

	t.Run("omitted fields are unchanged", func(t *testing.T) {
		svc, id := setup(domain.CourseStatusDraft)
		got, err := svc.UpdateCourse(context.Background(), id, owner, nil, str("new description"), nil)
		if err != nil {
			t.Fatal(err)
		}
		if got.Name != "Go" || got.Description != "new description" || *got.EnrollmentLimit != 10 {
			t.Fatalf("unexpected course: %+v", got)
		}
		same, err := svc.UpdateCourse(context.Background(), id, owner, nil, nil, nil)
		if err != nil || same.Name != "Go" || same.Description != "new description" || *same.EnrollmentLimit != 10 {
			t.Fatalf("empty update should return the course unchanged: %+v, %v", same, err)
		}
	})

	t.Run("validation", func(t *testing.T) {
		svc, id := setup(domain.CourseStatusDraft)
		if _, err := svc.UpdateCourse(context.Background(), id, owner, str("   "), nil, nil); err == nil {
			t.Error("blank name: want error")
		}
		for _, limit := range []int{0, -3} {
			if _, err := svc.UpdateCourse(context.Background(), id, owner, nil, nil, intPtr(limit)); err == nil {
				t.Errorf("limit %d: want error", limit)
			}
		}
	})

	for _, status := range []string{domain.CourseStatusPublishing, domain.CourseStatusArchived} {
		t.Run("not editable when "+status, func(t *testing.T) {
			svc, id := setup(status)
			_, err := svc.UpdateCourse(context.Background(), id, owner, str("x"), nil, nil)
			var notEditable *apperrors.CourseNotEditableError
			if !errors.As(err, &notEditable) || notEditable.Status != status {
				t.Fatalf("want CourseNotEditableError for %s, got %v", status, err)
			}
		})
	}

	t.Run("not the owner", func(t *testing.T) {
		svc, id := setup(domain.CourseStatusDraft)
		_, err := svc.UpdateCourse(context.Background(), id, other, str("x"), nil, nil)
		var notOwned *apperrors.CourseNotOwnedError
		if !errors.As(err, &notOwned) {
			t.Fatalf("want CourseNotOwnedError, got %v", err)
		}
	})

	t.Run("unknown course", func(t *testing.T) {
		svc, _ := setup(domain.CourseStatusDraft)
		_, err := svc.UpdateCourse(context.Background(), uuid.New(), owner, str("x"), nil, nil)
		var notFound *apperrors.CourseNotFoundError
		if !errors.As(err, &notFound) {
			t.Fatalf("want CourseNotFoundError, got %v", err)
		}
	})
}
