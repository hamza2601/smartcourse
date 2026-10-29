package errors

import (
	"fmt"

	"github.com/google/uuid"
)

type UserNotFoundError struct {
	UserID uuid.UUID
	Email  string
}

func (e *UserNotFoundError) Error() string {
	if e.Email != "" {
		return fmt.Sprintf("user not found: email %s", e.Email)
	}
	return fmt.Sprintf("user not found: %s", e.UserID)
}

type UserAlreadyExistsError struct {
	Email string
}

func (e *UserAlreadyExistsError) Error() string {
	return fmt.Sprintf("user already exists with email: %s", e.Email)
}
