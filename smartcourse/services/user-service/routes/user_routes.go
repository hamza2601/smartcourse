package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"smartcourse/libs/common/auth"
	"smartcourse/services/user-service/config"
	"smartcourse/services/user-service/domain"
	"smartcourse/services/user-service/handlers"
	"smartcourse/services/user-service/repositories"
	"smartcourse/services/user-service/services"
)

// SetupUserServiceRoutes returns the public API router and the internal
// (/internal/*) router. They are served on separate ports; only the public
// one may be exposed outside the service network or routed by a gateway.
func SetupUserServiceRoutes(db *gorm.DB, cfg *config.Config) (public, internal *gin.Engine) {
	userRepo := repositories.NewUserRepository(db)
	refreshTokenRepo := repositories.NewRefreshTokenRepository(db)

	tokenService := services.NewTokenService(cfg.JWT.AccessTokenExpiry, cfg.JWT.RefreshTokenExpiry)
	userService := services.NewUserService(userRepo)
	authService := services.NewAuthService(userRepo, refreshTokenRepo, tokenService, cfg.JWT)

	userHandler := handlers.NewUserHandler(userService)
	authHandler := handlers.NewAuthHandler(authService)

	public = gin.Default()
	public.POST("/auth/login", authHandler.Login)
	public.POST("/auth/refresh", authHandler.Refresh)

	v1 := public.Group("/api/v1", auth.AuthMiddleware(tokenService.AccessTokenVerifier(cfg.JWT.SecretKey)))
	{
		v1.POST("/users", auth.RequireRole(domain.RoleAdmin), userHandler.RegisterUser)
		v1.GET("/users/:id", userHandler.GetUser)
	}

	internal = gin.Default()
	internalAPI := internal.Group("/internal", auth.InternalAuth(cfg.InternalAPIKey))
	{
		internalAPI.GET("/users/:id", userHandler.GetUser)
	}

	return public, internal
}
