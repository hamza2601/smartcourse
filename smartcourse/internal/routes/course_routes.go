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

// Hardcoded for now; resolves via the docker-compose network.
const userServiceURL = "http://user-service:8081"

func SetupCourseServiceRoutes(db *gorm.DB, cfg *config.Config) *gin.Engine {
	router := gin.Default()

	courseRepo := repositories.NewCourseRepository(db)

	tokenService := services.NewTokenService(cfg.JWT.AccessTokenExpiry, cfg.JWT.RefreshTokenExpiry)
	courseService := services.NewCourseService(courseRepo, userServiceURL)

	courseHandler := handlers.NewCourseHandler(courseService)

	// Internal API endpoints (for other services)
	router.GET("/internal/courses/:id", courseHandler.GetCourse)

	// Protected routes (auth required)
	v1 := router.Group("/api/v1", middleware.AuthMiddleware(tokenService, cfg.JWT.SecretKey))
	{
		v1.POST("/courses", middleware.RequireRole(domain.RoleInstructor, domain.RoleAdmin), courseHandler.CreateCourse)
		v1.GET("/courses/:id", courseHandler.GetCourse)
		v1.GET("/courses", courseHandler.ListCourses)
	}

	return router
}
