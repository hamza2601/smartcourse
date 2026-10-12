package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"smartcourse/internal/domain"
	apperrors "smartcourse/internal/errors"
	"smartcourse/internal/mappers"
	"smartcourse/internal/persistence"
)

type RefreshTokenRepository struct {
	DB *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{DB: db}
}

func (rr *RefreshTokenRepository) CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, familyID uuid.UUID, expiresAt time.Time) error {
	tokenDB := &persistence.RefreshTokenDB{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: tokenHash,
		FamilyID:  familyID,
		ExpiresAt: expiresAt.UTC(),
		CreatedAt: time.Now().UTC(),
	}
	if err := rr.DB.WithContext(ctx).Create(tokenDB).Error; err != nil {
		return &apperrors.DatabaseError{Operation: "create refresh token", Err: err}
	}
	return nil
}

func (rr *RefreshTokenRepository) GetRefreshToken(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	var tokenDB persistence.RefreshTokenDB
	if err := rr.DB.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&tokenDB).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &apperrors.RefreshTokenNotFoundError{}
		}
		return nil, &apperrors.DatabaseError{Operation: "get refresh token", Err: err}
	}
	return mappers.RefreshTokenDBToDomain(&tokenDB), nil
}

// RevokeRefreshToken only touches tokens that are not yet revoked and returns
// RefreshTokenNotFoundError when none were, so a caller that loses a race to
// rotate the same token can tell.
func (rr *RefreshTokenRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	result := rr.DB.WithContext(ctx).
		Model(&persistence.RefreshTokenDB{}).
		Where("token_hash = ? AND revoked_at IS NULL", tokenHash).
		Update("revoked_at", time.Now().UTC())
	if result.Error != nil {
		return &apperrors.DatabaseError{Operation: "revoke refresh token", Err: result.Error}
	}
	if result.RowsAffected == 0 {
		return &apperrors.RefreshTokenNotFoundError{}
	}
	return nil
}

func (rr *RefreshTokenRepository) RevokeFamily(ctx context.Context, familyID uuid.UUID) error {
	if err := rr.DB.WithContext(ctx).
		Model(&persistence.RefreshTokenDB{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Update("revoked_at", time.Now().UTC()).Error; err != nil {
		return &apperrors.DatabaseError{Operation: "revoke token family", Err: err}
	}
	return nil
}

func (rr *RefreshTokenRepository) DeleteExpiredTokens(ctx context.Context) error {
	if err := rr.DB.WithContext(ctx).
		Where("expires_at < ?", time.Now().UTC()).
		Delete(&persistence.RefreshTokenDB{}).Error; err != nil {
		return &apperrors.DatabaseError{Operation: "delete expired refresh tokens", Err: err}
	}
	return nil
}
