package errors

import "fmt"

type InvalidCredentialsError struct{}

func (e *InvalidCredentialsError) Error() string {
	return "invalid email or password"
}

type InvalidTokenError struct {
	Reason string
}

func (e *InvalidTokenError) Error() string {
	return fmt.Sprintf("invalid token: %s", e.Reason)
}

type RefreshTokenNotFoundError struct{}

func (e *RefreshTokenNotFoundError) Error() string {
	return "refresh token not found"
}
