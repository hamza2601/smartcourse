package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	apperrors "smartcourse/libs/common/errors"
	"smartcourse/libs/common/outbox"
	"smartcourse/services/course-service/domain"
)

// CourseStore is satisfied by *repositories.CourseRepository; it is an
// interface so the service can be tested without a database. Writes take the
// outbox event to store in the same transaction.
type CourseStore interface {
	CreateCourse(ctx context.Context, course *domain.Course, event outbox.Event) error
	GetCourseByID(ctx context.Context, id uuid.UUID) (*domain.Course, error)
	ListAllCourses(ctx context.Context) ([]domain.Course, error)
	// UpdateCourse changes the non-nil fields, but only while the course is a
	// draft or ready; otherwise it returns CourseNotEditableError (or
	// CourseNotFoundError).
	UpdateCourse(ctx context.Context, id uuid.UUID, name, description *string, enrollmentLimit *int) (*domain.Course, error)
}

// PublishStarter starts the publish workflow for a course; it is satisfied by
// *workflows.PublishStarter.
type PublishStarter interface {
	StartPublish(ctx context.Context, courseID uuid.UUID, previousStatus string) error
}

type CourseService struct {
	Repo    CourseStore
	Starter PublishStarter
}

func NewCourseService(repo CourseStore, starter PublishStarter) *CourseService {
	return &CourseService{Repo: repo, Starter: starter}
}

// CreateCourse creates a draft course owned by instructorID, which must come
// from the caller's verified token (the route requires the instructor role).
func (cs *CourseService) CreateCourse(ctx context.Context, instructorID uuid.UUID, name, description string, enrollmentLimit *int) (*domain.Course, error) {
	if instructorID == uuid.Nil {
		return nil, fmt.Errorf("instructor ID must be provided")
	}
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("course name cannot be empty")
	}
	if enrollmentLimit != nil && *enrollmentLimit <= 0 {
		return nil, fmt.Errorf("enrollment limit must be greater than 0")
	}

	course := &domain.Course{
		ID:              uuid.New(),
		Name:            name,
		Description:     description,
		InstructorID:    instructorID,
		Status:          domain.CourseStatusDraft,
		EnrollmentLimit: enrollmentLimit,
	}
	event, err := outbox.NewEvent(domain.AggregateTypeCourse, course.ID, domain.EventCourseCreated, domain.CourseCreatedPayload{
		CourseID:        course.ID,
		InstructorID:    course.InstructorID,
		Name:            course.Name,
		Status:          course.Status,
		EnrollmentLimit: course.EnrollmentLimit,
	})
	if err != nil {
		return nil, err
	}
	if err := cs.Repo.CreateCourse(ctx, course, event); err != nil {
		return nil, err
	}
	return course, nil
}

// PublishCourse starts the publish workflow for a draft or ready course; the
// workflow does the rest asynchronously. Only the course's instructor may do
// this. A course that is publishing (or archived) cannot be published.
func (cs *CourseService) PublishCourse(ctx context.Context, courseID, callerID uuid.UUID) error {
	course, err := cs.Repo.GetCourseByID(ctx, courseID)
	if err != nil {
		return err
	}
	if !course.IsOwnedBy(callerID) {
		return &apperrors.CourseNotOwnedError{CourseID: courseID}
	}
	if course.Status != domain.CourseStatusDraft && course.Status != domain.CourseStatusReady {
		return &apperrors.CourseStatusConflictError{CourseID: courseID, Status: course.Status}
	}
	return cs.Starter.StartPublish(ctx, courseID, course.Status)
}

// UpdateCourse changes the given fields of a draft or ready course; nil fields
// are left unchanged. Only the course's instructor may do this.
func (cs *CourseService) UpdateCourse(ctx context.Context, courseID, callerID uuid.UUID, name, description *string, enrollmentLimit *int) (*domain.Course, error) {
	course, err := cs.Repo.GetCourseByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if !course.IsOwnedBy(callerID) {
		return nil, &apperrors.CourseNotOwnedError{CourseID: courseID}
	}
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return nil, fmt.Errorf("course name cannot be empty")
		}
		name = &trimmed
	}
	if enrollmentLimit != nil && *enrollmentLimit <= 0 {
		return nil, fmt.Errorf("enrollment limit must be greater than 0")
	}
	if course.Status != domain.CourseStatusDraft && course.Status != domain.CourseStatusReady {
		return nil, &apperrors.CourseNotEditableError{CourseID: courseID, Status: course.Status}
	}
	return cs.Repo.UpdateCourse(ctx, courseID, name, description, enrollmentLimit)
}

func (cs *CourseService) GetCourse(ctx context.Context, id uuid.UUID) (*domain.Course, error) {
	return cs.Repo.GetCourseByID(ctx, id)
}

func (cs *CourseService) ListCourses(ctx context.Context) ([]domain.Course, error) {
	return cs.Repo.ListAllCourses(ctx)
}
