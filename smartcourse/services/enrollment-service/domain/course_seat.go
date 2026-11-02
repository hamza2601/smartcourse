package domain

import "github.com/google/uuid"

// CourseSeat tracks how many seats of a course are taken. Capacity is copied
// from the course's enrollment_limit; nil means unlimited.
type CourseSeat struct {
	CourseID uuid.UUID `json:"course_id"`
	Capacity *int      `json:"capacity"`
	Taken    int       `json:"taken"`
}

func (s *CourseSeat) IsFull() bool {
	return s.Capacity != nil && s.Taken >= *s.Capacity
}
