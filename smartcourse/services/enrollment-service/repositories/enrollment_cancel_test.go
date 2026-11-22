package repositories

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	apperrors "smartcourse/libs/common/errors"
	"smartcourse/libs/common/outbox"
	"smartcourse/services/enrollment-service/domain"
	"smartcourse/services/enrollment-service/persistence"
)

// Same setup as the other repository tests (testRepo, enroll, seatsTaken, ...):
// needs SMARTCOURSE_TEST_DSN, and everything runs in a rolled-back transaction.

// cancelEvent builds the EnrollmentCancelled event the way the service does.
func cancelEvent(t *testing.T, e *domain.Enrollment) outbox.Event {
	t.Helper()
	ev, err := outbox.NewEvent(domain.AggregateTypeEnrollment, e.ID, domain.EventEnrollmentCancelled, domain.EnrollmentCancelledPayload{
		EnrollmentID: e.ID, StudentID: e.StudentID, CourseID: e.CourseID, Status: domain.EnrollmentStatusCancelled,
	})
	if err != nil {
		t.Fatal(err)
	}
	return ev
}

// cancel runs the repository's cancel for a copy of e, so the stored row is
// the only thing that decides the outcome (not the caller's struct).
func cancel(t *testing.T, repo *EnrollmentRepository, e *domain.Enrollment) error {
	t.Helper()
	fresh := &domain.Enrollment{ID: e.ID, StudentID: e.StudentID, CourseID: e.CourseID, Status: domain.EnrollmentStatusActive}
	return repo.CancelEnrollment(context.Background(), fresh, cancelEvent(t, fresh))
}

func countCancelledEvents(t *testing.T, tx *gorm.DB, enrollmentID uuid.UUID) int64 {
	t.Helper()
	var n int64
	if err := tx.Table(persistence.OutboxTable).
		Where("event_type = ? AND aggregate_id = ?", domain.EventEnrollmentCancelled, enrollmentID).
		Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func mustEnroll(t *testing.T, repo *EnrollmentRepository, studentID, courseID uuid.UUID, capacity *int) *domain.Enrollment {
	t.Helper()
	e, err := enroll(t, repo, studentID, courseID, capacity)
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}
	return e
}

func TestCancelReleasesTheSeat(t *testing.T) {
	repo, _ := testRepo(t)
	ctx := context.Background()
	courseID := uuid.New()

	a := mustEnroll(t, repo, uuid.New(), courseID, intPtr(2))
	mustEnroll(t, repo, uuid.New(), courseID, intPtr(2))
	if got := seatsTaken(t, repo, courseID); got != 2 {
		t.Fatalf("taken before cancel = %d, want 2", got)
	}

	// The course is full, so a third student is turned away ...
	var full *apperrors.CourseFullError
	if _, err := enroll(t, repo, uuid.New(), courseID, intPtr(2)); !errors.As(err, &full) {
		t.Fatalf("want CourseFullError, got %v", err)
	}

	toCancel := &domain.Enrollment{ID: a.ID, StudentID: a.StudentID, CourseID: courseID, Status: domain.EnrollmentStatusActive}
	if err := repo.CancelEnrollment(ctx, toCancel, cancelEvent(t, toCancel)); err != nil {
		t.Fatal(err)
	}
	if toCancel.Status != domain.EnrollmentStatusCancelled {
		t.Fatalf("returned enrollment status = %q, want cancelled", toCancel.Status)
	}
	if got := seatsTaken(t, repo, courseID); got != 1 {
		t.Fatalf("taken after cancel = %d, want 1", got)
	}
	stored, err := repo.GetEnrollmentByID(ctx, a.ID)
	if err != nil || stored.Status != domain.EnrollmentStatusCancelled {
		t.Fatalf("stored enrollment: %+v, %v; want cancelled", stored, err)
	}

	// ... and the freed seat can be taken.
	if _, err := enroll(t, repo, uuid.New(), courseID, intPtr(2)); err != nil {
		t.Fatalf("enrolling into the released seat: %v", err)
	}
}

func TestCancellingTwiceDoesNotDecrementAgain(t *testing.T) {
	repo, tx := testRepo(t)
	courseID := uuid.New()
	a := mustEnroll(t, repo, uuid.New(), courseID, intPtr(5))
	mustEnroll(t, repo, uuid.New(), courseID, intPtr(5))

	if err := cancel(t, repo, a); err != nil {
		t.Fatalf("first cancel: %v", err)
	}
	if got := seatsTaken(t, repo, courseID); got != 1 {
		t.Fatalf("taken after the first cancel = %d, want 1", got)
	}

	err := cancel(t, repo, a)
	var notActive *apperrors.EnrollmentNotActiveError
	if !errors.As(err, &notActive) || notActive.Status != domain.EnrollmentStatusCancelled {
		t.Fatalf("second cancel: want EnrollmentNotActiveError(cancelled), got %v", err)
	}
	if got := seatsTaken(t, repo, courseID); got != 1 {
		t.Fatalf("taken after the second cancel = %d, want still 1 (no double release)", got)
	}
	if got := countCancelledEvents(t, tx, a.ID); got != 1 {
		t.Fatalf("EnrollmentCancelled rows = %d, want 1", got)
	}
}

func TestReenrollAfterCancelSucceeds(t *testing.T) {
	repo, tx := testRepo(t)
	courseID, studentID := uuid.New(), uuid.New()

	first := mustEnroll(t, repo, studentID, courseID, intPtr(1))
	// The only seat is taken, so even this student cannot enroll twice ...
	var exists *apperrors.EnrollmentAlreadyExistsError
	if _, err := enroll(t, repo, studentID, courseID, intPtr(1)); !errors.As(err, &exists) {
		t.Fatalf("want EnrollmentAlreadyExistsError while active, got %v", err)
	}

	if err := cancel(t, repo, first); err != nil {
		t.Fatal(err)
	}

	// ... but after cancelling, the partial unique index no longer blocks them.
	second := mustEnroll(t, repo, studentID, courseID, intPtr(1))
	if second.ID == first.ID || second.Status != domain.EnrollmentStatusActive {
		t.Fatalf("re-enrollment: %+v", second)
	}
	if got := seatsTaken(t, repo, courseID); got != 1 {
		t.Fatalf("taken = %d, want 1 (one seat for the one active enrollment)", got)
	}
	if got := countEnrollments(t, tx, courseID); got != 2 {
		t.Fatalf("enrollment rows = %d, want 2 (one cancelled, one active)", got)
	}
}

func TestExactlyOneCancelledEventPerCancel(t *testing.T) {
	repo, tx := testRepo(t)
	courseID := uuid.New()
	a := mustEnroll(t, repo, uuid.New(), courseID, nil)
	b := mustEnroll(t, repo, uuid.New(), courseID, nil)

	if err := cancel(t, repo, a); err != nil {
		t.Fatal(err)
	}
	if got := countCancelledEvents(t, tx, a.ID); got != 1 {
		t.Fatalf("after one cancel: %d EnrollmentCancelled rows for the enrollment, want 1", got)
	}
	if err := cancel(t, repo, a); err == nil {
		t.Fatal("cancelling again should fail")
	}
	if got := countCancelledEvents(t, tx, a.ID); got != 1 {
		t.Fatalf("a rejected second cancel left %d rows, want still 1 (rolled back)", got)
	}
	if got := countCancelledEvents(t, tx, b.ID); got != 0 {
		t.Fatalf("the other enrollment has %d EnrollmentCancelled rows, want 0", got)
	}
}

func TestAnotherStudentCannotCancel(t *testing.T) {
	repo, tx := testRepo(t)
	ctx := context.Background()
	courseID := uuid.New()
	owner := mustEnroll(t, repo, uuid.New(), courseID, intPtr(3))

	intruder := &domain.Enrollment{ID: owner.ID, StudentID: uuid.New(), CourseID: courseID, Status: domain.EnrollmentStatusActive}
	err := repo.CancelEnrollment(ctx, intruder, cancelEvent(t, intruder))
	var notFound *apperrors.EnrollmentNotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("want EnrollmentNotFoundError for another student's enrollment, got %v", err)
	}

	stored, err := repo.GetEnrollmentByID(ctx, owner.ID)
	if err != nil || stored.Status != domain.EnrollmentStatusActive {
		t.Fatalf("the enrollment must stay active: %+v, %v", stored, err)
	}
	if got := seatsTaken(t, repo, courseID); got != 1 {
		t.Fatalf("taken = %d, want 1 (seat must not be released)", got)
	}
	if got := countCancelledEvents(t, tx, owner.ID); got != 0 {
		t.Fatalf("EnrollmentCancelled rows = %d, want 0", got)
	}

	// An id that does not exist at all gets the same answer.
	ghost := &domain.Enrollment{ID: uuid.New(), StudentID: owner.StudentID, CourseID: courseID, Status: domain.EnrollmentStatusActive}
	if err := repo.CancelEnrollment(ctx, ghost, cancelEvent(t, ghost)); !errors.As(err, &notFound) {
		t.Fatalf("unknown enrollment: want EnrollmentNotFoundError, got %v", err)
	}
}

// enrollAt enrolls with an explicit created_at, so ordering tests are exact.
func enrollAt(t *testing.T, repo *EnrollmentRepository, studentID, courseID uuid.UUID, createdAt time.Time) *domain.Enrollment {
	t.Helper()
	e := &domain.Enrollment{ID: uuid.New(), StudentID: studentID, CourseID: courseID, Status: domain.EnrollmentStatusActive, CreatedAt: createdAt}
	ev, err := outbox.NewEvent(domain.AggregateTypeEnrollment, e.ID, domain.EventEnrollmentCreated, domain.EnrollmentCreatedPayload{
		EnrollmentID: e.ID, StudentID: studentID, CourseID: courseID, Status: e.Status,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Enroll(context.Background(), e, nil, ev); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	return e
}

func TestHistoryIsNewestFirstAndCoversAllStatuses(t *testing.T) {
	repo, _ := testRepo(t)
	ctx := context.Background()
	me, someoneElse := uuid.New(), uuid.New()
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	oldest := enrollAt(t, repo, me, uuid.New(), base)
	middle := enrollAt(t, repo, me, uuid.New(), base.Add(time.Hour))
	newer := enrollAt(t, repo, me, uuid.New(), base.Add(2*time.Hour))
	// Two enrollments created at the same instant: id breaks the tie (DESC).
	tieA := enrollAt(t, repo, me, uuid.New(), base.Add(3*time.Hour))
	tieB := enrollAt(t, repo, me, uuid.New(), base.Add(3*time.Hour))
	enrollAt(t, repo, someoneElse, uuid.New(), base.Add(4*time.Hour))

	if err := cancel(t, repo, middle); err != nil {
		t.Fatal(err)
	}

	tied := []*domain.Enrollment{tieA, tieB}
	sort.Slice(tied, func(i, j int) bool { return tied[i].ID.String() > tied[j].ID.String() })
	want := []uuid.UUID{tied[0].ID, tied[1].ID, newer.ID, middle.ID, oldest.ID}

	got, err := repo.ListEnrollmentsByStudent(ctx, me)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("history has %d enrollments, want %d (the other student's must not appear)", len(got), len(want))
	}
	for i, id := range want {
		if got[i].ID != id {
			t.Fatalf("history[%d] = %v, want %v (newest first, id as tie-break)", i, got[i].ID, id)
		}
		if got[i].StudentID != me {
			t.Fatalf("history leaked another student's enrollment: %+v", got[i])
		}
	}
	if got[3].Status != domain.EnrollmentStatusCancelled {
		t.Fatalf("the cancelled enrollment should still be listed, with status cancelled: %+v", got[3])
	}

	none, err := repo.ListEnrollmentsByStudent(ctx, uuid.New())
	if err != nil || none == nil || len(none) != 0 {
		t.Fatalf("student without enrollments: %#v, %v; want empty non-nil list", none, err)
	}
}
