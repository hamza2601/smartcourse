package persistence

import (
	"time"

	"github.com/google/uuid"
)

type ModuleDB struct {
	ID        uuid.UUID `gorm:"primaryKey"`
	CourseID  uuid.UUID `gorm:"not null"`
	Title     string    `gorm:"type:varchar(255);not null"`
	Position  int       `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ModuleDB) TableName() string {
	return "course_service.modules"
}

type LessonDB struct {
	ID        uuid.UUID `gorm:"primaryKey"`
	ModuleID  uuid.UUID `gorm:"not null"`
	Title     string    `gorm:"type:varchar(255);not null"`
	Content   string    `gorm:"type:text;not null;default:''"`
	Position  int       `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (LessonDB) TableName() string {
	return "course_service.lessons"
}
