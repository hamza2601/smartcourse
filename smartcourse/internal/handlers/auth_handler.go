package handlers

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"

	apperrors "smartcourse/internal/errors"
	"smartcourse/internal/services"
)

type AuthHandler struct {
	Service *services.AuthService
}

func NewAuthHandler(service *services.AuthService) *AuthHandler {
	return &AuthHandler{Service: service}
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (ah *AuthHandler) Login(c *gin.Context) {
	ctx := c.Request.Context()

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	tokens, err := ah.Service.Login(ctx, req.Email, req.Password)
	if err != nil {
		writeAuthError(c, err)
		return
	}

	c.JSON(200, tokens)
}

func (ah *AuthHandler) Refresh(c *gin.Context) {
	ctx := c.Request.Context()

	refreshToken := refreshTokenFromRequest(c)
	if refreshToken == "" {
		c.JSON(401, gin.H{"error": "refresh token required"})
		return
	}

	tokens, err := ah.Service.RefreshAccessToken(ctx, refreshToken)
	if err != nil {
		writeAuthError(c, err)
		return
	}

	c.JSON(200, tokens)
}

// The refresh token comes from "Authorization: Bearer <token>" or, failing
// that, the "refresh_token" cookie.
func refreshTokenFromRequest(c *gin.Context) string {
	if token, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer "); ok && token != "" {
		return token
	}
	if cookie, err := c.Cookie("refresh_token"); err == nil {
		return cookie
	}
	return ""
}

func writeAuthError(c *gin.Context, err error) {
	var credsErr *apperrors.InvalidCredentialsError
	var tokenErr *apperrors.InvalidTokenError
	var dbErr *apperrors.DatabaseError
	switch {
	case errors.As(err, &credsErr):
		c.JSON(401, gin.H{"error": "invalid email or password"})
	case errors.As(err, &tokenErr):
		c.JSON(401, gin.H{"error": "invalid or expired token"})
	case errors.As(err, &dbErr):
		c.JSON(500, gin.H{"error": "internal server error"})
	default:
		c.JSON(500, gin.H{"error": "internal server error"})
	}
}
