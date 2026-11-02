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

// SetupEnrollmentServiceRoutes returns the public API router and the internal
// (/internal/*) router. They are served on separate ports; only the public
// one may be exposed outside the service network or routed by a gateway.
func SetupEnrollmentServiceRoutes(db *gorm.DB, cfg *config.Config) (public, internal *gin.Engine) {
	enrollmentRepo := repositories.NewEnrollmentRepository(db)

	tokenService := services.NewTokenService(cfg.JWT.AccessTokenExpiry, cfg.JWT.RefreshTokenExpiry)
	enrollmentService := services.NewEnrollmentService(enrollmentRepo, cfg.CourseServiceURL, cfg.InternalAPIKey)

	enrollmentHandler := handlers.NewEnrollmentHandler(enrollmentService)

	public = gin.Default()
	v1 := public.Group("/api/v1", auth.AuthMiddleware(tokenService.AccessTokenVerifier(cfg.JWT.SecretKey)))
	{
		v1.POST("/enrollments", enrollmentHandler.Enroll)
		v1.GET("/enrollments/:id", enrollmentHandler.GetEnrollment)
	}

	internal = gin.Default()
	internalAPI := internal.Group("/internal", auth.InternalAuth(cfg.InternalAPIKey))
	{
		internalAPI.GET("/enrollments/:id", enrollmentHandler.GetEnrollment)
	}

	return public, internal
}
