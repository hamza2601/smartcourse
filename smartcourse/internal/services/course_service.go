package services

import (
	"fmt"

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

func (cs *CourseService) CreateCourse(name string, instructorID uint) (*models.Course, error) {
	if name == "" {
		return nil, fmt.Errorf("course name cannot be empty")
	}
	if instructorID == 0 {
		return nil, fmt.Errorf("instructor ID must be provided")
	}

	instructor, err := cs.UserRepo.GetUserByID(instructorID)
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
	if err := cs.CourseRepo.CreateCourse(course); err != nil {
		return nil, err
	}
	return course, nil
}

func (cs *CourseService) GetCourse(id uint) (*models.Course, error) {
	return cs.CourseRepo.GetCourseByID(id)
}

func (cs *CourseService) ListCourses() ([]models.Course, error) {
	return cs.CourseRepo.ListAllCourses()
}
