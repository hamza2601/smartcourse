package models

import "time"

type Progress struct {
	StudentID        uint      `gorm:"primaryKey;uniqueIndex:idx_student_course" json:"student_id"`
	CourseID         uint      `gorm:"primaryKey;uniqueIndex:idx_student_course" json:"course_id"`
	Student          User      `gorm:"foreignKey:StudentID" json:"-"`
	Course           Course    `gorm:"foreignKey:CourseID" json:"-"`
	CompletedLessons int       `gorm:"default:0" json:"completed_lessons"`
	LastAccessed     time.Time `json:"last_accessed"`
}

func (Progress) TableName() string {
	return "progress"
}
