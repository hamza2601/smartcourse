package persistence

import (
	"time"

	"github.com/google/uuid"
)

type UserDB struct {
	ID        uuid.UUID `gorm:"primaryKey"`
	Name      string    `gorm:"type:varchar(255);not null"`
	Email     string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	Role      string    `gorm:"type:varchar(50);not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (UserDB) TableName() string {
	return "user_service.users"
}
