package repositories

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"smartcourse/services/course-service/domain"
)

func TestTransitionStatusIsConditionalAndWritesNoEvent(t *testing.T) {
	repo, tx := testRepo(t)
	ctx := context.Background()
	course := &domain.Course{ID: uuid.New(), Name: "Go", InstructorID: uuid.New(), Status: domain.CourseStatusDraft}
	if err := repo.CreateCourse(ctx, course, newEvent(t, domain.EventCourseCreated, course.ID)); err != nil {
		t.Fatal(err)
	}
	var before int64
	if err := tx.Table("course_service.outbox_events").Count(&before).Error; err != nil {
		t.Fatal(err)
	}

	moved, err := repo.TransitionStatus(ctx, course.ID, domain.CourseStatusDraft, domain.CourseStatusPublishing)
	if err != nil || !moved {
		t.Fatalf("draft -> publishing: moved=%v err=%v", moved, err)
	}
	got, err := repo.GetCourseByID(ctx, course.ID)
	if err != nil || got.Status != domain.CourseStatusPublishing {
		t.Fatalf("status = %v, err = %v; want publishing", got, err)
	}

	// Wrong current status: nothing changes.
	moved, err = repo.TransitionStatus(ctx, course.ID, domain.CourseStatusDraft, domain.CourseStatusPublishing)
	if err != nil || moved {
		t.Fatalf("second move: moved=%v err=%v, want false", moved, err)
	}
	// Unknown course: nothing changes.
	moved, err = repo.TransitionStatus(ctx, uuid.New(), domain.CourseStatusDraft, domain.CourseStatusPublishing)
	if err != nil || moved {
		t.Fatalf("unknown course: moved=%v err=%v, want false", moved, err)
	}

	var after int64
	if err := tx.Table("course_service.outbox_events").Count(&after).Error; err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("outbox rows %d -> %d; TransitionStatus must not write events", before, after)
	}
}
