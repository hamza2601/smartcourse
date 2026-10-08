package repositories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"smartcourse/internal/domain"
	apperrors "smartcourse/internal/errors"
	"smartcourse/internal/mappers"
	"smartcourse/internal/persistence"
)

type UserRepository struct {
	DB *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{DB: db}
}

func (ur *UserRepository) CreateUser(ctx context.Context, user *domain.User) error {
	userDB := mappers.UserToPersistence(user)
	userDB.ID = uuid.New()
	if err := ur.DB.WithContext(ctx).Create(userDB).Error; err != nil {
		if isDuplicateKey(err) {
			return &apperrors.UserAlreadyExistsError{Email: user.Email}
		}
		return &apperrors.DatabaseError{Operation: "create user", Err: err}
	}
	*user = *mappers.UserDBToDomain(userDB)
	return nil
}

func (ur *UserRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var userDB persistence.UserDB
	if err := ur.DB.WithContext(ctx).Where("id = ?", id).First(&userDB).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &apperrors.UserNotFoundError{UserID: id}
		}
		return nil, &apperrors.DatabaseError{Operation: "get user by id", Err: err}
	}
	return mappers.UserDBToDomain(&userDB), nil
}

func (ur *UserRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	var userDB persistence.UserDB
	if err := ur.DB.WithContext(ctx).Where("email = ?", email).First(&userDB).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &apperrors.UserNotFoundError{Email: email}
		}
		return nil, &apperrors.DatabaseError{Operation: "get user by email", Err: err}
	}
	return mappers.UserDBToDomain(&userDB), nil
}

func (ur *UserRepository) GetAllUsers(ctx context.Context) ([]domain.User, error) {
	var usersDB []persistence.UserDB
	if err := ur.DB.WithContext(ctx).Find(&usersDB).Error; err != nil {
		return nil, &apperrors.DatabaseError{Operation: "list users", Err: err}
	}
	users := make([]domain.User, len(usersDB))
	for i := range usersDB {
		users[i] = *mappers.UserDBToDomain(&usersDB[i])
	}
	return users, nil
}
