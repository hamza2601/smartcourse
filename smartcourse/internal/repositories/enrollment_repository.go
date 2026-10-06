package repositories

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"smartcourse/internal/models"
)

type EnrollmentRepository struct {
	DB *gorm.DB
}

func NewEnrollmentRepository(db *gorm.DB) *EnrollmentRepository {
	return &EnrollmentRepository{DB: db}
}

func (er *EnrollmentRepository) CreateEnrollment(ctx context.Context, enrollment *models.Enrollment) error {
	return er.DB.WithContext(ctx).Create(enrollment).Error
}

func (er *EnrollmentRepository) GetEnrollmentByID(ctx context.Context, id uuid.UUID) (*models.Enrollment, error) {
	var enrollment models.Enrollment
	if err := er.DB.WithContext(ctx).Where("id = ?", id).First(&enrollment).Error; err != nil {
		return nil, err
	}
	return &enrollment, nil
}

func (er *EnrollmentRepository) GetEnrollmentByStudentCourse(ctx context.Context, studentID, courseID uuid.UUID) (*models.Enrollment, error) {
	var enrollment models.Enrollment
	if err := er.DB.WithContext(ctx).
		Where("student_id = ? AND course_id = ?", studentID, courseID).
		Order("created_at desc").
		First(&enrollment).Error; err != nil {
		return nil, err
	}
	return &enrollment, nil
}
