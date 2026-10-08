package domain

import (
	"time"

	"github.com/google/uuid"
)

type Progress struct {
	StudentID        uuid.UUID `json:"student_id"`
	CourseID         uuid.UUID `json:"course_id"`
	CompletedLessons int       `json:"completed_lessons"`
	LastAccessed     time.Time `json:"last_accessed"`
}
