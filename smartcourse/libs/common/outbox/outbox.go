// Package outbox implements the transactional outbox: events are written to
// the service's outbox_events table in the same transaction as the business
// change, and a Relay later publishes them. Delivery is at-least-once.
package outbox

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Event struct {
	ID            uuid.UUID
	AggregateType string
	AggregateID   uuid.UUID
	EventType     string
	Payload       json.RawMessage
	CreatedAt     time.Time
	ProcessedAt   *time.Time
}

// NewEvent builds an event, marshalling payload to JSON.
func NewEvent(aggregateType string, aggregateID uuid.UUID, eventType string, payload any) (Event, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return Event{}, fmt.Errorf("marshal %s payload: %w", eventType, err)
	}
	return Event{
		ID:            uuid.New(),
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		EventType:     eventType,
		Payload:       data,
	}, nil
}

// record is a row of an outbox_events table (same shape in every schema).
type record struct {
	ID            uuid.UUID `gorm:"primaryKey"`
	AggregateType string
	AggregateID   uuid.UUID
	EventType     string
	Payload       string `gorm:"type:jsonb"`
	CreatedAt     time.Time
	ProcessedAt   *time.Time
}

func (r record) toEvent() Event {
	return Event{
		ID:            r.ID,
		AggregateType: r.AggregateType,
		AggregateID:   r.AggregateID,
		EventType:     r.EventType,
		Payload:       json.RawMessage(r.Payload),
		CreatedAt:     r.CreatedAt,
		ProcessedAt:   r.ProcessedAt,
	}
}

// Insert writes ev to table (e.g. "user_service.outbox_events") using tx, so
// the event commits or rolls back together with the caller's business write.
// tx must be the caller's transaction handle, not a fresh connection.
func Insert(tx *gorm.DB, table string, ev Event) error {
	if ev.ID == uuid.Nil {
		ev.ID = uuid.New()
	}
	createdAt := ev.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	rec := record{
		ID:            ev.ID,
		AggregateType: ev.AggregateType,
		AggregateID:   ev.AggregateID,
		EventType:     ev.EventType,
		Payload:       string(ev.Payload),
		CreatedAt:     createdAt,
	}
	if err := tx.Table(table).Create(&rec).Error; err != nil {
		return fmt.Errorf("insert outbox event %s: %w", ev.EventType, err)
	}
	return nil
}
