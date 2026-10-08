package repositories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"smartcourse/internal/domain"
	apperrors "smartcourse/internal/errors"
	"smartcourse/internal/mappers"
	"smartcourse/internal/persistence"
)

type EnrollmentRepository struct {
	DB *gorm.DB
}

func NewEnrollmentRepository(db *gorm.DB) *EnrollmentRepository {
	return &EnrollmentRepository{DB: db}
}

func (er *EnrollmentRepository) CreateEnrollment(ctx context.Context, enrollment *domain.Enrollment) error {
	enrollmentDB := mappers.EnrollmentToPersistence(enrollment)
	if err := er.DB.WithContext(ctx).Create(enrollmentDB).Error; err != nil {
		if isDuplicateKey(err) {
			return &apperrors.EnrollmentAlreadyExistsError{
				StudentID: enrollment.StudentID,
				CourseID:  enrollment.CourseID,
			}
		}
		return &apperrors.DatabaseError{Operation: "create enrollment", Err: err}
	}
	*enrollment = *mappers.EnrollmentDBToDomain(enrollmentDB)
	return nil
}

func (er *EnrollmentRepository) GetEnrollmentByID(ctx context.Context, id uuid.UUID) (*domain.Enrollment, error) {
	var enrollmentDB persistence.EnrollmentDB
	if err := er.DB.WithContext(ctx).Where("id = ?", id).First(&enrollmentDB).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &apperrors.EnrollmentNotFoundError{EnrollmentID: id}
		}
		return nil, &apperrors.DatabaseError{Operation: "get enrollment by id", Err: err}
	}
	return mappers.EnrollmentDBToDomain(&enrollmentDB), nil
}

// GetEnrollmentByStudentCourse returns (nil, nil) when no enrollment exists,
// since "not enrolled yet" is an expected outcome for callers, not a failure.
func (er *EnrollmentRepository) GetEnrollmentByStudentCourse(ctx context.Context, studentID, courseID uuid.UUID) (*domain.Enrollment, error) {
	var enrollmentDB persistence.EnrollmentDB
	if err := er.DB.WithContext(ctx).
		Where("student_id = ? AND course_id = ?", studentID, courseID).
		Order("created_at desc").
		First(&enrollmentDB).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, &apperrors.DatabaseError{Operation: "get enrollment by student and course", Err: err}
	}
	return mappers.EnrollmentDBToDomain(&enrollmentDB), nil
}
