package persistence

import (
	"time"

	"github.com/google/uuid"
)

type RefreshTokenDB struct {
	ID        uuid.UUID `gorm:"primaryKey"`
	UserID    uuid.UUID `gorm:"not null"`
	TokenHash string    `gorm:"type:varchar(128);uniqueIndex;not null"`
	FamilyID  uuid.UUID `gorm:"not null"`
	RevokedAt *time.Time
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time
}

func (RefreshTokenDB) TableName() string {
	return "user_service.refresh_tokens"
}
