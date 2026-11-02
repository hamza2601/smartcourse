package repositories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	apperrors "smartcourse/libs/common/errors"
	"smartcourse/libs/common/outbox"
	"smartcourse/services/user-service/domain"
	"smartcourse/services/user-service/mappers"
	"smartcourse/services/user-service/persistence"
)

type UserRepository struct {
	DB *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{DB: db}
}

// CreateUser inserts the user and its UserRegistered outbox event in one
// transaction: either both are stored or neither is.
func (ur *UserRepository) CreateUser(ctx context.Context, user *domain.User, event outbox.Event) error {
	userDB := mappers.UserToPersistence(user)
	if userDB.ID == uuid.Nil {
		userDB.ID = uuid.New()
	}

	err := ur.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(userDB).Error; err != nil {
			if isDuplicateKey(err) {
				return &apperrors.UserAlreadyExistsError{Email: user.Email}
			}
			return &apperrors.DatabaseError{Operation: "create user", Err: err}
		}
		if err := outbox.Insert(tx, persistence.OutboxTable, event); err != nil {
			return &apperrors.DatabaseError{Operation: "write user outbox event", Err: err}
		}
		return nil
	})
	if err != nil {
		var exists *apperrors.UserAlreadyExistsError
		var dbErr *apperrors.DatabaseError
		if errors.As(err, &exists) || errors.As(err, &dbErr) {
			return err
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
