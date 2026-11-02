package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"smartcourse/libs/common/auth"
	apperrors "smartcourse/libs/common/errors"
	"smartcourse/services/course-service/services"
)

type CourseHandler struct {
	Service *services.CourseService
}

func NewCourseHandler(service *services.CourseService) *CourseHandler {
	return &CourseHandler{Service: service}
}

// CreateCourseRequest has no instructor_id: the owner is always the caller
// from the verified token; an instructor_id sent in the body is ignored.
type CreateCourseRequest struct {
	Name            string `json:"name" binding:"required"`
	Description     string `json:"description"`
	EnrollmentLimit *int   `json:"enrollment_limit" binding:"omitempty,gt=0"`
}

func (ch *CourseHandler) CreateCourse(c *gin.Context) {
	ctx := c.Request.Context()

	instructorID, ok := auth.UserIDFrom(c)
	if !ok {
		c.JSON(401, gin.H{"error": "unauthenticated"})
		return
	}

	var req CreateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	course, err := ch.Service.CreateCourse(ctx, instructorID, req.Name, req.Description, req.EnrollmentLimit)
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

func (ch *CourseHandler) PublishCourse(c *gin.Context) {
	ctx := c.Request.Context()

	callerID, ok := auth.UserIDFrom(c)
	if !ok {
		c.JSON(401, gin.H{"error": "unauthenticated"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid course id"})
		return
	}

	course, err := ch.Service.PublishCourse(ctx, id, callerID)
	if err != nil {
		var notFoundErr *apperrors.CourseNotFoundError
		var notOwnedErr *apperrors.CourseNotOwnedError
		var conflictErr *apperrors.CourseStatusConflictError
		var dbErr *apperrors.DatabaseError
		switch {
		case errors.As(err, &notFoundErr):
			c.JSON(404, gin.H{"error": "course not found"})
		case errors.As(err, &notOwnedErr):
			c.JSON(403, gin.H{"error": "only the course's instructor can publish it"})
		case errors.As(err, &conflictErr):
			c.JSON(409, gin.H{"error": conflictErr.Error()})
		case errors.As(err, &dbErr):
			c.JSON(500, gin.H{"error": "internal server error"})
		default:
			c.JSON(400, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(200, course)
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
