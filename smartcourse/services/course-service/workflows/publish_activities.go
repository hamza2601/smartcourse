package workflows

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"

	"smartcourse/libs/common/outbox"
	"smartcourse/services/course-service/domain"
)

// verifyConcurrency bounds how many lessons VerifyAssets checks at once.
const verifyConcurrency = 4

// CourseStore is the part of the course repository the activities use.
type CourseStore interface {
	GetCourseByID(ctx context.Context, id uuid.UUID) (*domain.Course, error)
	UpdateCourseStatus(ctx context.Context, id uuid.UUID, from, to string, event outbox.Event) (bool, error)
	TransitionStatus(ctx context.Context, id uuid.UUID, from, to string) (bool, error)
}

// ContentStore is the part of the content repository the activities use.
type ContentStore interface {
	GetCourseContent(ctx context.Context, courseID uuid.UUID) ([]domain.ModuleWithLessons, error)
}

// RevertInput is the input of RevertPublishing.
type RevertInput struct {
	CourseID       uuid.UUID `json:"course_id"`
	PreviousStatus string    `json:"previous_status"`
	Reason         string    `json:"reason"`
}

// PublishActivities holds the activities of PublishCourseWorkflow. They are
// idempotent, because Temporal may run an activity more than once.
type PublishActivities struct {
	courses CourseStore
	content ContentStore
	log     *slog.Logger
}

func NewPublishActivities(courses CourseStore, content ContentStore, log *slog.Logger) *PublishActivities {
	return &PublishActivities{courses: courses, content: content, log: log}
}

// BeginPublishing moves the course from its previous status to publishing.
// A course already publishing counts as done (a retry of this activity).
func (a *PublishActivities) BeginPublishing(ctx context.Context, in PublishCourseInput) error {
	course, err := a.courses.GetCourseByID(ctx, in.CourseID)
	if err != nil {
		return err
	}
	if course.IsPublishing() {
		return nil
	}
	if course.Status != in.PreviousStatus || !domain.CanTransition(course.Status, domain.CourseStatusPublishing) {
		return statusConflict(in.CourseID, course.Status, "cannot start publishing")
	}

	moved, err := a.courses.TransitionStatus(ctx, in.CourseID, in.PreviousStatus, domain.CourseStatusPublishing)
	if err != nil {
		return err
	}
	if moved {
		return nil
	}
	// Lost a race: see where the course is now.
	current, err := a.courses.GetCourseByID(ctx, in.CourseID)
	if err != nil {
		return err
	}
	if current.IsPublishing() {
		return nil
	}
	return statusConflict(in.CourseID, current.Status, "cannot start publishing")
}

// ValidateCourse checks the course is complete enough to publish and reports
// every problem it finds, not just the first.
func (a *PublishActivities) ValidateCourse(ctx context.Context, courseID uuid.UUID) error {
	course, err := a.courses.GetCourseByID(ctx, courseID)
	if err != nil {
		return err
	}
	modules, err := a.content.GetCourseContent(ctx, courseID)
	if err != nil {
		return err
	}

	var problems []string
	if strings.TrimSpace(course.Description) == "" {
		problems = append(problems, "course description is empty")
	}
	for _, m := range modules {
		if len(m.Lessons) == 0 {
			problems = append(problems, fmt.Sprintf("module %q has no lessons", m.Title))
		}
		for _, l := range m.Lessons {
			if strings.TrimSpace(l.Title) == "" {
				problems = append(problems, fmt.Sprintf("module %q has a lesson with an empty title", m.Title))
			}
		}
	}
	if len(problems) > 0 {
		return temporal.NewNonRetryableApplicationError(
			"course validation failed: "+strings.Join(problems, "; "), errTypeValidation, nil)
	}
	return nil
}

// VerifyAssets checks that every lesson has content. All lessons are checked
// (a bounded number at a time) and every empty one is reported, ordered by
// module position and then lesson position. ctx only stops the work early when
// the activity itself is cancelled.
func (a *PublishActivities) VerifyAssets(ctx context.Context, courseID uuid.UUID) error {
	modules, err := a.content.GetCourseContent(ctx, courseID)
	if err != nil {
		return err
	}

	type lessonRef struct {
		modulePos, lessonPos int
		title, content       string
	}
	var lessons []lessonRef
	for _, m := range modules {
		for _, l := range m.Lessons {
			lessons = append(lessons, lessonRef{m.Position, l.Position, l.Title, l.Content})
		}
	}

	type failure struct {
		modulePos, lessonPos int
		title                string
	}
	failures := make(chan failure, len(lessons))
	sem := make(chan struct{}, verifyConcurrency)
	var wg sync.WaitGroup
	for _, l := range lessons {
		if ctx.Err() != nil {
			break
		}
		sem <- struct{}{}
		wg.Add(1)
		go func(l lessonRef) {
			defer wg.Done()
			defer func() { <-sem }()
			if ctx.Err() != nil {
				return
			}
			if strings.TrimSpace(l.content) == "" {
				failures <- failure{l.modulePos, l.lessonPos, l.title}
			}
		}(l)
	}
	wg.Wait()
	close(failures)

	if err := ctx.Err(); err != nil {
		return err
	}

	var failed []failure
	for f := range failures {
		failed = append(failed, f)
	}
	if len(failed) == 0 {
		return nil
	}
	sort.Slice(failed, func(i, j int) bool {
		if failed[i].modulePos != failed[j].modulePos {
			return failed[i].modulePos < failed[j].modulePos
		}
		return failed[i].lessonPos < failed[j].lessonPos
	})
	titles := make([]string, len(failed))
	for i, f := range failed {
		titles[i] = fmt.Sprintf("%q", f.title)
	}
	return temporal.NewNonRetryableApplicationError(
		"lessons with empty content: "+strings.Join(titles, ", "), errTypeValidation, nil)
}

// UpdateSearchIndex is a stub: there is no search index yet.
func (a *PublishActivities) UpdateSearchIndex(ctx context.Context, courseID uuid.UUID) error {
	a.log.Info("publish step: update search index (stub)", "course_id", courseID)
	return nil
}

// RefreshCache is a stub: there is no cache yet.
func (a *PublishActivities) RefreshCache(ctx context.Context, courseID uuid.UUID) error {
	a.log.Info("publish step: refresh cache (stub)", "course_id", courseID)
	return nil
}

// InitAnalytics is a stub: there is no analytics store yet.
func (a *PublishActivities) InitAnalytics(ctx context.Context, courseID uuid.UUID) error {
	a.log.Info("publish step: init analytics (stub)", "course_id", courseID)
	return nil
}

// FinalizePublishing moves the course from publishing to ready and records the
// CoursePublished event in the same transaction. A course already ready counts
// as done (a retry of this activity).
func (a *PublishActivities) FinalizePublishing(ctx context.Context, courseID uuid.UUID) error {
	course, err := a.courses.GetCourseByID(ctx, courseID)
	if err != nil {
		return err
	}
	switch course.Status {
	case domain.CourseStatusReady:
		return nil
	case domain.CourseStatusPublishing:
	default:
		return statusConflict(courseID, course.Status, "cannot finalize publishing")
	}

	event, err := outbox.NewEvent(domain.AggregateTypeCourse, course.ID, domain.EventCoursePublished, domain.CoursePublishedPayload{
		CourseID:        course.ID,
		InstructorID:    course.InstructorID,
		Name:            course.Name,
		EnrollmentLimit: course.EnrollmentLimit,
	})
	if err != nil {
		return err
	}
	moved, err := a.courses.UpdateCourseStatus(ctx, courseID, domain.CourseStatusPublishing, domain.CourseStatusReady, event)
	if err != nil {
		return err
	}
	if moved {
		return nil
	}
	current, err := a.courses.GetCourseByID(ctx, courseID)
	if err != nil {
		return err
	}
	if current.IsReady() {
		return nil
	}
	return statusConflict(courseID, current.Status, "cannot finalize publishing")
}

// RevertPublishing moves the course from publishing back to its previous status
// and records a CoursePublishFailed event in the same transaction. A course no
// longer publishing is left alone (a retry of this activity).
func (a *PublishActivities) RevertPublishing(ctx context.Context, in RevertInput) error {
	course, err := a.courses.GetCourseByID(ctx, in.CourseID)
	if err != nil {
		return err
	}
	if !course.IsPublishing() {
		return nil
	}

	event, err := outbox.NewEvent(domain.AggregateTypeCourse, course.ID, domain.EventCoursePublishFailed, domain.CoursePublishFailedPayload{
		CourseID:     course.ID,
		InstructorID: course.InstructorID,
		Reason:       in.Reason,
	})
	if err != nil {
		return err
	}
	moved, err := a.courses.UpdateCourseStatus(ctx, in.CourseID, domain.CourseStatusPublishing, in.PreviousStatus, event)
	if err != nil {
		return err
	}
	if moved {
		return nil
	}
	current, err := a.courses.GetCourseByID(ctx, in.CourseID)
	if err != nil {
		return err
	}
	if !current.IsPublishing() {
		return nil
	}
	return fmt.Errorf("course %s is still publishing after a lost update; retry", in.CourseID)
}

func statusConflict(courseID uuid.UUID, status, what string) error {
	return temporal.NewNonRetryableApplicationError(
		fmt.Sprintf("%s: course %s is %s", what, courseID, status), errTypeStatusConflict, nil)
}
