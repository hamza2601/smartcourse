package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	apperrors "smartcourse/internal/errors"
	"smartcourse/internal/services"
)

type EnrollmentHandler struct {
	Service *services.EnrollmentService
}

func NewEnrollmentHandler(service *services.EnrollmentService) *EnrollmentHandler {
	return &EnrollmentHandler{Service: service}
}

type EnrollRequest struct {
	StudentID uuid.UUID `json:"student_id" binding:"required"`
	CourseID  uuid.UUID `json:"course_id" binding:"required"`
}

func (eh *EnrollmentHandler) Enroll(c *gin.Context) {
	ctx := c.Request.Context()

	var req EnrollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	enrollment, err := eh.Service.EnrollStudent(ctx, req.StudentID, req.CourseID)
	if err != nil {
		var existsErr *apperrors.EnrollmentAlreadyExistsError
		var dbErr *apperrors.DatabaseError
		switch {
		case errors.As(err, &existsErr):
			c.JSON(409, gin.H{"error": "student already enrolled"})
		case errors.As(err, &dbErr):
			c.JSON(500, gin.H{"error": "internal server error"})
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
