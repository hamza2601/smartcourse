package persistence

import (
	"time"

	"github.com/google/uuid"
)

type LessonProgressDB struct {
	ID           uuid.UUID `gorm:"primaryKey"`
	EnrollmentID uuid.UUID `gorm:"not null"`
	LessonID     uuid.UUID `gorm:"not null"`
	Status       string    `gorm:"type:varchar(20);not null;default:'not_started'"`
	CompletedAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (LessonProgressDB) TableName() string {
	return "enrollment_service.lesson_progress"
}
