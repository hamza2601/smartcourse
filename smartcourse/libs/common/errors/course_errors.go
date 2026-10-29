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
