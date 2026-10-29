package domain

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Audience values keep a refresh token from being accepted as an access token
// (both are signed with the same secret).
const (
	TokenAudienceAccess  = "access"
	TokenAudienceRefresh = "refresh"
)

type AccessTokenClaims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Role   string    `json:"role"`
	jwt.RegisteredClaims
}

type RefreshTokenClaims struct {
	UserID   uuid.UUID `json:"user_id"`
	FamilyID uuid.UUID `json:"family_id"`
	jwt.RegisteredClaims
}
