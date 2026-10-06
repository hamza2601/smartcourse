package models

type EnrollmentServiceOutboxEvent struct {
	OutboxEvent
}

func (EnrollmentServiceOutboxEvent) TableName() string {
	return "enrollment_service.outbox_events"
}
