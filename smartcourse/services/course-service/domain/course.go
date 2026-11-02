package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	CourseStatusDraft     = "draft"
	CourseStatusPublished = "published"
	CourseStatusArchived  = "archived"
)

type Course struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	InstructorID uuid.UUID `json:"instructor_id"`
	Status       string    `json:"status"`
	// nil means unlimited.
	EnrollmentLimit *int      `json:"enrollment_limit"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (c *Course) IsPublished() bool {
	return c.Status == CourseStatusPublished
}

func (c *Course) IsDraft() bool {
	return c.Status == CourseStatusDraft
}

func (c *Course) IsOwnedBy(userID uuid.UUID) bool {
	return c.InstructorID == userID
}

func (c *Course) CanBeDeletedBy(userID uuid.UUID) bool {
	return c.IsOwnedBy(userID)
}
