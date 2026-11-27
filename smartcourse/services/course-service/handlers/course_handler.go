package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"smartcourse/libs/common/auth"
	apperrors "smartcourse/libs/common/errors"
	"smartcourse/services/course-service/services"
	"smartcourse/services/course-service/workflows"
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

	if err := ch.Service.PublishCourse(ctx, id, callerID); err != nil {
		var notFoundErr *apperrors.CourseNotFoundError
		var notOwnedErr *apperrors.CourseNotOwnedError
		var conflictErr *apperrors.CourseStatusConflictError
		switch {
		case errors.As(err, &notFoundErr):
			c.JSON(404, gin.H{"error": "course not found"})
		case errors.As(err, &notOwnedErr):
			c.JSON(403, gin.H{"error": "only the course's instructor can publish it"})
		case errors.As(err, &conflictErr):
			c.JSON(409, gin.H{"error": conflictErr.Error()})
		case errors.Is(err, workflows.ErrPublishInProgress):
			c.JSON(409, gin.H{"error": "publishing already in progress"})
		case errors.Is(err, workflows.ErrPublishUnavailable):
			c.JSON(503, gin.H{"error": "publishing service unavailable"})
		default:
			c.JSON(500, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(202, gin.H{
		"course_id":   id,
		"workflow_id": workflows.PublishWorkflowID(id),
		"message":     "publishing started",
	})
}

// UpdateCourseRequest fields are all optional; omitted fields stay unchanged.
type UpdateCourseRequest struct {
	Name            *string `json:"name"`
	Description     *string `json:"description"`
	EnrollmentLimit *int    `json:"enrollment_limit"`
}

func (ch *CourseHandler) UpdateCourse(c *gin.Context) {
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
	var req UpdateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	course, err := ch.Service.UpdateCourse(ctx, id, callerID, req.Name, req.Description, req.EnrollmentLimit)
	if err != nil {
		var notFoundErr *apperrors.CourseNotFoundError
		var notOwnedErr *apperrors.CourseNotOwnedError
		var notEditableErr *apperrors.CourseNotEditableError
		var dbErr *apperrors.DatabaseError
		switch {
		case errors.As(err, &notFoundErr):
			c.JSON(404, gin.H{"error": "course not found"})
		case errors.As(err, &notOwnedErr):
			c.JSON(403, gin.H{"error": "only the course's instructor can update it"})
		case errors.As(err, &notEditableErr):
			c.JSON(409, gin.H{"error": notEditableErr.Error()})
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
