package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"

	apperrors "smartcourse/libs/common/errors"
	"smartcourse/libs/common/outbox"
	"smartcourse/services/enrollment-service/domain"
)

// memEnrollmentStore is an in-memory EnrollmentStore that actually holds
// enrollments, for the read, history and cancel rules.
type memEnrollmentStore struct {
	enrollments map[uuid.UUID]*domain.Enrollment
	order       []uuid.UUID // insertion order

	cancelErr    error // returned by CancelEnrollment when set (simulates the store's guarded update losing a race)
	cancelCalls  int
	cancelEvents []outbox.Event
}

func newMemStore() *memEnrollmentStore {
	return &memEnrollmentStore{enrollments: map[uuid.UUID]*domain.Enrollment{}}
}

func (m *memEnrollmentStore) add(studentID uuid.UUID, status string) *domain.Enrollment {
	e := &domain.Enrollment{ID: uuid.New(), StudentID: studentID, CourseID: uuid.New(), Status: status}
	m.enrollments[e.ID] = e
	m.order = append(m.order, e.ID)
	return e
}

func (m *memEnrollmentStore) GetEnrollmentByID(_ context.Context, id uuid.UUID) (*domain.Enrollment, error) {
	e, ok := m.enrollments[id]
	if !ok {
		return nil, &apperrors.EnrollmentNotFoundError{EnrollmentID: id}
	}
	copy := *e
	return &copy, nil
}

func (m *memEnrollmentStore) Enroll(context.Context, *domain.Enrollment, *int, outbox.Event) error {
	return nil
}

func (m *memEnrollmentStore) ListEnrollmentsByStudent(_ context.Context, studentID uuid.UUID) ([]domain.Enrollment, error) {
	out := []domain.Enrollment{}
	for _, id := range m.order {
		if e := m.enrollments[id]; e.StudentID == studentID {
			out = append(out, *e)
		}
	}
	return out, nil
}

func (m *memEnrollmentStore) CancelEnrollment(_ context.Context, e *domain.Enrollment, ev outbox.Event) error {
	m.cancelCalls++
	if m.cancelErr != nil {
		return m.cancelErr
	}
	stored := m.enrollments[e.ID]
	stored.Status = domain.EnrollmentStatusCancelled
	m.cancelEvents = append(m.cancelEvents, ev)
	*e = *stored
	return nil
}

func newAccessService(store EnrollmentStore) *EnrollmentService {
	return NewEnrollmentService(store, "http://unused", testInternalKey)
}

func TestGetEnrollmentForCaller(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	owner, otherStudent, admin, instructor := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	enrollment := store.add(owner, domain.EnrollmentStatusActive)
	svc := newAccessService(store)

	got, err := svc.GetEnrollmentForCaller(ctx, owner, domain.RoleStudent, enrollment.ID)
	if err != nil || got.ID != enrollment.ID {
		t.Fatalf("owner: %v, %v", got, err)
	}

	got, err = svc.GetEnrollmentForCaller(ctx, admin, domain.RoleAdmin, enrollment.ID)
	if err != nil || got.ID != enrollment.ID {
		t.Fatalf("admin: %v, %v", got, err)
	}

	var notFound *apperrors.EnrollmentNotFoundError
	if _, err := svc.GetEnrollmentForCaller(ctx, otherStudent, domain.RoleStudent, enrollment.ID); !errors.As(err, &notFound) {
		t.Fatalf("other student: want EnrollmentNotFoundError, got %v", err)
	}
	if _, err := svc.GetEnrollmentForCaller(ctx, instructor, domain.RoleInstructor, enrollment.ID); !errors.As(err, &notFound) {
		t.Fatalf("non-owner instructor: want EnrollmentNotFoundError, got %v", err)
	}
	if _, err := svc.GetEnrollmentForCaller(ctx, owner, domain.RoleStudent, uuid.New()); !errors.As(err, &notFound) {
		t.Fatalf("unknown id: want EnrollmentNotFoundError, got %v", err)
	}
}

func TestCancelEnrollmentByOwner(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	student := uuid.New()
	enrollment := store.add(student, domain.EnrollmentStatusActive)
	svc := newAccessService(store)

	got, err := svc.CancelEnrollment(ctx, student, enrollment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != enrollment.ID || got.Status != domain.EnrollmentStatusCancelled {
		t.Fatalf("returned enrollment: %+v", got)
	}
	if store.cancelCalls != 1 || len(store.cancelEvents) != 1 {
		t.Fatalf("store called %d times with %d events, want 1 and 1", store.cancelCalls, len(store.cancelEvents))
	}

	ev := store.cancelEvents[0]
	var payload domain.EnrollmentCancelledPayload
	if err := json.Unmarshal(ev.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if ev.EventType != domain.EventEnrollmentCancelled || ev.AggregateType != domain.AggregateTypeEnrollment || ev.AggregateID != enrollment.ID {
		t.Fatalf("bad event envelope: %+v", ev)
	}
	if payload.EnrollmentID != enrollment.ID || payload.StudentID != student || payload.CourseID != enrollment.CourseID ||
		payload.Status != domain.EnrollmentStatusCancelled {
		t.Fatalf("bad EnrollmentCancelled payload: %+v", payload)
	}
}

func TestCancelEnrollmentByNonOwnerIsNotFound(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	owner, other := uuid.New(), uuid.New()
	enrollment := store.add(owner, domain.EnrollmentStatusActive)
	svc := newAccessService(store)

	var notFound *apperrors.EnrollmentNotFoundError
	if _, err := svc.CancelEnrollment(ctx, other, enrollment.ID); !errors.As(err, &notFound) {
		t.Fatalf("other student: want EnrollmentNotFoundError, got %v", err)
	}
	if _, err := svc.CancelEnrollment(ctx, owner, uuid.New()); !errors.As(err, &notFound) {
		t.Fatalf("unknown id: want EnrollmentNotFoundError, got %v", err)
	}
	if store.cancelCalls != 0 {
		t.Fatalf("a rejected cancel reached the store %d times", store.cancelCalls)
	}
	if store.enrollments[enrollment.ID].Status != domain.EnrollmentStatusActive {
		t.Fatal("a rejected cancel changed the enrollment")
	}
}

func TestCancelingTwiceReturnsNotActive(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	student := uuid.New()
	enrollment := store.add(student, domain.EnrollmentStatusActive)
	svc := newAccessService(store)

	if _, err := svc.CancelEnrollment(ctx, student, enrollment.ID); err != nil {
		t.Fatalf("first cancel: %v", err)
	}
	_, err := svc.CancelEnrollment(ctx, student, enrollment.ID)
	var notActive *apperrors.EnrollmentNotActiveError
	if !errors.As(err, &notActive) || notActive.Status != domain.EnrollmentStatusCancelled {
		t.Fatalf("second cancel: want EnrollmentNotActiveError(cancelled), got %v", err)
	}
	if store.cancelCalls != 1 || len(store.cancelEvents) != 1 {
		t.Fatalf("store called %d times with %d events, want exactly 1 and 1", store.cancelCalls, len(store.cancelEvents))
	}
}

func TestCancelPassesThroughTheStoresVerdict(t *testing.T) {
	// The service's pre-check can be stale (a concurrent cancel); whatever the
	// store's guarded update decides must reach the caller unchanged.
	ctx := context.Background()
	store := newMemStore()
	student := uuid.New()
	enrollment := store.add(student, domain.EnrollmentStatusActive)
	store.cancelErr = &apperrors.EnrollmentNotActiveError{EnrollmentID: enrollment.ID, Status: domain.EnrollmentStatusCancelled}

	_, err := newAccessService(store).CancelEnrollment(ctx, student, enrollment.ID)
	var notActive *apperrors.EnrollmentNotActiveError
	if !errors.As(err, &notActive) {
		t.Fatalf("want the store's EnrollmentNotActiveError, got %v", err)
	}
}

func TestHistoryReturnsOnlyTheCallersRows(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	me, someoneElse := uuid.New(), uuid.New()
	mine1 := store.add(me, domain.EnrollmentStatusActive)
	store.add(someoneElse, domain.EnrollmentStatusActive)
	mine2 := store.add(me, domain.EnrollmentStatusCancelled)
	store.add(someoneElse, domain.EnrollmentStatusCancelled)
	svc := newAccessService(store)

	got, err := svc.ListEnrollmentsForStudent(ctx, me)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != mine1.ID || got[1].ID != mine2.ID {
		t.Fatalf("history = %+v, want exactly the caller's two enrollments (any status)", got)
	}
	for _, e := range got {
		if e.StudentID != me {
			t.Fatalf("history leaked another student's enrollment: %+v", e)
		}
	}

	none, err := svc.ListEnrollmentsForStudent(ctx, uuid.New())
	if err != nil || none == nil || len(none) != 0 {
		t.Fatalf("student without enrollments: %#v, %v; want empty non-nil list", none, err)
	}
}
