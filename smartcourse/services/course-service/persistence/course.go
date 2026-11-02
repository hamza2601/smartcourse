package persistence

import (
	"time"

	"github.com/google/uuid"
)

type CourseDB struct {
	ID              uuid.UUID `gorm:"primaryKey"`
	Name            string    `gorm:"type:varchar(255);not null"`
	Description     string    `gorm:"type:text;not null;default:''"`
	InstructorID    uuid.UUID `gorm:"not null"`
	Status          string    `gorm:"type:varchar(20);not null;default:'draft'"`
	EnrollmentLimit *int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (CourseDB) TableName() string {
	return "course_service.courses"
}
