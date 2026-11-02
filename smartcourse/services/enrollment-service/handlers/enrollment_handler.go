package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"smartcourse/libs/common/auth"
	apperrors "smartcourse/libs/common/errors"
	"smartcourse/services/enrollment-service/services"
)

type EnrollmentHandler struct {
	Service *services.EnrollmentService
}

func NewEnrollmentHandler(service *services.EnrollmentService) *EnrollmentHandler {
	return &EnrollmentHandler{Service: service}
}

// EnrollRequest has no student_id: the student is always the caller from the
// verified token; a student_id sent in the body is ignored.
type EnrollRequest struct {
	CourseID uuid.UUID `json:"course_id" binding:"required"`
}

func (eh *EnrollmentHandler) Enroll(c *gin.Context) {
	ctx := c.Request.Context()

	studentID, ok := auth.UserIDFrom(c)
	if !ok {
		c.JSON(401, gin.H{"error": "unauthenticated"})
		return
	}

	var req EnrollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	enrollment, err := eh.Service.EnrollStudent(ctx, studentID, req.CourseID)
	if err != nil {
		var existsErr *apperrors.EnrollmentAlreadyExistsError
		var fullErr *apperrors.CourseFullError
		var notPublishedErr *apperrors.CourseNotPublishedError
		var dbErr *apperrors.DatabaseError
		var upstreamErr *apperrors.UpstreamError
		switch {
		case errors.As(err, &existsErr):
			c.JSON(409, gin.H{"error": "student already enrolled"})
		case errors.As(err, &fullErr):
			c.JSON(409, gin.H{"error": "course is full"})
		case errors.As(err, &notPublishedErr):
			c.JSON(409, gin.H{"error": "course is not open for enrollment"})
		case errors.As(err, &dbErr):
			c.JSON(500, gin.H{"error": "internal server error"})
		case errors.As(err, &upstreamErr):
			c.JSON(502, gin.H{"error": "dependent service unavailable"})
		default:
			c.JSON(400, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(201, enrollment)
}

func (eh *EnrollmentHandler) GetEnrollment(c *gin.Context) {
	ctx := c.Request.Context()

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid enrollment id"})
		return
	}

	enrollment, err := eh.Service.GetEnrollment(ctx, id)
	if err != nil {
		var notFoundErr *apperrors.EnrollmentNotFoundError
		var dbErr *apperrors.DatabaseError
		switch {
		case errors.As(err, &notFoundErr):
			c.JSON(404, gin.H{"error": "enrollment not found"})
		case errors.As(err, &dbErr):
			c.JSON(500, gin.H{"error": "internal server error"})
		default:
			c.JSON(400, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(200, enrollment)
}
