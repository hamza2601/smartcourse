package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"smartcourse/internal/config"
	"smartcourse/internal/handlers"
	"smartcourse/internal/middleware"
	"smartcourse/internal/repositories"
	"smartcourse/internal/services"
)

// Hardcoded for now; resolves via the docker-compose network.
// (userServiceURL is declared in course_routes.go, same package.)
const courseServiceURL = "http://course-service:8082"

func SetupEnrollmentServiceRoutes(db *gorm.DB, cfg *config.Config) *gin.Engine {
	router := gin.Default()

	enrollmentRepo := repositories.NewEnrollmentRepository(db)

	tokenService := services.NewTokenService(cfg.JWT.AccessTokenExpiry, cfg.JWT.RefreshTokenExpiry)
	enrollmentService := services.NewEnrollmentService(enrollmentRepo, courseServiceURL, userServiceURL)

	enrollmentHandler := handlers.NewEnrollmentHandler(enrollmentService)

	// Internal API endpoints (for other services)
	router.GET("/internal/enrollments/:id", enrollmentHandler.GetEnrollment)

	// Protected routes (auth required)
	v1 := router.Group("/api/v1", middleware.AuthMiddleware(tokenService, cfg.JWT.SecretKey))
	{
		v1.POST("/enrollments", enrollmentHandler.Enroll)
		v1.GET("/enrollments/:id", enrollmentHandler.GetEnrollment)
	}

	return router
}
