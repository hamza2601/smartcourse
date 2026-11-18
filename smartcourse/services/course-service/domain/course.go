package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	CourseStatusDraft      = "draft"
	CourseStatusPublishing = "publishing"
	CourseStatusReady      = "ready"
	CourseStatusArchived   = "archived"
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

func (c *Course) IsReady() bool {
	return c.Status == CourseStatusReady
}

func (c *Course) IsPublishing() bool {
	return c.Status == CourseStatusPublishing
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
