package persistence

import (
	"time"

	"github.com/google/uuid"
)

type CourseDB struct {
	ID           uuid.UUID `gorm:"primaryKey"`
	Name         string    `gorm:"type:varchar(255);not null"`
	InstructorID uuid.UUID `gorm:"not null"`
	Status       string    `gorm:"type:varchar(50);not null;default:'draft'"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (CourseDB) TableName() string {
	return "course_service.courses"
}
