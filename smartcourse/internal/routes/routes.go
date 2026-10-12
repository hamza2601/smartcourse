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

func SetupRoutes(db *gorm.DB, cfg *config.Config) *gin.Engine {
	router := gin.Default()

	userRepo := repositories.NewUserRepository(db)
	courseRepo := repositories.NewCourseRepository(db)
	enrollmentRepo := repositories.NewEnrollmentRepository(db)
	refreshTokenRepo := repositories.NewRefreshTokenRepository(db)

	tokenService := services.NewTokenService(cfg.JWT.AccessTokenExpiry, cfg.JWT.RefreshTokenExpiry)
	userService := services.NewUserService(userRepo)
	courseService := services.NewCourseService(courseRepo, userRepo)
	enrollmentService := services.NewEnrollmentService(enrollmentRepo)
	authService := services.NewAuthService(userRepo, refreshTokenRepo, tokenService, cfg.JWT)

	userHandler := handlers.NewUserHandler(userService)
	courseHandler := handlers.NewCourseHandler(courseService)
	enrollmentHandler := handlers.NewEnrollmentHandler(enrollmentService)
	authHandler := handlers.NewAuthHandler(authService)

	router.POST("/auth/login", authHandler.Login)
	router.POST("/auth/refresh", authHandler.Refresh)

	v1 := router.Group("/api/v1", middleware.AuthMiddleware(tokenService, cfg.JWT.SecretKey))
	{
		v1.POST("/users", middleware.RequireRole(domain.RoleAdmin), userHandler.RegisterUser)
		v1.GET("/users/:id", userHandler.GetUser)

		v1.POST("/courses", middleware.RequireRole(domain.RoleInstructor, domain.RoleAdmin), courseHandler.CreateCourse)
		v1.GET("/courses/:id", courseHandler.GetCourse)
		v1.GET("/courses", courseHandler.ListCourses)

		v1.POST("/enrollments", enrollmentHandler.Enroll)
		v1.GET("/enrollments/:id", enrollmentHandler.GetEnrollment)
	}

	return router
}
