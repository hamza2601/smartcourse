package repositories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	apperrors "smartcourse/internal/errors"
	"smartcourse/internal/models"
)

type EnrollmentRepository struct {
	DB *gorm.DB
}

func NewEnrollmentRepository(db *gorm.DB) *EnrollmentRepository {
	return &EnrollmentRepository{DB: db}
}

func (er *EnrollmentRepository) CreateEnrollment(ctx context.Context, enrollment *models.Enrollment) error {
	if err := er.DB.WithContext(ctx).Create(enrollment).Error; err != nil {
		if isDuplicateKey(err) {
			return &apperrors.EnrollmentAlreadyExistsError{
				StudentID: enrollment.StudentID,
				CourseID:  enrollment.CourseID,
			}
		}
		return &apperrors.DatabaseError{Operation: "create enrollment", Err: err}
	}
	return nil
}

func (er *EnrollmentRepository) GetEnrollmentByID(ctx context.Context, id uuid.UUID) (*models.Enrollment, error) {
	var enrollment models.Enrollment
	if err := er.DB.WithContext(ctx).Where("id = ?", id).First(&enrollment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &apperrors.EnrollmentNotFoundError{EnrollmentID: id}
		}
		return nil, &apperrors.DatabaseError{Operation: "get enrollment by id", Err: err}
	}
	return &enrollment, nil
}

// GetEnrollmentByStudentCourse returns (nil, nil) when no enrollment exists,
// since "not enrolled yet" is an expected outcome for callers, not a failure.
func (er *EnrollmentRepository) GetEnrollmentByStudentCourse(ctx context.Context, studentID, courseID uuid.UUID) (*models.Enrollment, error) {
	var enrollment models.Enrollment
	if err := er.DB.WithContext(ctx).
		Where("student_id = ? AND course_id = ?", studentID, courseID).
		Order("created_at desc").
		First(&enrollment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, &apperrors.DatabaseError{Operation: "get enrollment by student and course", Err: err}
	}
	return &enrollment, nil
}
