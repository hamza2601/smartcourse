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
