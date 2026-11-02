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
