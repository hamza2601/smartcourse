package models

import (
	"time"

	"github.com/google/uuid"
)

type Enrollment struct {
	ID        uuid.UUID `gorm:"primaryKey" json:"id"`
	StudentID uuid.UUID `gorm:"not null" json:"student_id"`
	CourseID  uuid.UUID `gorm:"not null" json:"course_id"`
	Status    string    `gorm:"type:varchar(50);not null;default:'active'" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Enrollment) TableName() string {
	return "enrollment_service.enrollments"
}
