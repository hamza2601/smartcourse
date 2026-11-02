package repositories

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"smartcourse/libs/common/outbox"
	"smartcourse/services/course-service/domain"
	"smartcourse/services/course-service/persistence"
)

// Needs SMARTCOURSE_TEST_DSN pointing at a database with course_service
// migrated. Each test runs in a rolled-back transaction.
func testRepo(t *testing.T) (*CourseRepository, *gorm.DB) {
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
	return NewCourseRepository(tx), tx
}

func countEvents(t *testing.T, tx *gorm.DB, eventType string, courseID uuid.UUID) int64 {
	t.Helper()
	var n int64
	if err := tx.Table(persistence.OutboxTable).Where("event_type = ? AND aggregate_id = ?", eventType, courseID).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func newEvent(t *testing.T, eventType string, courseID uuid.UUID) outbox.Event {
	t.Helper()
	ev, err := outbox.NewEvent(domain.AggregateTypeCourse, courseID, eventType, map[string]string{"course_id": courseID.String()})
	if err != nil {
		t.Fatal(err)
	}
	return ev
}

func TestCreateCourseWritesCourseCreatedEvent(t *testing.T) {
	repo, tx := testRepo(t)
	course := &domain.Course{ID: uuid.New(), Name: "Go", InstructorID: uuid.New(), Status: domain.CourseStatusDraft}

	if err := repo.CreateCourse(context.Background(), course, newEvent(t, domain.EventCourseCreated, course.ID)); err != nil {
		t.Fatal(err)
	}
	if got := countEvents(t, tx, domain.EventCourseCreated, course.ID); got != 1 {
		t.Fatalf("CourseCreated rows = %d, want 1", got)
	}
}

func TestPublishWritesEventOnlyWhenStatusChanges(t *testing.T) {
	repo, tx := testRepo(t)
	ctx := context.Background()
	course := &domain.Course{ID: uuid.New(), Name: "Go", InstructorID: uuid.New(), Status: domain.CourseStatusDraft}
	if err := repo.CreateCourse(ctx, course, newEvent(t, domain.EventCourseCreated, course.ID)); err != nil {
		t.Fatal(err)
	}

	updated, err := repo.UpdateCourseStatus(ctx, course.ID, domain.CourseStatusDraft, domain.CourseStatusPublished, newEvent(t, domain.EventCoursePublished, course.ID))
	if err != nil || !updated {
		t.Fatalf("publish: updated=%v err=%v", updated, err)
	}
	// Second attempt: no longer a draft, so no update and no second event.
	updated, err = repo.UpdateCourseStatus(ctx, course.ID, domain.CourseStatusDraft, domain.CourseStatusPublished, newEvent(t, domain.EventCoursePublished, course.ID))
	if err != nil || updated {
		t.Fatalf("re-publish: updated=%v err=%v, want false", updated, err)
	}
	if got := countEvents(t, tx, domain.EventCoursePublished, course.ID); got != 1 {
		t.Fatalf("CoursePublished rows = %d, want exactly 1", got)
	}
}
