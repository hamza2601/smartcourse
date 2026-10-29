package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	CourseStatusDraft     = "draft"
	CourseStatusPublished = "published"
)

type Course struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	InstructorID uuid.UUID `json:"instructor_id"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (c *Course) IsPublished() bool {
	return c.Status == CourseStatusPublished
}

func (c *Course) CanBeDeletedBy(userID uuid.UUID) bool {
	return c.InstructorID == userID
}
