package models

import (
	"time"

	"github.com/google/uuid"
)

type OutboxEvent struct {
	ID          uuid.UUID  `gorm:"primaryKey" json:"id"`
	EventType   string     `gorm:"type:varchar(255);not null" json:"event_type"`
	Payload     string     `gorm:"type:jsonb;not null" json:"payload"`
	CreatedAt   time.Time  `json:"created_at"`
	ProcessedAt *time.Time `json:"processed_at"`
}
