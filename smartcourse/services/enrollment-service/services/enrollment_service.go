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
	// ListEnrollmentsByStudent returns all of a student's enrollments, newest first.
	ListEnrollmentsByStudent(ctx context.Context, studentID uuid.UUID) ([]domain.Enrollment, error)
	// CancelEnrollment, in one transaction: moves the enrollment from active to
	// cancelled (guarded by id, student and status), releases its seat and
	// stores the outbox event. On success it updates *enrollment.
	CancelEnrollment(ctx context.Context, enrollment *domain.Enrollment, event outbox.Event) error
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
	if !course.IsReady() {
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

// GetEnrollmentForCaller returns the enrollment only to its student or to an
// admin; anyone else gets EnrollmentNotFoundError, as if it did not exist.
func (es *EnrollmentService) GetEnrollmentForCaller(ctx context.Context, callerID uuid.UUID, callerRole string, id uuid.UUID) (*domain.Enrollment, error) {
	enrollment, err := es.Repo.GetEnrollmentByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if enrollment.StudentID != callerID && callerRole != domain.RoleAdmin {
		return nil, &apperrors.EnrollmentNotFoundError{EnrollmentID: id}
	}
	return enrollment, nil
}

// ListEnrollmentsForStudent returns the caller's own enrollments, newest first.
func (es *EnrollmentService) ListEnrollmentsForStudent(ctx context.Context, studentID uuid.UUID) ([]domain.Enrollment, error) {
	return es.Repo.ListEnrollmentsByStudent(ctx, studentID)
}

// CancelEnrollment cancels the caller's own active enrollment. Another
// student's enrollment is reported as not found; an enrollment that is not
// active (e.g. already cancelled) as EnrollmentNotActiveError.
func (es *EnrollmentService) CancelEnrollment(ctx context.Context, callerID, enrollmentID uuid.UUID) (*domain.Enrollment, error) {
	enrollment, err := es.Repo.GetEnrollmentByID(ctx, enrollmentID)
	if err != nil {
		return nil, err
	}
	if enrollment.StudentID != callerID {
		return nil, &apperrors.EnrollmentNotFoundError{EnrollmentID: enrollmentID}
	}
	if !enrollment.IsActive() {
		return nil, &apperrors.EnrollmentNotActiveError{EnrollmentID: enrollmentID, Status: enrollment.Status}
	}

	event, err := outbox.NewEvent(domain.AggregateTypeEnrollment, enrollment.ID, domain.EventEnrollmentCancelled, domain.EnrollmentCancelledPayload{
		EnrollmentID: enrollment.ID,
		StudentID:    enrollment.StudentID,
		CourseID:     enrollment.CourseID,
		Status:       domain.EnrollmentStatusCancelled,
	})
	if err != nil {
		return nil, err
	}
	// The store's guarded update has the final say if the enrollment changed
	// since it was read above.
	if err := es.Repo.CancelEnrollment(ctx, enrollment, event); err != nil {
		return nil, err
	}
	return enrollment, nil
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
