package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	LessonStatusNotStarted = "not_started"
	LessonStatusInProgress = "in_progress"
	LessonStatusCompleted  = "completed"
)

// LessonProgress is one row per (enrollment, lesson). LessonID refers to a
// lesson in course-service, so it is a plain ID with no foreign key.
type LessonProgress struct {
	ID           uuid.UUID  `json:"id"`
	EnrollmentID uuid.UUID  `json:"enrollment_id"`
	LessonID     uuid.UUID  `json:"lesson_id"`
	Status       string     `json:"status"`
	CompletedAt  *time.Time `json:"completed_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (p *LessonProgress) IsCompleted() bool {
	return p.Status == LessonStatusCompleted
}
