package models

import "time"

type Course struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"type:varchar(255);not null" json:"name"`
	InstructorID uint      `gorm:"not null;foreignKey:ID;references:ID" json:"instructor_id"`
	Instructor   User      `gorm:"foreignKey:InstructorID" json:"-"`
	Status       string    `gorm:"type:varchar(50);not null;default:'draft'" json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
