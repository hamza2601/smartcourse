package services

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"smartcourse/libs/common/httpclient"
	"smartcourse/services/course-service/domain"
	"smartcourse/services/course-service/repositories"
)

type CourseService struct {
	CourseRepo     *repositories.CourseRepository
	UserServiceURL string
	HTTPClient     *http.Client
}

func NewCourseService(courseRepo *repositories.CourseRepository, userServiceURL string) *CourseService {
	return &CourseService{
		CourseRepo:     courseRepo,
		UserServiceURL: userServiceURL,
		HTTPClient:     httpclient.NewClient(),
	}
}

func (cs *CourseService) CreateCourse(ctx context.Context, name string, instructorID uuid.UUID) (*domain.Course, error) {
	if name == "" {
		return nil, fmt.Errorf("course name cannot be empty")
	}
	if instructorID == uuid.Nil {
		return nil, fmt.Errorf("instructor ID must be provided")
	}

	instructor, err := cs.fetchUser(ctx, instructorID)
	if err != nil {
		return nil, err
	}

	if instructor.Role != "instructor" {
		return nil, fmt.Errorf("user is not an instructor")
	}

	course := &domain.Course{
		Name:         name,
		InstructorID: instructorID,
		Status:       "draft",
	}
	if err := cs.CourseRepo.CreateCourse(ctx, course); err != nil {
		return nil, err
	}
	return course, nil
}

func (cs *CourseService) GetCourse(ctx context.Context, id uuid.UUID) (*domain.Course, error) {
	return cs.CourseRepo.GetCourseByID(ctx, id)
}

func (cs *CourseService) ListCourses(ctx context.Context) ([]domain.Course, error) {
	return cs.CourseRepo.ListAllCourses(ctx)
}

func (cs *CourseService) fetchUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	found, err := httpclient.FetchInternal(ctx, cs.HTTPClient, userServiceName, fmt.Sprintf("%s/internal/users/%s", cs.UserServiceURL, id), &user)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("instructor not found")
	}
	return &user, nil
}
