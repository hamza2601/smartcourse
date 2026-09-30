package models

import "time"

type Enrollment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	StudentID uint      `gorm:"not null;uniqueIndex:idx_student_course" json:"student_id"`
	CourseID  uint      `gorm:"not null;uniqueIndex:idx_student_course" json:"course_id"`
	Student   User      `gorm:"foreignKey:StudentID" json:"-"`
	Course    Course    `gorm:"foreignKey:CourseID" json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

func (Enrollment) TableName() string {
	return "enrollments"
}
