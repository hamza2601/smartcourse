package models

type UserServiceOutboxEvent struct {
	OutboxEvent
}

func (UserServiceOutboxEvent) TableName() string {
	return "user_service.outbox_events"
}
