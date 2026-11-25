package workflows

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"

	"smartcourse/libs/common/outbox"
	"smartcourse/services/course-service/domain"
)

type fakeCourses struct {
	course *domain.Course
	events []outbox.Event
	// raceTo, when set, changes the status just before the next conditional
	// update, as if another actor got there first.
	raceTo string
}

func (f *fakeCourses) GetCourseByID(_ context.Context, id uuid.UUID) (*domain.Course, error) {
	c := *f.course
	return &c, nil
}

func (f *fakeCourses) race() {
	if f.raceTo != "" {
		f.course.Status, f.raceTo = f.raceTo, ""
	}
}

func (f *fakeCourses) UpdateCourseStatus(_ context.Context, _ uuid.UUID, from, to string, ev outbox.Event) (bool, error) {
	f.race()
	if f.course.Status != from {
		return false, nil
	}
	f.course.Status = to
	f.events = append(f.events, ev)
	return true, nil
}

func (f *fakeCourses) TransitionStatus(_ context.Context, _ uuid.UUID, from, to string) (bool, error) {
	f.race()
	if f.course.Status != from {
		return false, nil
	}
	f.course.Status = to
	return true, nil
}

type fakeContent struct{ modules []domain.ModuleWithLessons }

func (f fakeContent) GetCourseContent(context.Context, uuid.UUID) ([]domain.ModuleWithLessons, error) {
	return f.modules, nil
}

func newActivities(status string, modules ...domain.ModuleWithLessons) (*PublishActivities, *fakeCourses) {
	courses := &fakeCourses{course: &domain.Course{
		ID: uuid.New(), Name: "Go", Description: "About Go", InstructorID: uuid.New(), Status: status,
	}}
	return NewPublishActivities(courses, fakeContent{modules}, slog.New(slog.NewTextHandler(io.Discard, nil))), courses
}

func module(title string, pos int, lessons ...domain.Lesson) domain.ModuleWithLessons {
	return domain.ModuleWithLessons{Module: domain.Module{Title: title, Position: pos}, Lessons: lessons}
}

func lesson(title, content string, pos int) domain.Lesson {
	return domain.Lesson{Title: title, Content: content, Position: pos}
}

func assertNonRetryable(t *testing.T, err error, wantMsg string) {
	t.Helper()
	var appErr *temporal.ApplicationError
	if !errors.As(err, &appErr) || !appErr.NonRetryable() {
		t.Fatalf("want a non-retryable application error, got %v", err)
	}
	if appErr.Message() != wantMsg {
		t.Fatalf("message = %q, want %q", appErr.Message(), wantMsg)
	}
}

func TestBeginPublishing(t *testing.T) {
	for _, prev := range []string{"draft", "ready"} {
		a, courses := newActivities(prev)
		in := PublishCourseInput{CourseID: courses.course.ID, PreviousStatus: prev}
		if err := a.BeginPublishing(context.Background(), in); err != nil {
			t.Fatalf("%s: %v", prev, err)
		}
		if courses.course.Status != "publishing" {
			t.Fatalf("%s: status = %s, want publishing", prev, courses.course.Status)
		}
		// A retry after success is a no-op.
		if err := a.BeginPublishing(context.Background(), in); err != nil {
			t.Fatalf("%s retry: %v", prev, err)
		}
	}
}

func TestBeginPublishingRejectsWrongStatus(t *testing.T) {
	a, courses := newActivities("archived")
	err := a.BeginPublishing(context.Background(), PublishCourseInput{CourseID: courses.course.ID, PreviousStatus: "draft"})
	if err == nil {
		t.Fatal("want an error")
	}
	var appErr *temporal.ApplicationError
	if !errors.As(err, &appErr) || !appErr.NonRetryable() {
		t.Fatalf("want a non-retryable error, got %v", err)
	}
	if courses.course.Status != "archived" {
		t.Fatalf("status changed to %s", courses.course.Status)
	}
}

func TestBeginPublishingLostRace(t *testing.T) {
	// Someone else already moved it to publishing: treated as done.
	a, courses := newActivities("draft")
	courses.raceTo = "publishing"
	if err := a.BeginPublishing(context.Background(), PublishCourseInput{CourseID: courses.course.ID, PreviousStatus: "draft"}); err != nil {
		t.Fatal(err)
	}
	// Someone archived it: a conflict.
	a, courses = newActivities("draft")
	courses.raceTo = "archived"
	if err := a.BeginPublishing(context.Background(), PublishCourseInput{CourseID: courses.course.ID, PreviousStatus: "draft"}); err == nil {
		t.Fatal("want a conflict error")
	}
}

func TestValidateCourse(t *testing.T) {
	a, courses := newActivities("publishing", module("M1", 1, lesson("L1", "text", 1)))
	if err := a.ValidateCourse(context.Background(), courses.course.ID); err != nil {
		t.Fatal(err)
	}
}

func TestValidateCourseReportsEveryProblem(t *testing.T) {
	a, courses := newActivities("publishing",
		module("Empty", 1),
		module("Untitled", 2, lesson("  ", "text", 1)),
	)
	courses.course.Description = " "

	err := a.ValidateCourse(context.Background(), courses.course.ID)

	assertNonRetryable(t, err, "course validation failed: course description is empty; "+
		`module "Empty" has no lessons; module "Untitled" has a lesson with an empty title`)
}

func TestVerifyAssetsListsEmptyLessonsInOrder(t *testing.T) {
	// More lessons than the concurrency bound, listed out of position order.
	var lessons []domain.Lesson
	for i := 10; i >= 1; i-- {
		content := "text"
		if i%3 == 0 {
			content = "  "
		}
		lessons = append(lessons, lesson(fmt.Sprintf("B%d", i), content, i))
	}
	a, courses := newActivities("publishing",
		module("second", 2, lessons...),
		module("first", 1, lesson("A1", "", 1), lesson("A2", "text", 2)),
	)

	for range 20 { // run repeatedly: the result must not depend on scheduling
		err := a.VerifyAssets(context.Background(), courses.course.ID)
		assertNonRetryable(t, err, `lessons with empty content: "A1", "B3", "B6", "B9"`)
	}
}

func TestVerifyAssetsPassesWhenAllLessonsHaveContent(t *testing.T) {
	a, courses := newActivities("publishing", module("M", 1, lesson("L1", "x", 1), lesson("L2", "y", 2)))
	if err := a.VerifyAssets(context.Background(), courses.course.ID); err != nil {
		t.Fatal(err)
	}
}

func TestFinalizePublishing(t *testing.T) {
	a, courses := newActivities("publishing")
	courses.course.EnrollmentLimit = new(25)

	if err := a.FinalizePublishing(context.Background(), courses.course.ID); err != nil {
		t.Fatal(err)
	}
	if courses.course.Status != "ready" {
		t.Fatalf("status = %s, want ready", courses.course.Status)
	}
	if len(courses.events) != 1 || courses.events[0].EventType != domain.EventCoursePublished {
		t.Fatalf("events = %+v, want one CoursePublished", courses.events)
	}
	var p domain.CoursePublishedPayload
	if err := json.Unmarshal(courses.events[0].Payload, &p); err != nil {
		t.Fatal(err)
	}
	if p.CourseID != courses.course.ID || p.InstructorID != courses.course.InstructorID || p.Name != "Go" ||
		p.EnrollmentLimit == nil || *p.EnrollmentLimit != 25 {
		t.Fatalf("payload = %+v", p)
	}

	// A retry after success is a no-op and writes no second event.
	if err := a.FinalizePublishing(context.Background(), courses.course.ID); err != nil {
		t.Fatal(err)
	}
	if len(courses.events) != 1 {
		t.Fatalf("events after retry = %d, want 1", len(courses.events))
	}
}

func TestFinalizePublishingRejectsOtherStatuses(t *testing.T) {
	a, courses := newActivities("draft")
	if err := a.FinalizePublishing(context.Background(), courses.course.ID); err == nil {
		t.Fatal("want an error for a draft course")
	}
	if len(courses.events) != 0 {
		t.Fatal("no event may be written")
	}
}

func TestFinalizePublishingLostRace(t *testing.T) {
	a, courses := newActivities("publishing")
	courses.raceTo = "ready"
	if err := a.FinalizePublishing(context.Background(), courses.course.ID); err != nil {
		t.Fatalf("already ready after a lost race should succeed: %v", err)
	}
	a, courses = newActivities("publishing")
	courses.raceTo = "draft"
	if err := a.FinalizePublishing(context.Background(), courses.course.ID); err == nil {
		t.Fatal("want a conflict error")
	}
}

func TestRevertPublishing(t *testing.T) {
	for _, prev := range []string{"draft", "ready"} {
		a, courses := newActivities("publishing")
		in := RevertInput{CourseID: courses.course.ID, PreviousStatus: prev, Reason: "lessons with empty content: \"L1\""}

		if err := a.RevertPublishing(context.Background(), in); err != nil {
			t.Fatalf("%s: %v", prev, err)
		}
		if courses.course.Status != prev {
			t.Fatalf("status = %s, want %s", courses.course.Status, prev)
		}
		if len(courses.events) != 1 || courses.events[0].EventType != domain.EventCoursePublishFailed {
			t.Fatalf("events = %+v, want one CoursePublishFailed", courses.events)
		}
		var p domain.CoursePublishFailedPayload
		if err := json.Unmarshal(courses.events[0].Payload, &p); err != nil {
			t.Fatal(err)
		}
		if p.CourseID != courses.course.ID || p.InstructorID != courses.course.InstructorID || p.Reason != in.Reason {
			t.Fatalf("payload = %+v", p)
		}

		// A retry after success is a no-op and writes no second event.
		if err := a.RevertPublishing(context.Background(), in); err != nil {
			t.Fatal(err)
		}
		if len(courses.events) != 1 {
			t.Fatalf("events after retry = %d, want 1", len(courses.events))
		}
	}
}

func TestRevertPublishingLeavesOtherStatusesAlone(t *testing.T) {
	a, courses := newActivities("ready")
	if err := a.RevertPublishing(context.Background(), RevertInput{CourseID: courses.course.ID, PreviousStatus: "draft", Reason: "x"}); err != nil {
		t.Fatal(err)
	}
	if courses.course.Status != "ready" || len(courses.events) != 0 {
		t.Fatalf("status = %s, events = %d; want untouched", courses.course.Status, len(courses.events))
	}
}

func TestRevertPublishingLostRace(t *testing.T) {
	a, courses := newActivities("publishing")
	courses.raceTo = "ready" // finalized by someone else meanwhile
	if err := a.RevertPublishing(context.Background(), RevertInput{CourseID: courses.course.ID, PreviousStatus: "draft", Reason: "x"}); err != nil {
		t.Fatal(err)
	}
	if courses.course.Status != "ready" || len(courses.events) != 0 {
		t.Fatalf("status = %s, events = %d; want untouched", courses.course.Status, len(courses.events))
	}
}

func TestStubActivitiesSucceed(t *testing.T) {
	a, courses := newActivities("publishing")
	ctx := context.Background()
	for name, fn := range map[string]func(context.Context, uuid.UUID) error{
		"UpdateSearchIndex": a.UpdateSearchIndex, "RefreshCache": a.RefreshCache, "InitAnalytics": a.InitAnalytics,
	} {
		if err := fn(ctx, courses.course.ID); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
}
