package repositories

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"smartcourse/internal/models"
)

type CourseRepository struct {
	DB *gorm.DB
}

func NewCourseRepository(db *gorm.DB) *CourseRepository {
	return &CourseRepository{DB: db}
}

func (cr *CourseRepository) CreateCourse(ctx context.Context, course *models.Course) error {
	course.ID = uuid.New()
	return cr.DB.WithContext(ctx).Create(course).Error
}

func (cr *CourseRepository) GetCourseByID(ctx context.Context, id uuid.UUID) (*models.Course, error) {
	var course models.Course
	if err := cr.DB.WithContext(ctx).Where("id = ?", id).First(&course).Error; err != nil {
		return nil, err
	}
	return &course, nil
}

func (cr *CourseRepository) GetCoursesByInstructor(ctx context.Context, instructorID uuid.UUID) ([]models.Course, error) {
	var courses []models.Course
	if err := cr.DB.WithContext(ctx).Where("instructor_id = ?", instructorID).Find(&courses).Error; err != nil {
		return nil, err
	}
	return courses, nil
}

func (cr *CourseRepository) ListAllCourses(ctx context.Context) ([]models.Course, error) {
	var courses []models.Course
	if err := cr.DB.WithContext(ctx).Find(&courses).Error; err != nil {
		return nil, err
	}
	return courses, nil
}
