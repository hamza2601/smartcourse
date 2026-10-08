package domain

import (
	"time"

	"github.com/google/uuid"
)

const EnrollmentStatusActive = "active"

type Enrollment struct {
	ID        uuid.UUID `json:"id"`
	StudentID uuid.UUID `json:"student_id"`
	CourseID  uuid.UUID `json:"course_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (e *Enrollment) IsActive() bool {
	return e.Status == EnrollmentStatusActive
}
