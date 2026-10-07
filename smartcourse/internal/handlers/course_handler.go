package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	apperrors "smartcourse/internal/errors"
	"smartcourse/internal/services"
)

type CourseHandler struct {
	Service *services.CourseService
}

func NewCourseHandler(service *services.CourseService) *CourseHandler {
	return &CourseHandler{Service: service}
}

type CreateCourseRequest struct {
	Name         string    `json:"name" binding:"required"`
	InstructorID uuid.UUID `json:"instructor_id" binding:"required"`
}

func (ch *CourseHandler) CreateCourse(c *gin.Context) {
	ctx := c.Request.Context()

	var req CreateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	course, err := ch.Service.CreateCourse(ctx, req.Name, req.InstructorID)
	if err != nil {
		var dbErr *apperrors.DatabaseError
		if errors.As(err, &dbErr) {
			c.JSON(500, gin.H{"error": "internal server error"})
			return
		}
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(201, course)
}

func (ch *CourseHandler) GetCourse(c *gin.Context) {
	ctx := c.Request.Context()

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid course id"})
		return
	}

	course, err := ch.Service.GetCourse(ctx, id)
	if err != nil {
		var notFoundErr *apperrors.CourseNotFoundError
		var dbErr *apperrors.DatabaseError
		switch {
		case errors.As(err, &notFoundErr):
			c.JSON(404, gin.H{"error": "course not found"})
		case errors.As(err, &dbErr):
			c.JSON(500, gin.H{"error": "internal server error"})
		default:
			c.JSON(400, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(200, course)
}

func (ch *CourseHandler) ListCourses(c *gin.Context) {
	ctx := c.Request.Context()

	courses, err := ch.Service.ListCourses(ctx)
	if err != nil {
		c.JSON(500, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(200, courses)
}
