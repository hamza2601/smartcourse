package errors

import (
	"fmt"

	"github.com/google/uuid"
)

type CourseNotFoundError struct {
	CourseID uuid.UUID
}

func (e *CourseNotFoundError) Error() string {
	return fmt.Sprintf("course not found: %s", e.CourseID)
}

// CourseNotPublishedError: the course exists but is not open for enrollment.
type CourseNotPublishedError struct {
	CourseID uuid.UUID
}

func (e *CourseNotPublishedError) Error() string {
	return fmt.Sprintf("course %s is not published", e.CourseID)
}

// CourseFullError: every seat allowed by the course's enrollment limit is taken.
type CourseFullError struct {
	CourseID uuid.UUID
}

func (e *CourseFullError) Error() string {
	return fmt.Sprintf("course %s is full", e.CourseID)
}

// CourseNotOwnedError: the caller is not the course's instructor.
type CourseNotOwnedError struct {
	CourseID uuid.UUID
}

func (e *CourseNotOwnedError) Error() string {
	return fmt.Sprintf("course %s is not owned by the caller", e.CourseID)
}

// CourseStatusConflictError: the course is not in a state that allows the
// requested transition (e.g. publishing a course that is not a draft).
type CourseStatusConflictError struct {
	CourseID uuid.UUID
	Status   string
}

func (e *CourseStatusConflictError) Error() string {
	return fmt.Sprintf("course %s is %s; only draft courses can be published", e.CourseID, e.Status)
}

// ModuleNotFoundError: no module with this ID exists, or the caller may not see it.
type ModuleNotFoundError struct {
	ModuleID uuid.UUID
}

func (e *ModuleNotFoundError) Error() string {
	return fmt.Sprintf("module not found: %s", e.ModuleID)
}

// LessonNotFoundError: no lesson with this ID exists, or the caller may not see it.
type LessonNotFoundError struct {
	LessonID uuid.UUID
}

func (e *LessonNotFoundError) Error() string {
	return fmt.Sprintf("lesson not found: %s", e.LessonID)
}

// CourseNotEditableError: a course's modules and lessons can only be changed
// while the course is a draft.
type CourseNotEditableError struct {
	CourseID uuid.UUID
	Status   string
}

func (e *CourseNotEditableError) Error() string {
	return fmt.Sprintf("course %s is %s; its content can only be changed while it is a draft", e.CourseID, e.Status)
}

// PositionConflictError: another module (in the course) or lesson (in the
// module) already uses this position.
type PositionConflictError struct {
	Position int
}

func (e *PositionConflictError) Error() string {
	return fmt.Sprintf("position %d is already in use", e.Position)
}
