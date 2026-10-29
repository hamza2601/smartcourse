package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"smartcourse/libs/common/auth"
	"smartcourse/services/enrollment-service/config"
	"smartcourse/services/enrollment-service/handlers"
	"smartcourse/services/enrollment-service/repositories"
	"smartcourse/services/enrollment-service/services"
)

// Hardcoded for now; resolves via the docker-compose network.
const userServiceURL = "http://user-service:8081"
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
	v1 := router.Group("/api/v1", auth.AuthMiddleware(tokenService.AccessTokenVerifier(cfg.JWT.SecretKey)))
	{
		v1.POST("/enrollments", enrollmentHandler.Enroll)
		v1.GET("/enrollments/:id", enrollmentHandler.GetEnrollment)
	}

	return router
}
