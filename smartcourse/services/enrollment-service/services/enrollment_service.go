package services

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	apperrors "smartcourse/libs/common/errors"
	"smartcourse/libs/common/httpclient"
	"smartcourse/libs/common/outbox"
	"smartcourse/services/enrollment-service/domain"
)

// EnrollmentStore is satisfied by *repositories.EnrollmentRepository; it is an
// interface so the service can be tested without a database.
type EnrollmentStore interface {
	GetEnrollmentByID(ctx context.Context, id uuid.UUID) (*domain.Enrollment, error)
	// Enroll, in one transaction: initialises the course's seat row from
	// capacity (nil = unlimited), rejects a second active enrollment, reserves
	// a seat, inserts the enrollment and its outbox event. Nothing is kept if
	// any step fails.
	Enroll(ctx context.Context, enrollment *domain.Enrollment, capacity *int, event outbox.Event) error
}

type EnrollmentService struct {
	Repo             EnrollmentStore
	CourseServiceURL string
	HTTPClient       *http.Client
}

func NewEnrollmentService(repo EnrollmentStore, courseServiceURL, internalAPIKey string) *EnrollmentService {
	return &EnrollmentService{
		Repo:             repo,
		CourseServiceURL: courseServiceURL,
		HTTPClient:       httpclient.NewClient(internalAPIKey),
	}
}

// EnrollStudent enrolls studentID, which must come from the caller's verified
// token, so the student needs no separate existence check.
func (es *EnrollmentService) EnrollStudent(ctx context.Context, studentID, courseID uuid.UUID) (*domain.Enrollment, error) {
	if studentID == uuid.Nil {
		return nil, fmt.Errorf("student ID must be provided")
	}
	if courseID == uuid.Nil {
		return nil, fmt.Errorf("course ID must be provided")
	}

	course, err := es.fetchCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if !course.IsPublished() {
		return nil, &apperrors.CourseNotPublishedError{CourseID: courseID}
	}

	enrollment := &domain.Enrollment{
		ID:        uuid.New(),
		StudentID: studentID,
		CourseID:  courseID,
		Status:    domain.EnrollmentStatusActive,
	}
	event, err := outbox.NewEvent(domain.AggregateTypeEnrollment, enrollment.ID, domain.EventEnrollmentCreated, domain.EnrollmentCreatedPayload{
		EnrollmentID: enrollment.ID,
		StudentID:    enrollment.StudentID,
		CourseID:     enrollment.CourseID,
		Status:       enrollment.Status,
	})
	if err != nil {
		return nil, err
	}
	if err := es.Repo.Enroll(ctx, enrollment, course.EnrollmentLimit, event); err != nil {
		return nil, err
	}
	return enrollment, nil
}

func (es *EnrollmentService) GetEnrollment(ctx context.Context, id uuid.UUID) (*domain.Enrollment, error) {
	return es.Repo.GetEnrollmentByID(ctx, id)
}

func (es *EnrollmentService) fetchCourse(ctx context.Context, id uuid.UUID) (*domain.Course, error) {
	var course domain.Course
	found, err := httpclient.FetchInternal(ctx, es.HTTPClient, courseServiceName, fmt.Sprintf("%s/internal/courses/%s", es.CourseServiceURL, id), &course)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("course not found")
	}
	return &course, nil
}
