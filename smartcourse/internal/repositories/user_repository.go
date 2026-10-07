package repositories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	apperrors "smartcourse/internal/errors"
	"smartcourse/internal/models"
)

type UserRepository struct {
	DB *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{DB: db}
}

func (ur *UserRepository) CreateUser(ctx context.Context, user *models.User) error {
	user.ID = uuid.New()
	if err := ur.DB.WithContext(ctx).Create(user).Error; err != nil {
		if isDuplicateKey(err) {
			return &apperrors.UserAlreadyExistsError{Email: user.Email}
		}
		return &apperrors.DatabaseError{Operation: "create user", Err: err}
	}
	return nil
}

func (ur *UserRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	if err := ur.DB.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &apperrors.UserNotFoundError{UserID: id}
		}
		return nil, &apperrors.DatabaseError{Operation: "get user by id", Err: err}
	}
	return &user, nil
}

func (ur *UserRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := ur.DB.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &apperrors.UserNotFoundError{Email: email}
		}
		return nil, &apperrors.DatabaseError{Operation: "get user by email", Err: err}
	}
	return &user, nil
}

func (ur *UserRepository) GetAllUsers(ctx context.Context) ([]models.User, error) {
	var users []models.User
	if err := ur.DB.WithContext(ctx).Find(&users).Error; err != nil {
		return nil, &apperrors.DatabaseError{Operation: "list users", Err: err}
	}
	return users, nil
}
