package services

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	apperrors "smartcourse/libs/common/errors"
	"smartcourse/services/course-service/domain"
)

// maxTitleLength matches the VARCHAR(255) columns; longer titles are rejected
// here with a validation error instead of failing in Postgres.
const maxTitleLength = 255

// ContentStore is satisfied by *repositories.ContentRepository; it is an
// interface so the service can be tested without a database. A zero Position
// on create means "omitted": the store assigns the next free position.
type ContentStore interface {
	CreateModule(ctx context.Context, module *domain.Module) error
	ListModules(ctx context.Context, courseID uuid.UUID) ([]domain.Module, error)
	GetModuleWithCourse(ctx context.Context, moduleID uuid.UUID) (*domain.Module, *domain.Course, error)
	UpdateModule(ctx context.Context, module *domain.Module) error
	DeleteModule(ctx context.Context, moduleID uuid.UUID) error

	CreateLesson(ctx context.Context, lesson *domain.Lesson) error
	ListLessons(ctx context.Context, moduleID uuid.UUID) ([]domain.Lesson, error)
	GetLessonWithCourse(ctx context.Context, lessonID uuid.UUID) (*domain.Lesson, *domain.Course, error)
	UpdateLesson(ctx context.Context, lesson *domain.Lesson) error
	DeleteLesson(ctx context.Context, lessonID uuid.UUID) error

	GetCourseContent(ctx context.Context, courseID uuid.UUID) ([]domain.ModuleWithLessons, error)
}

type ContentService struct {
	Courses CourseStore
	Content ContentStore
}

func NewContentService(courseStore CourseStore, contentStore ContentStore) *ContentService {
	return &ContentService{Courses: courseStore, Content: contentStore}
}

// CreateModule adds a module to a draft course owned by callerID. A nil
// position appends the module after the existing ones.
func (cs *ContentService) CreateModule(ctx context.Context, callerID, courseID uuid.UUID, title string, position *int) (*domain.Module, error) {
	if err := validateTitle(title); err != nil {
		return nil, err
	}
	if err := validatePosition(position); err != nil {
		return nil, err
	}

	course, err := cs.Courses.GetCourseByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if err := requireEditable(course, callerID); err != nil {
		return nil, err
	}

	module := &domain.Module{ID: uuid.New(), CourseID: courseID, Title: title}
	if position != nil {
		module.Position = *position
	}
	if err := cs.Content.CreateModule(ctx, module); err != nil {
		return nil, err
	}
	return module, nil
}

func (cs *ContentService) ListModules(ctx context.Context, callerID, courseID uuid.UUID) ([]domain.Module, error) {
	course, err := cs.Courses.GetCourseByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if !canRead(course, callerID) {
		return nil, &apperrors.CourseNotFoundError{CourseID: courseID}
	}
	return cs.Content.ListModules(ctx, courseID)
}

func (cs *ContentService) GetCourseContent(ctx context.Context, callerID, courseID uuid.UUID) ([]domain.ModuleWithLessons, error) {
	course, err := cs.Courses.GetCourseByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if !canRead(course, callerID) {
		return nil, &apperrors.CourseNotFoundError{CourseID: courseID}
	}
	return cs.Content.GetCourseContent(ctx, courseID)
}

// UpdateModule changes the title and/or position; nil fields are left as is.
func (cs *ContentService) UpdateModule(ctx context.Context, callerID, moduleID uuid.UUID, title *string, position *int) (*domain.Module, error) {
	if title != nil {
		if err := validateTitle(*title); err != nil {
			return nil, err
		}
	}
	if err := validatePosition(position); err != nil {
		return nil, err
	}

	module, course, err := cs.Content.GetModuleWithCourse(ctx, moduleID)
	if err != nil {
		return nil, err
	}
	if err := requireEditable(course, callerID); err != nil {
		return nil, err
	}

	if title != nil {
		module.Title = *title
	}
	if position != nil {
		module.Position = *position
	}
	if err := cs.Content.UpdateModule(ctx, module); err != nil {
		return nil, err
	}
	return module, nil
}

func (cs *ContentService) DeleteModule(ctx context.Context, callerID, moduleID uuid.UUID) error {
	_, course, err := cs.Content.GetModuleWithCourse(ctx, moduleID)
	if err != nil {
		return err
	}
	if err := requireEditable(course, callerID); err != nil {
		return err
	}
	return cs.Content.DeleteModule(ctx, moduleID)
}

// CreateLesson adds a lesson to a module of a draft course owned by callerID.
// A nil position appends the lesson after the existing ones.
func (cs *ContentService) CreateLesson(ctx context.Context, callerID, moduleID uuid.UUID, title, content string, position *int) (*domain.Lesson, error) {
	if err := validateTitle(title); err != nil {
		return nil, err
	}
	if err := validatePosition(position); err != nil {
		return nil, err
	}

	_, course, err := cs.Content.GetModuleWithCourse(ctx, moduleID)
	if err != nil {
		return nil, err
	}
	if err := requireEditable(course, callerID); err != nil {
		return nil, err
	}

	lesson := &domain.Lesson{ID: uuid.New(), ModuleID: moduleID, Title: title, Content: content}
	if position != nil {
		lesson.Position = *position
	}
	if err := cs.Content.CreateLesson(ctx, lesson); err != nil {
		return nil, err
	}
	return lesson, nil
}

func (cs *ContentService) ListLessons(ctx context.Context, callerID, moduleID uuid.UUID) ([]domain.Lesson, error) {
	_, course, err := cs.Content.GetModuleWithCourse(ctx, moduleID)
	if err != nil {
		return nil, err
	}
	if !canRead(course, callerID) {
		return nil, &apperrors.ModuleNotFoundError{ModuleID: moduleID}
	}
	return cs.Content.ListLessons(ctx, moduleID)
}

// UpdateLesson changes the title, content and/or position; nil fields are left as is.
func (cs *ContentService) UpdateLesson(ctx context.Context, callerID, lessonID uuid.UUID, title, content *string, position *int) (*domain.Lesson, error) {
	if title != nil {
		if err := validateTitle(*title); err != nil {
			return nil, err
		}
	}
	if err := validatePosition(position); err != nil {
		return nil, err
	}

	lesson, course, err := cs.Content.GetLessonWithCourse(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	if err := requireEditable(course, callerID); err != nil {
		return nil, err
	}

	if title != nil {
		lesson.Title = *title
	}
	if content != nil {
		lesson.Content = *content
	}
	if position != nil {
		lesson.Position = *position
	}
	if err := cs.Content.UpdateLesson(ctx, lesson); err != nil {
		return nil, err
	}
	return lesson, nil
}

func (cs *ContentService) DeleteLesson(ctx context.Context, callerID, lessonID uuid.UUID) error {
	_, course, err := cs.Content.GetLessonWithCourse(ctx, lessonID)
	if err != nil {
		return err
	}
	if err := requireEditable(course, callerID); err != nil {
		return err
	}
	return cs.Content.DeleteLesson(ctx, lessonID)
}

// requireEditable allows a write only for the course's instructor, and only
// while the course is a draft.
func requireEditable(course *domain.Course, callerID uuid.UUID) error {
	if !course.IsOwnedBy(callerID) {
		return &apperrors.CourseNotOwnedError{CourseID: course.ID}
	}
	if !course.IsDraft() {
		return &apperrors.CourseNotEditableError{CourseID: course.ID, Status: course.Status}
	}
	return nil
}

// canRead lets the instructor see the content in any state, everyone else
// only once the course is published.
func canRead(course *domain.Course, callerID uuid.UUID) bool {
	return course.IsOwnedBy(callerID) || course.IsPublished()
}

func validateTitle(title string) error {
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("title cannot be empty")
	}
	if utf8.RuneCountInString(title) > maxTitleLength {
		return fmt.Errorf("title must be at most %d characters", maxTitleLength)
	}
	return nil
}

func validatePosition(position *int) error {
	if position != nil && *position < 1 {
		return fmt.Errorf("position must be at least 1")
	}
	return nil
}
