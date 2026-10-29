package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	apperrors "smartcourse/libs/common/errors"
	"smartcourse/services/user-service/config"
	"smartcourse/services/user-service/domain"
	"smartcourse/services/user-service/repositories"
)

type AuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type AuthService struct {
	UserRepo  *repositories.UserRepository
	TokenRepo *repositories.RefreshTokenRepository
	TokenSvc  *TokenService
	JWT       config.JWTConfig
}

func NewAuthService(
	userRepo *repositories.UserRepository,
	tokenRepo *repositories.RefreshTokenRepository,
	tokenSvc *TokenService,
	jwtCfg config.JWTConfig,
) *AuthService {
	return &AuthService{UserRepo: userRepo, TokenRepo: tokenRepo, TokenSvc: tokenSvc, JWT: jwtCfg}
}

// Compared against when the user is unknown, so a missing account and a wrong
// password take about the same time.
var dummyPasswordHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password"), bcrypt.DefaultCost)

func (as *AuthService) Login(ctx context.Context, email, password string) (*AuthTokens, error) {
	user, err := as.UserRepo.GetUserByEmail(ctx, email)
	if err != nil {
		var notFound *apperrors.UserNotFoundError
		if errors.As(err, &notFound) {
			bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(password))
			return nil, &apperrors.InvalidCredentialsError{}
		}
		return nil, err
	}

	hash := []byte(user.PasswordHash)
	hasPassword := len(hash) > 0
	if !hasPassword {
		hash = dummyPasswordHash
	}
	if err := bcrypt.CompareHashAndPassword(hash, []byte(password)); err != nil || !hasPassword {
		return nil, &apperrors.InvalidCredentialsError{}
	}

	return as.issueTokens(ctx, user, uuid.New())
}

func (as *AuthService) RefreshAccessToken(ctx context.Context, refreshToken string) (*AuthTokens, error) {
	claims, err := as.TokenSvc.VerifyRefreshToken(refreshToken, as.JWT.SecretKey)
	if err != nil {
		return nil, err
	}

	tokenHash := hashToken(refreshToken)
	stored, err := as.TokenRepo.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		var notFound *apperrors.RefreshTokenNotFoundError
		if errors.As(err, &notFound) {
			return nil, &apperrors.InvalidTokenError{Reason: "refresh token not recognized"}
		}
		return nil, err
	}
	if stored.UserID != claims.UserID || stored.FamilyID != claims.FamilyID {
		return nil, &apperrors.InvalidTokenError{Reason: "refresh token does not match stored record"}
	}

	// A revoked token showing up again means it was stolen or replayed:
	// kill the whole family so the legitimate holder has to log in again.
	if stored.IsRevoked() {
		if err := as.TokenRepo.RevokeFamily(ctx, stored.FamilyID); err != nil {
			return nil, err
		}
		return nil, &apperrors.InvalidTokenError{Reason: "refresh token reuse detected"}
	}
	if stored.IsExpired() {
		return nil, &apperrors.InvalidTokenError{Reason: "refresh token expired"}
	}

	user, err := as.UserRepo.GetUserByID(ctx, stored.UserID)
	if err != nil {
		var notFound *apperrors.UserNotFoundError
		if errors.As(err, &notFound) {
			return nil, &apperrors.InvalidTokenError{Reason: "user no longer exists"}
		}
		return nil, err
	}

	// Atomic claim of the old token: if another request already rotated it,
	// treat this as reuse.
	if err := as.TokenRepo.RevokeRefreshToken(ctx, tokenHash); err != nil {
		var notFound *apperrors.RefreshTokenNotFoundError
		if errors.As(err, &notFound) {
			if err := as.TokenRepo.RevokeFamily(ctx, stored.FamilyID); err != nil {
				return nil, err
			}
			return nil, &apperrors.InvalidTokenError{Reason: "refresh token reuse detected"}
		}
		return nil, err
	}

	return as.issueTokens(ctx, user, stored.FamilyID)
}

func (as *AuthService) issueTokens(ctx context.Context, user *domain.User, familyID uuid.UUID) (*AuthTokens, error) {
	accessToken, err := as.TokenSvc.GenerateAccessToken(user, as.JWT.SecretKey)
	if err != nil {
		return nil, err
	}
	refreshToken, err := as.TokenSvc.GenerateRefreshTokenForFamily(user.ID, familyID, as.JWT.SecretKey)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(as.JWT.RefreshTokenExpiry)
	if err := as.TokenRepo.CreateRefreshToken(ctx, user.ID, hashToken(refreshToken), familyID, expiresAt); err != nil {
		return nil, err
	}
	return &AuthTokens{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
