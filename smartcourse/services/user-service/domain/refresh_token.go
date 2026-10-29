package domain

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	FamilyID  uuid.UUID
	RevokedAt *time.Time
	ExpiresAt time.Time
	CreatedAt time.Time
}

func (t *RefreshToken) IsRevoked() bool {
	return t.RevokedAt != nil
}

func (t *RefreshToken) IsExpired() bool {
	return !time.Now().Before(t.ExpiresAt)
}

func (t *RefreshToken) IsValid() bool {
	return !t.IsRevoked() && !t.IsExpired()
}
