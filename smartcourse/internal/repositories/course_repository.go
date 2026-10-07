package repositories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	apperrors "smartcourse/internal/errors"
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
	if err := cr.DB.WithContext(ctx).Create(course).Error; err != nil {
		return &apperrors.DatabaseError{Operation: "create course", Err: err}
	}
	return nil
}

func (cr *CourseRepository) GetCourseByID(ctx context.Context, id uuid.UUID) (*models.Course, error) {
	var course models.Course
	if err := cr.DB.WithContext(ctx).Where("id = ?", id).First(&course).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &apperrors.CourseNotFoundError{CourseID: id}
		}
		return nil, &apperrors.DatabaseError{Operation: "get course by id", Err: err}
	}
	return &course, nil
}

func (cr *CourseRepository) GetCoursesByInstructor(ctx context.Context, instructorID uuid.UUID) ([]models.Course, error) {
	var courses []models.Course
	if err := cr.DB.WithContext(ctx).Where("instructor_id = ?", instructorID).Find(&courses).Error; err != nil {
		return nil, &apperrors.DatabaseError{Operation: "get courses by instructor", Err: err}
	}
	return courses, nil
}

func (cr *CourseRepository) ListAllCourses(ctx context.Context) ([]models.Course, error) {
	var courses []models.Course
	if err := cr.DB.WithContext(ctx).Find(&courses).Error; err != nil {
		return nil, &apperrors.DatabaseError{Operation: "list courses", Err: err}
	}
	return courses, nil
}
