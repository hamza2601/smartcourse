package repositories

import (
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

func (cr *CourseRepository) CreateCourse(course *models.Course) error {
	course.ID = uuid.New()
	return cr.DB.Create(course).Error
}

func (cr *CourseRepository) GetCourseByID(id uuid.UUID) (*models.Course, error) {
	var course models.Course
	if err := cr.DB.Where("id = ?", id).First(&course).Error; err != nil {
		return nil, err
	}
	return &course, nil
}

func (cr *CourseRepository) GetCoursesByInstructor(instructorID uuid.UUID) ([]models.Course, error) {
	var courses []models.Course
	if err := cr.DB.Where("instructor_id = ?", instructorID).Find(&courses).Error; err != nil {
		return nil, err
	}
	return courses, nil
}

func (cr *CourseRepository) ListAllCourses() ([]models.Course, error) {
	var courses []models.Course
	if err := cr.DB.Find(&courses).Error; err != nil {
		return nil, err
	}
	return courses, nil
}
