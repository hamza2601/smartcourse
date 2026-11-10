package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"smartcourse/libs/common/auth"
	"smartcourse/services/course-service/config"
	"smartcourse/services/course-service/domain"
	"smartcourse/services/course-service/handlers"
	"smartcourse/services/course-service/repositories"
	"smartcourse/services/course-service/services"
)

// SetupCourseServiceRoutes returns the public API router and the internal
// (/internal/*) router. They are served on separate ports; only the public
// one may be exposed outside the service network or routed by a gateway.
func SetupCourseServiceRoutes(db *gorm.DB, cfg *config.Config) (public, internal *gin.Engine) {
	courseRepo := repositories.NewCourseRepository(db)
	contentRepo := repositories.NewContentRepository(db)

	tokenService := services.NewTokenService(cfg.JWT.AccessTokenExpiry, cfg.JWT.RefreshTokenExpiry)
	courseService := services.NewCourseService(courseRepo)

	courseHandler := handlers.NewCourseHandler(courseService)
	contentHandler := handlers.NewContentHandler(services.NewContentService(courseRepo, contentRepo))

	public = gin.Default()
	v1 := public.Group("/api/v1", auth.AuthMiddleware(tokenService.AccessTokenVerifier(cfg.JWT.SecretKey)))
	{
		v1.POST("/courses", auth.RequireRole(domain.RoleInstructor), courseHandler.CreateCourse)
		v1.POST("/courses/:id/publish", auth.RequireRole(domain.RoleInstructor), courseHandler.PublishCourse)
		v1.GET("/courses/:id", courseHandler.GetCourse)
		v1.GET("/courses", courseHandler.ListCourses)

		v1.POST("/courses/:id/modules", auth.RequireRole(domain.RoleInstructor), contentHandler.CreateModule)
		v1.GET("/courses/:id/modules", contentHandler.ListModules)
		v1.GET("/courses/:id/content", contentHandler.GetCourseContent)
		v1.PUT("/modules/:moduleId", auth.RequireRole(domain.RoleInstructor), contentHandler.UpdateModule)
		v1.DELETE("/modules/:moduleId", auth.RequireRole(domain.RoleInstructor), contentHandler.DeleteModule)
		v1.POST("/modules/:moduleId/lessons", auth.RequireRole(domain.RoleInstructor), contentHandler.CreateLesson)
		v1.GET("/modules/:moduleId/lessons", contentHandler.ListLessons)
		v1.PUT("/lessons/:lessonId", auth.RequireRole(domain.RoleInstructor), contentHandler.UpdateLesson)
		v1.DELETE("/lessons/:lessonId", auth.RequireRole(domain.RoleInstructor), contentHandler.DeleteLesson)
	}

	internal = gin.Default()
	internalAPI := internal.Group("/internal", auth.InternalAuth(cfg.InternalAPIKey))
	{
		internalAPI.GET("/courses/:id", courseHandler.GetCourse)
	}

	return public, internal
}
