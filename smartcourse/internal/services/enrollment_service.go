package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	apperrors "smartcourse/internal/errors"
	"smartcourse/internal/models"
	"smartcourse/internal/repositories"
)

type EnrollmentService struct {
	Repo *repositories.EnrollmentRepository
}

func NewEnrollmentService(repo *repositories.EnrollmentRepository) *EnrollmentService {
	return &EnrollmentService{Repo: repo}
}

func (es *EnrollmentService) EnrollStudent(ctx context.Context, studentID, courseID uuid.UUID) (*models.Enrollment, error) {
	if studentID == uuid.Nil {
		return nil, fmt.Errorf("student ID must be provided")
	}
	if courseID == uuid.Nil {
		return nil, fmt.Errorf("course ID must be provided")
	}

	existing, err := es.Repo.GetEnrollmentByStudentCourse(ctx, studentID, courseID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.Status == "active" {
		return nil, &apperrors.EnrollmentAlreadyExistsError{StudentID: studentID, CourseID: courseID}
	}

	enrollment := &models.Enrollment{
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

func (es *EnrollmentService) GetEnrollment(ctx context.Context, id uuid.UUID) (*models.Enrollment, error) {
	return es.Repo.GetEnrollmentByID(ctx, id)
}
