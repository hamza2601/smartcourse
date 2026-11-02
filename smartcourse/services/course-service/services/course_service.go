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
	// UpdateCourseStatus moves a course from one status to another and
	// reports false (writing no event) if the course was not in status `from`.
	UpdateCourseStatus(ctx context.Context, id uuid.UUID, from, to string, event outbox.Event) (bool, error)
}

type CourseService struct {
	Repo CourseStore
}

func NewCourseService(repo CourseStore) *CourseService {
	return &CourseService{Repo: repo}
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

// PublishCourse opens a draft course for enrollment. Only its instructor may
// publish it.
func (cs *CourseService) PublishCourse(ctx context.Context, courseID, callerID uuid.UUID) (*domain.Course, error) {
	course, err := cs.Repo.GetCourseByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if !course.IsOwnedBy(callerID) {
		return nil, &apperrors.CourseNotOwnedError{CourseID: courseID}
	}
	if !course.IsDraft() {
		return nil, &apperrors.CourseStatusConflictError{CourseID: courseID, Status: course.Status}
	}

	event, err := outbox.NewEvent(domain.AggregateTypeCourse, course.ID, domain.EventCoursePublished, domain.CoursePublishedPayload{
		CourseID:        course.ID,
		InstructorID:    course.InstructorID,
		Name:            course.Name,
		EnrollmentLimit: course.EnrollmentLimit,
	})
	if err != nil {
		return nil, err
	}

	updated, err := cs.Repo.UpdateCourseStatus(ctx, courseID, domain.CourseStatusDraft, domain.CourseStatusPublished, event)
	if err != nil {
		return nil, err
	}
	if !updated {
		// Status changed between the read and the conditional update.
		current, err := cs.Repo.GetCourseByID(ctx, courseID)
		if err != nil {
			return nil, err
		}
		return nil, &apperrors.CourseStatusConflictError{CourseID: courseID, Status: current.Status}
	}
	return cs.Repo.GetCourseByID(ctx, courseID)
}

func (cs *CourseService) GetCourse(ctx context.Context, id uuid.UUID) (*domain.Course, error) {
	return cs.Repo.GetCourseByID(ctx, id)
}

func (cs *CourseService) ListCourses(ctx context.Context) ([]domain.Course, error) {
	return cs.Repo.ListAllCourses(ctx)
}
