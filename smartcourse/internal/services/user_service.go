package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"smartcourse/internal/domain"
	apperrors "smartcourse/internal/errors"
	"smartcourse/internal/repositories"
)

const minPasswordLength = 8

type UserService struct {
	Repo *repositories.UserRepository
}

func NewUserService(repo *repositories.UserRepository) *UserService {
	return &UserService{Repo: repo}
}

func (us *UserService) RegisterUser(ctx context.Context, name, email, role, password string) (*domain.User, error) {
	if name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}
	if email == "" {
		return nil, fmt.Errorf("email cannot be empty")
	}
	if !us.ValidateRole(role) {
		return nil, fmt.Errorf("invalid role: %s", role)
	}

	existing, err := us.Repo.GetUserByEmail(ctx, email)
	if err == nil && existing != nil {
		return nil, &apperrors.UserAlreadyExistsError{Email: email}
	}

	user := &domain.User{
		Name:  name,
		Email: email,
		Role:  role,
	}
	if password != "" {
		if len(password) < minPasswordLength {
			return nil, fmt.Errorf("password must be at least %d characters", minPasswordLength)
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("invalid password: %w", err)
		}
		user.PasswordHash = string(hash)
	}
	if err := us.Repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (us *UserService) GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return us.Repo.GetUserByID(ctx, id)
}

func (us *UserService) ValidateRole(role string) bool {
	validRoles := map[string]bool{
		"student":    true,
		"instructor": true,
		"admin":      true,
	}
	return validRoles[role]
}
