package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	apperrors "smartcourse/internal/errors"
	"smartcourse/internal/models"
	"smartcourse/internal/repositories"
)

type UserService struct {
	Repo *repositories.UserRepository
}

func NewUserService(repo *repositories.UserRepository) *UserService {
	return &UserService{Repo: repo}
}

func (us *UserService) RegisterUser(ctx context.Context, name, email, role string) (*models.User, error) {
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

	user := &models.User{
		Name:  name,
		Email: email,
		Role:  role,
	}
	if err := us.Repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (us *UserService) GetUser(ctx context.Context, id uuid.UUID) (*models.User, error) {
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
