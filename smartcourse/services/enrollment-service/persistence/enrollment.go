package persistence

import (
	"time"

	"github.com/google/uuid"
)

type EnrollmentDB struct {
	ID        uuid.UUID `gorm:"primaryKey"`
	StudentID uuid.UUID `gorm:"not null"`
	CourseID  uuid.UUID `gorm:"not null"`
	Status    string    `gorm:"type:varchar(50);not null;default:'active'"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (EnrollmentDB) TableName() string {
	return "enrollment_service.enrollments"
}
