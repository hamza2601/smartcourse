package domain

import "github.com/google/uuid"

const (
	AggregateTypeCourse  = "course"
	EventCourseCreated   = "CourseCreated"
	EventCoursePublished = "CoursePublished"
)

type CourseCreatedPayload struct {
	CourseID        uuid.UUID `json:"course_id"`
	InstructorID    uuid.UUID `json:"instructor_id"`
	Name            string    `json:"name"`
	Status          string    `json:"status"`
	EnrollmentLimit *int      `json:"enrollment_limit"`
}

type CoursePublishedPayload struct {
	CourseID        uuid.UUID `json:"course_id"`
	InstructorID    uuid.UUID `json:"instructor_id"`
	Name            string    `json:"name"`
	EnrollmentLimit *int      `json:"enrollment_limit"`
}
