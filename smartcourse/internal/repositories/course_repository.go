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

type CourseRepository struct {
	DB *gorm.DB
}

func NewCourseRepository(db *gorm.DB) *CourseRepository {
	return &CourseRepository{DB: db}
}

func (cr *CourseRepository) CreateCourse(ctx context.Context, course *domain.Course) error {
	courseDB := mappers.CourseToPersistence(course)
	courseDB.ID = uuid.New()
	if err := cr.DB.WithContext(ctx).Create(courseDB).Error; err != nil {
		return &apperrors.DatabaseError{Operation: "create course", Err: err}
	}
	*course = *mappers.CourseDBToDomain(courseDB)
	return nil
}

func (cr *CourseRepository) GetCourseByID(ctx context.Context, id uuid.UUID) (*domain.Course, error) {
	var courseDB persistence.CourseDB
	if err := cr.DB.WithContext(ctx).Where("id = ?", id).First(&courseDB).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &apperrors.CourseNotFoundError{CourseID: id}
		}
		return nil, &apperrors.DatabaseError{Operation: "get course by id", Err: err}
	}
	return mappers.CourseDBToDomain(&courseDB), nil
}

func (cr *CourseRepository) GetCoursesByInstructor(ctx context.Context, instructorID uuid.UUID) ([]domain.Course, error) {
	var coursesDB []persistence.CourseDB
	if err := cr.DB.WithContext(ctx).Where("instructor_id = ?", instructorID).Find(&coursesDB).Error; err != nil {
		return nil, &apperrors.DatabaseError{Operation: "get courses by instructor", Err: err}
	}
	return coursesToDomain(coursesDB), nil
}

func (cr *CourseRepository) ListAllCourses(ctx context.Context) ([]domain.Course, error) {
	var coursesDB []persistence.CourseDB
	if err := cr.DB.WithContext(ctx).Find(&coursesDB).Error; err != nil {
		return nil, &apperrors.DatabaseError{Operation: "list courses", Err: err}
	}
	return coursesToDomain(coursesDB), nil
}

func coursesToDomain(coursesDB []persistence.CourseDB) []domain.Course {
	courses := make([]domain.Course, len(coursesDB))
	for i := range coursesDB {
		courses[i] = *mappers.CourseDBToDomain(&coursesDB[i])
	}
	return courses
}
