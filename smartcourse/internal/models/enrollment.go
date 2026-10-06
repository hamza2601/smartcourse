package models

import (
	"time"

	"github.com/google/uuid"
)

type Enrollment struct {
	ID        uuid.UUID `gorm:"primaryKey" json:"id"`
	StudentID uuid.UUID `gorm:"not null;uniqueIndex:idx_student_course" json:"student_id"`
	CourseID  uuid.UUID `gorm:"not null;uniqueIndex:idx_student_course" json:"course_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (Enrollment) TableName() string {
	return "enrollment_service.enrollments"
}
