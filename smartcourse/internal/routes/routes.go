package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"smartcourse/internal/handlers"
	"smartcourse/internal/repositories"
	"smartcourse/internal/services"
)

func SetupRoutes(db *gorm.DB) *gin.Engine {
	router := gin.Default()

	userRepo := repositories.NewUserRepository(db)
	courseRepo := repositories.NewCourseRepository(db)

	userService := services.NewUserService(userRepo)
	courseService := services.NewCourseService(courseRepo, userRepo)

	userHandler := handlers.NewUserHandler(userService)
	courseHandler := handlers.NewCourseHandler(courseService)

	v1 := router.Group("/api/v1")
	{
		v1.POST("/users", userHandler.RegisterUser)
		v1.GET("/users/:id", userHandler.GetUser)

		v1.POST("/courses", courseHandler.CreateCourse)
		v1.GET("/courses/:id", courseHandler.GetCourse)
		v1.GET("/courses", courseHandler.ListCourses)
	}

	return router
}
