package mappers

import (
	"smartcourse/services/user-service/domain"
	"smartcourse/services/user-service/persistence"
)

func UserDBToDomain(udb *persistence.UserDB) *domain.User {
	return &domain.User{
		ID:           udb.ID,
		Name:         udb.Name,
		Email:        udb.Email,
		Role:         udb.Role,
		CreatedAt:    udb.CreatedAt,
		UpdatedAt:    udb.UpdatedAt,
		PasswordHash: udb.PasswordHash,
	}
}

func UserToPersistence(u *domain.User) *persistence.UserDB {
	return &persistence.UserDB{
		ID:           u.ID,
		Name:         u.Name,
		Email:        u.Email,
		Role:         u.Role,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
		PasswordHash: u.PasswordHash,
	}
}
