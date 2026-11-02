package persistence

import "github.com/google/uuid"

type CourseSeatDB struct {
	CourseID uuid.UUID `gorm:"primaryKey"`
	Capacity *int
	Taken    int `gorm:"not null;default:0"`
}

func (CourseSeatDB) TableName() string {
	return "enrollment_service.course_seats"
}
