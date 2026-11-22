package errors

import (
	"fmt"

	"github.com/google/uuid"
)

type EnrollmentNotFoundError struct {
	EnrollmentID uuid.UUID
}

func (e *EnrollmentNotFoundError) Error() string {
	return fmt.Sprintf("enrollment not found: %s", e.EnrollmentID)
}

type EnrollmentAlreadyExistsError struct {
	StudentID uuid.UUID
	CourseID  uuid.UUID
}

func (e *EnrollmentAlreadyExistsError) Error() string {
	return fmt.Sprintf("student %s already enrolled in course %s", e.StudentID, e.CourseID)
}

// EnrollmentNotActiveError: the enrollment exists but is no longer active
// (e.g. it was already cancelled), so the requested change does not apply.
type EnrollmentNotActiveError struct {
	EnrollmentID uuid.UUID
	Status       string
}

func (e *EnrollmentNotActiveError) Error() string {
	return fmt.Sprintf("enrollment %s is %s, not active", e.EnrollmentID, e.Status)
}
