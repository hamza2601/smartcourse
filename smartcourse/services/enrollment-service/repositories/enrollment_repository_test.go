package repositories

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	apperrors "smartcourse/libs/common/errors"
	"smartcourse/libs/common/outbox"
	"smartcourse/services/enrollment-service/domain"
	"smartcourse/services/enrollment-service/persistence"
)

// These tests need a Postgres with the enrollment_service schema migrated,
// e.g. the docker-compose database:
//
//	SMARTCOURSE_TEST_DSN="host=localhost port=5432 user=postgres password=password dbname=smartcourse_db sslmode=disable"
//
// Each test runs inside a transaction that is rolled back, so nothing is left
// behind; the repository's own transaction becomes a savepoint inside it.
func testRepo(t *testing.T) (*EnrollmentRepository, *gorm.DB) {
	t.Helper()
	dsn := os.Getenv("SMARTCOURSE_TEST_DSN")
	if dsn == "" {
		t.Skip("SMARTCOURSE_TEST_DSN not set; skipping database test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	tx := db.Begin()
	t.Cleanup(func() {
		tx.Rollback()
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return NewEnrollmentRepository(tx), tx
}

func intPtr(n int) *int { return &n }

// enroll builds an enrollment (and its EnrollmentCreated event, keyed by the
// same enrollment ID) and runs it through the repository.
func enroll(t *testing.T, repo *EnrollmentRepository, studentID, courseID uuid.UUID, capacity *int) (*domain.Enrollment, error) {
	t.Helper()
	e := &domain.Enrollment{ID: uuid.New(), StudentID: studentID, CourseID: courseID, Status: domain.EnrollmentStatusActive}
	ev, err := outbox.NewEvent(domain.AggregateTypeEnrollment, e.ID, domain.EventEnrollmentCreated, domain.EnrollmentCreatedPayload{
		EnrollmentID: e.ID, StudentID: studentID, CourseID: courseID, Status: e.Status,
	})
	if err != nil {
		t.Fatal(err)
	}
	return e, repo.Enroll(context.Background(), e, capacity, ev)
}

func seatsTaken(t *testing.T, repo *EnrollmentRepository, courseID uuid.UUID) int {
	t.Helper()
	seat, err := repo.GetCourseSeat(context.Background(), courseID)
	if err != nil || seat == nil {
		t.Fatalf("seat row: %v %v", seat, err)
	}
	return seat.Taken
}

func countEnrollments(t *testing.T, tx *gorm.DB, courseID uuid.UUID) int64 {
	t.Helper()
	var n int64
	if err := tx.Table(persistence.EnrollmentDB{}.TableName()).Where("course_id = ?", courseID).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

// countEvents counts EnrollmentCreated outbox rows for the given enrollments.
func countEvents(t *testing.T, tx *gorm.DB, enrollmentIDs ...uuid.UUID) int64 {
	t.Helper()
	var n int64
	if err := tx.Table(persistence.OutboxTable).
		Where("event_type = ? AND aggregate_id IN ?", domain.EventEnrollmentCreated, enrollmentIDs).
		Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func TestEnrollWritesEnrollmentSeatAndEventTogether(t *testing.T) {
	repo, tx := testRepo(t)
	courseID := uuid.New()

	e, err := enroll(t, repo, uuid.New(), courseID, intPtr(3))
	if err != nil {
		t.Fatal(err)
	}
	if got := countEnrollments(t, tx, courseID); got != 1 {
		t.Fatalf("enrollments = %d, want 1", got)
	}
	if got := seatsTaken(t, repo, courseID); got != 1 {
		t.Fatalf("taken = %d, want 1", got)
	}
	if got := countEvents(t, tx, e.ID); got != 1 {
		t.Fatalf("EnrollmentCreated rows = %d, want 1", got)
	}
}

func TestEnrollFillsSeatsThenRejectsWithoutEvent(t *testing.T) {
	repo, tx := testRepo(t)
	courseID := uuid.New()

	for i := 0; i < 2; i++ {
		if _, err := enroll(t, repo, uuid.New(), courseID, intPtr(2)); err != nil {
			t.Fatalf("enroll %d: %v", i+1, err)
		}
	}
	rejected, err := enroll(t, repo, uuid.New(), courseID, intPtr(2))
	var full *apperrors.CourseFullError
	if !errors.As(err, &full) {
		t.Fatalf("third enroll: want CourseFullError, got %v", err)
	}
	if got := seatsTaken(t, repo, courseID); got != 2 {
		t.Fatalf("taken = %d, want 2", got)
	}
	if got := countEnrollments(t, tx, courseID); got != 2 {
		t.Fatalf("enrollments = %d, want 2 (rejected enroll must not insert)", got)
	}
	if got := countEvents(t, tx, rejected.ID); got != 0 {
		t.Fatalf("rejected enroll left %d outbox rows, want 0 (rollback)", got)
	}
}

func TestEnrollDuplicateRollsBackSeatAndEvent(t *testing.T) {
	repo, tx := testRepo(t)
	courseID, studentID := uuid.New(), uuid.New()

	if _, err := enroll(t, repo, studentID, courseID, intPtr(5)); err != nil {
		t.Fatal(err)
	}
	again, err := enroll(t, repo, studentID, courseID, intPtr(5))
	var exists *apperrors.EnrollmentAlreadyExistsError
	if !errors.As(err, &exists) {
		t.Fatalf("want EnrollmentAlreadyExistsError, got %v", err)
	}
	if got := seatsTaken(t, repo, courseID); got != 1 {
		t.Fatalf("taken = %d, want 1 (duplicate must not keep a seat)", got)
	}
	if got := countEnrollments(t, tx, courseID); got != 1 {
		t.Fatalf("enrollments = %d, want 1", got)
	}
	if got := countEvents(t, tx, again.ID); got != 0 {
		t.Fatalf("duplicate enroll left %d outbox rows, want 0", got)
	}
}

func TestEnrollUnlimitedCourse(t *testing.T) {
	repo, _ := testRepo(t)
	courseID := uuid.New()

	for i := 0; i < 5; i++ {
		if _, err := enroll(t, repo, uuid.New(), courseID, nil); err != nil {
			t.Fatalf("enroll %d into unlimited course: %v", i+1, err)
		}
	}
	if got := seatsTaken(t, repo, courseID); got != 5 {
		t.Fatalf("taken = %d, want 5", got)
	}
}

func TestEnrollLimitLoweredBelowTakenMeansFull(t *testing.T) {
	repo, _ := testRepo(t)
	courseID := uuid.New()

	for i := 0; i < 3; i++ {
		if _, err := enroll(t, repo, uuid.New(), courseID, intPtr(10)); err != nil {
			t.Fatal(err)
		}
	}
	// The course's limit is now 2 although 3 seats are taken.
	_, err := enroll(t, repo, uuid.New(), courseID, intPtr(2))
	var full *apperrors.CourseFullError
	if !errors.As(err, &full) {
		t.Fatalf("want CourseFullError, got %v", err)
	}
	// The rejected enroll rolled back entirely, including its capacity refresh.
	if got := seatsTaken(t, repo, courseID); got != 3 {
		t.Fatalf("taken = %d, want 3", got)
	}
}
