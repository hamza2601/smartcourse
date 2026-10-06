package models

import (
	"time"

	"github.com/google/uuid"
)

type Progress struct {
	StudentID        uuid.UUID `gorm:"primaryKey;uniqueIndex:idx_student_course" json:"student_id"`
	CourseID         uuid.UUID `gorm:"primaryKey;uniqueIndex:idx_student_course" json:"course_id"`
	CompletedLessons int       `gorm:"default:0" json:"completed_lessons"`
	LastAccessed     time.Time `json:"last_accessed"`
}

func (Progress) TableName() string {
	return "course_service.progress"
}
