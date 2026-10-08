package persistence

import (
	"time"

	"github.com/google/uuid"
)

type ProgressDB struct {
	StudentID        uuid.UUID `gorm:"primaryKey"`
	CourseID         uuid.UUID `gorm:"primaryKey"`
	CompletedLessons int       `gorm:"default:0"`
	LastAccessed     time.Time
}

func (ProgressDB) TableName() string {
	return "course_service.progress"
}
