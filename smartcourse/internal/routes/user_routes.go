package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"smartcourse/internal/config"
	"smartcourse/internal/domain"
	"smartcourse/internal/handlers"
	"smartcourse/internal/middleware"
	"smartcourse/internal/repositories"
	"smartcourse/internal/services"
)

func SetupUserServiceRoutes(db *gorm.DB, cfg *config.Config) *gin.Engine {
	router := gin.Default()

	userRepo := repositories.NewUserRepository(db)
	refreshTokenRepo := repositories.NewRefreshTokenRepository(db)

	tokenService := services.NewTokenService(cfg.JWT.AccessTokenExpiry, cfg.JWT.RefreshTokenExpiry)
	userService := services.NewUserService(userRepo)
	authService := services.NewAuthService(userRepo, refreshTokenRepo, tokenService, cfg.JWT)

	userHandler := handlers.NewUserHandler(userService)
	authHandler := handlers.NewAuthHandler(authService)

	// Public routes (no auth required)
	router.POST("/auth/login", authHandler.Login)
	router.POST("/auth/refresh", authHandler.Refresh)

	// Internal API endpoints (for other services)
	router.GET("/internal/users/:id", userHandler.GetUser)

	// Protected routes (auth required)
	v1 := router.Group("/api/v1", middleware.AuthMiddleware(tokenService, cfg.JWT.SecretKey))
	{
		v1.POST("/users", middleware.RequireRole(domain.RoleAdmin), userHandler.RegisterUser)
		v1.GET("/users/:id", userHandler.GetUser)
	}

	return router
}
