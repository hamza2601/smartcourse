package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	apperrors "smartcourse/internal/errors"
	"smartcourse/internal/services"
)

type UserHandler struct {
	Service *services.UserService
}

func NewUserHandler(service *services.UserService) *UserHandler {
	return &UserHandler{Service: service}
}

type RegisterRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required"`
	Role  string `json:"role" binding:"required"`
}

func (uh *UserHandler) RegisterUser(c *gin.Context) {
	ctx := c.Request.Context()

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	user, err := uh.Service.RegisterUser(ctx, req.Name, req.Email, req.Role)
	if err != nil {
		var existsErr *apperrors.UserAlreadyExistsError
		var dbErr *apperrors.DatabaseError
		switch {
		case errors.As(err, &existsErr):
			c.JSON(409, gin.H{"error": "email already exists"})
		case errors.As(err, &dbErr):
			c.JSON(500, gin.H{"error": "internal server error"})
		default:
			c.JSON(400, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(201, user)
}

func (uh *UserHandler) GetUser(c *gin.Context) {
	ctx := c.Request.Context()

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid user id"})
		return
	}

	user, err := uh.Service.GetUser(ctx, id)
	if err != nil {
		var notFoundErr *apperrors.UserNotFoundError
		var dbErr *apperrors.DatabaseError
		switch {
		case errors.As(err, &notFoundErr):
			c.JSON(404, gin.H{"error": "user not found"})
		case errors.As(err, &dbErr):
			c.JSON(500, gin.H{"error": "internal server error"})
		default:
			c.JSON(400, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(200, user)
}
