package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"smartcourse/internal/models"
	"smartcourse/internal/repositories"
)

type CourseService struct {
	CourseRepo *repositories.CourseRepository
	UserRepo   *repositories.UserRepository
}

func NewCourseService(
	courseRepo *repositories.CourseRepository,
	userRepo *repositories.UserRepository,
) *CourseService {
	return &CourseService{
		CourseRepo: courseRepo,
		UserRepo:   userRepo,
	}
}

func (cs *CourseService) CreateCourse(ctx context.Context, name string, instructorID uuid.UUID) (*models.Course, error) {
	if name == "" {
		return nil, fmt.Errorf("course name cannot be empty")
	}
	if instructorID == uuid.Nil {
		return nil, fmt.Errorf("instructor ID must be provided")
	}

	instructor, err := cs.UserRepo.GetUserByID(ctx, instructorID)
	if err != nil {
		return nil, fmt.Errorf("instructor not found")
	}

	if instructor.Role != "instructor" {
		return nil, fmt.Errorf("user is not an instructor")
	}

	course := &models.Course{
		Name:         name,
		InstructorID: instructorID,
		Status:       "draft",
	}
	if err := cs.CourseRepo.CreateCourse(ctx, course); err != nil {
		return nil, err
	}
	return course, nil
}

func (cs *CourseService) GetCourse(ctx context.Context, id uuid.UUID) (*models.Course, error) {
	return cs.CourseRepo.GetCourseByID(ctx, id)
}

func (cs *CourseService) ListCourses(ctx context.Context) ([]models.Course, error) {
	return cs.CourseRepo.ListAllCourses(ctx)
}
