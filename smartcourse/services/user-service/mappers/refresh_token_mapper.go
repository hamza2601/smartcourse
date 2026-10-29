package mappers

import (
	"smartcourse/services/user-service/domain"
	"smartcourse/services/user-service/persistence"
)

func RefreshTokenDBToDomain(rdb *persistence.RefreshTokenDB) *domain.RefreshToken {
	return &domain.RefreshToken{
		ID:        rdb.ID,
		UserID:    rdb.UserID,
		TokenHash: rdb.TokenHash,
		FamilyID:  rdb.FamilyID,
		RevokedAt: rdb.RevokedAt,
		ExpiresAt: rdb.ExpiresAt,
		CreatedAt: rdb.CreatedAt,
	}
}

func RefreshTokenToPersistence(t *domain.RefreshToken) *persistence.RefreshTokenDB {
	return &persistence.RefreshTokenDB{
		ID:        t.ID,
		UserID:    t.UserID,
		TokenHash: t.TokenHash,
		FamilyID:  t.FamilyID,
		RevokedAt: t.RevokedAt,
		ExpiresAt: t.ExpiresAt,
		CreatedAt: t.CreatedAt,
	}
}
