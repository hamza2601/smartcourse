package services

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	apperrors "smartcourse/libs/common/errors"
	"smartcourse/libs/common/httpclient"
	"smartcourse/services/enrollment-service/domain"
)

// EnrollmentStore is satisfied by *repositories.EnrollmentRepository; it is an
// interface so the service can be tested without a database.
type EnrollmentStore interface {
	CreateEnrollment(ctx context.Context, enrollment *domain.Enrollment) error
	GetEnrollmentByID(ctx context.Context, id uuid.UUID) (*domain.Enrollment, error)
	GetEnrollmentByStudentCourse(ctx context.Context, studentID, courseID uuid.UUID) (*domain.Enrollment, error)
}

type EnrollmentService struct {
	Repo             EnrollmentStore
	CourseServiceURL string
	UserServiceURL   string
	HTTPClient       *http.Client
}

func NewEnrollmentService(repo EnrollmentStore, courseServiceURL, userServiceURL string) *EnrollmentService {
	return &EnrollmentService{
		Repo:             repo,
		CourseServiceURL: courseServiceURL,
		UserServiceURL:   userServiceURL,
		HTTPClient:       httpclient.NewClient(),
	}
}

func (es *EnrollmentService) EnrollStudent(ctx context.Context, studentID, courseID uuid.UUID) (*domain.Enrollment, error) {
	if studentID == uuid.Nil {
		return nil, fmt.Errorf("student ID must be provided")
	}
	if courseID == uuid.Nil {
		return nil, fmt.Errorf("course ID must be provided")
	}

	if _, err := es.fetchCourse(ctx, courseID); err != nil {
		return nil, err
	}
	if _, err := es.fetchStudent(ctx, studentID); err != nil {
		return nil, err
	}

	existing, err := es.Repo.GetEnrollmentByStudentCourse(ctx, studentID, courseID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.Status == "active" {
		return nil, &apperrors.EnrollmentAlreadyExistsError{StudentID: studentID, CourseID: courseID}
	}

	enrollment := &domain.Enrollment{
		ID:        uuid.New(),
		StudentID: studentID,
		CourseID:  courseID,
		Status:    "active",
	}
	if err := es.Repo.CreateEnrollment(ctx, enrollment); err != nil {
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

func (es *EnrollmentService) fetchStudent(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	found, err := httpclient.FetchInternal(ctx, es.HTTPClient, userServiceName, fmt.Sprintf("%s/internal/users/%s", es.UserServiceURL, id), &user)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("student not found")
	}
	return &user, nil
}
