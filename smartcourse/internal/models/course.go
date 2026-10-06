package models

import (
	"time"

	"github.com/google/uuid"
)

type Course struct {
	ID           uuid.UUID `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"type:varchar(255);not null" json:"name"`
	InstructorID uuid.UUID `gorm:"not null" json:"instructor_id"`
	Status       string    `gorm:"type:varchar(50);not null;default:'draft'" json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (Course) TableName() string {
	return "course_service.courses"
}
