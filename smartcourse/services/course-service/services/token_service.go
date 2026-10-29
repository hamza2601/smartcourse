package services

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"smartcourse/libs/common/auth"
	apperrors "smartcourse/libs/common/errors"
	"smartcourse/services/course-service/domain"
)

type TokenService struct {
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewTokenService(accessExpiry, refreshExpiry time.Duration) *TokenService {
	return &TokenService{accessExpiry: accessExpiry, refreshExpiry: refreshExpiry}
}

func (ts *TokenService) GenerateAccessToken(user *domain.User, secretKey string) (string, error) {
	now := time.Now()
	claims := domain.AccessTokenClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			Audience:  jwt.ClaimStrings{domain.TokenAudienceAccess},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ts.accessExpiry)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secretKey))
}

// GenerateRefreshToken starts a new token family.
func (ts *TokenService) GenerateRefreshToken(userID uuid.UUID, secretKey string) (string, uuid.UUID, error) {
	familyID := uuid.New()
	token, err := ts.GenerateRefreshTokenForFamily(userID, familyID, secretKey)
	if err != nil {
		return "", uuid.Nil, err
	}
	return token, familyID, nil
}

// GenerateRefreshTokenForFamily is used on rotation to keep the family_id.
// The unique jti guarantees two tokens issued in the same second still differ,
// which the unique token_hash column relies on.
func (ts *TokenService) GenerateRefreshTokenForFamily(userID, familyID uuid.UUID, secretKey string) (string, error) {
	now := time.Now()
	claims := domain.RefreshTokenClaims{
		UserID:   userID,
		FamilyID: familyID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Subject:   userID.String(),
			Audience:  jwt.ClaimStrings{domain.TokenAudienceRefresh},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ts.refreshExpiry)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secretKey))
}

// AccessTokenVerifier adapts VerifyAccessToken for the shared auth middleware.
func (ts *TokenService) AccessTokenVerifier(secretKey string) auth.VerifyFunc {
	return func(token string) (*auth.Identity, error) {
		claims, err := ts.VerifyAccessToken(token, secretKey)
		if err != nil {
			return nil, err
		}
		return &auth.Identity{UserID: claims.UserID, Email: claims.Email, Role: claims.Role}, nil
	}
}

func (ts *TokenService) VerifyAccessToken(tokenString, secretKey string) (*domain.AccessTokenClaims, error) {
	claims := &domain.AccessTokenClaims{}
	if err := parseToken(tokenString, secretKey, domain.TokenAudienceAccess, claims); err != nil {
		return nil, err
	}
	return claims, nil
}

func (ts *TokenService) VerifyRefreshToken(tokenString, secretKey string) (*domain.RefreshTokenClaims, error) {
	claims := &domain.RefreshTokenClaims{}
	if err := parseToken(tokenString, secretKey, domain.TokenAudienceRefresh, claims); err != nil {
		return nil, err
	}
	return claims, nil
}

// Pinning HS256 blocks alg-confusion/"none" tokens; the audience check stops
// a refresh token from being used as an access token and vice versa.
func parseToken(tokenString, secretKey, audience string, claims jwt.Claims) error {
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(*jwt.Token) (any, error) { return []byte(secretKey), nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithAudience(audience),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return &apperrors.InvalidTokenError{Reason: err.Error()}
	}
	if !token.Valid {
		return &apperrors.InvalidTokenError{Reason: "token is not valid"}
	}
	return nil
}
