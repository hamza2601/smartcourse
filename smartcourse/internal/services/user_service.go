package services

import (
	"fmt"

	"smartcourse/internal/models"
	"smartcourse/internal/repositories"
)

type UserService struct {
	Repo *repositories.UserRepository
}

func NewUserService(repo *repositories.UserRepository) *UserService {
	return &UserService{Repo: repo}
}

func (us *UserService) RegisterUser(name, email, role string) (*models.User, error) {
	if name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}
	if email == "" {
		return nil, fmt.Errorf("email cannot be empty")
	}
	if !us.ValidateRole(role) {
		return nil, fmt.Errorf("invalid role: %s", role)
	}

	existing, err := us.Repo.GetUserByEmail(email)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("email already exists")
	}

	user := &models.User{
		Name:  name,
		Email: email,
		Role:  role,
	}
	if err := us.Repo.CreateUser(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (us *UserService) GetUser(id uint) (*models.User, error) {
	return us.Repo.GetUserByID(id)
}

func (us *UserService) ValidateRole(role string) bool {
	validRoles := map[string]bool{
		"student":    true,
		"instructor": true,
		"admin":      true,
	}
	return validRoles[role]
}
