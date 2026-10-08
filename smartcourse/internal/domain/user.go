package domain

import (
	"fmt"
	"net/mail"
	"time"

	"github.com/google/uuid"
)

const (
	RoleStudent    = "student"
	RoleInstructor = "instructor"
	RoleAdmin      = "admin"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

func (u *User) CanEditCourse(ownerID uuid.UUID) bool {
	return u.IsAdmin() || u.ID == ownerID
}

func (u *User) ValidateEmail() error {
	addr, err := mail.ParseAddress(u.Email)
	if err != nil || addr.Address != u.Email {
		return fmt.Errorf("invalid email address: %s", u.Email)
	}
	return nil
}
