package domain

import "github.com/google/uuid"

const (
	AggregateTypeEnrollment = "enrollment"
	EventEnrollmentCreated  = "EnrollmentCreated"
)

type EnrollmentCreatedPayload struct {
	EnrollmentID uuid.UUID `json:"enrollment_id"`
	StudentID    uuid.UUID `json:"student_id"`
	CourseID     uuid.UUID `json:"course_id"`
	Status       string    `json:"status"`
}
