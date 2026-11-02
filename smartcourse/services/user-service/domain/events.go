package domain

import "github.com/google/uuid"

const (
	AggregateTypeUser   = "user"
	EventUserRegistered = "UserRegistered"
)

// UserRegisteredPayload never includes credentials.
type UserRegisteredPayload struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Name   string    `json:"name"`
	Role   string    `json:"role"`
}
