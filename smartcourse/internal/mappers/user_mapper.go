package mappers

import (
	"smartcourse/internal/domain"
	"smartcourse/internal/persistence"
)

func UserDBToDomain(udb *persistence.UserDB) *domain.User {
	return &domain.User{
		ID:        udb.ID,
		Name:      udb.Name,
		Email:     udb.Email,
		Role:      udb.Role,
		CreatedAt: udb.CreatedAt,
		UpdatedAt: udb.UpdatedAt,
	}
}

func UserToPersistence(u *domain.User) *persistence.UserDB {
	return &persistence.UserDB{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
