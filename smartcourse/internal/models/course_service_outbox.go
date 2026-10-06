package models

type CourseServiceOutboxEvent struct {
	OutboxEvent
}

func (CourseServiceOutboxEvent) TableName() string {
	return "course_service.outbox_events"
}
