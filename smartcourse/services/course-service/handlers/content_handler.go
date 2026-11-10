package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"smartcourse/libs/common/auth"
	apperrors "smartcourse/libs/common/errors"
	"smartcourse/services/course-service/services"
)

type ContentHandler struct {
	Service *services.ContentService
}

func NewContentHandler(service *services.ContentService) *ContentHandler {
	return &ContentHandler{Service: service}
}

type CreateModuleRequest struct {
	Title    string `json:"title" binding:"required"`
	Position *int   `json:"position"`
}

type UpdateModuleRequest struct {
	Title    *string `json:"title"`
	Position *int    `json:"position"`
}

type CreateLessonRequest struct {
	Title    string `json:"title" binding:"required"`
	Content  string `json:"content"`
	Position *int   `json:"position"`
}

type UpdateLessonRequest struct {
	Title    *string `json:"title"`
	Content  *string `json:"content"`
	Position *int    `json:"position"`
}

func (h *ContentHandler) CreateModule(c *gin.Context) {
	callerID, ok := caller(c)
	if !ok {
		return
	}
	courseID, ok := idParam(c, "id", "course")
	if !ok {
		return
	}
	var req CreateModuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	module, err := h.Service.CreateModule(c.Request.Context(), callerID, courseID, req.Title, req.Position)
	if err != nil {
		writeContentError(c, err)
		return
	}
	c.JSON(201, module)
}

func (h *ContentHandler) ListModules(c *gin.Context) {
	callerID, ok := caller(c)
	if !ok {
		return
	}
	courseID, ok := idParam(c, "id", "course")
	if !ok {
		return
	}

	modules, err := h.Service.ListModules(c.Request.Context(), callerID, courseID)
	if err != nil {
		writeContentError(c, err)
		return
	}
	c.JSON(200, modules)
}

func (h *ContentHandler) GetCourseContent(c *gin.Context) {
	callerID, ok := caller(c)
	if !ok {
		return
	}
	courseID, ok := idParam(c, "id", "course")
	if !ok {
		return
	}

	content, err := h.Service.GetCourseContent(c.Request.Context(), callerID, courseID)
	if err != nil {
		writeContentError(c, err)
		return
	}
	c.JSON(200, content)
}

func (h *ContentHandler) UpdateModule(c *gin.Context) {
	callerID, ok := caller(c)
	if !ok {
		return
	}
	moduleID, ok := idParam(c, "moduleId", "module")
	if !ok {
		return
	}
	var req UpdateModuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	module, err := h.Service.UpdateModule(c.Request.Context(), callerID, moduleID, req.Title, req.Position)
	if err != nil {
		writeContentError(c, err)
		return
	}
	c.JSON(200, module)
}

func (h *ContentHandler) DeleteModule(c *gin.Context) {
	callerID, ok := caller(c)
	if !ok {
		return
	}
	moduleID, ok := idParam(c, "moduleId", "module")
	if !ok {
		return
	}

	if err := h.Service.DeleteModule(c.Request.Context(), callerID, moduleID); err != nil {
		writeContentError(c, err)
		return
	}
	c.Status(204)
}

func (h *ContentHandler) CreateLesson(c *gin.Context) {
	callerID, ok := caller(c)
	if !ok {
		return
	}
	moduleID, ok := idParam(c, "moduleId", "module")
	if !ok {
		return
	}
	var req CreateLessonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	lesson, err := h.Service.CreateLesson(c.Request.Context(), callerID, moduleID, req.Title, req.Content, req.Position)
	if err != nil {
		writeContentError(c, err)
		return
	}
	c.JSON(201, lesson)
}

func (h *ContentHandler) ListLessons(c *gin.Context) {
	callerID, ok := caller(c)
	if !ok {
		return
	}
	moduleID, ok := idParam(c, "moduleId", "module")
	if !ok {
		return
	}

	lessons, err := h.Service.ListLessons(c.Request.Context(), callerID, moduleID)
	if err != nil {
		writeContentError(c, err)
		return
	}
	c.JSON(200, lessons)
}

func (h *ContentHandler) UpdateLesson(c *gin.Context) {
	callerID, ok := caller(c)
	if !ok {
		return
	}
	lessonID, ok := idParam(c, "lessonId", "lesson")
	if !ok {
		return
	}
	var req UpdateLessonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	lesson, err := h.Service.UpdateLesson(c.Request.Context(), callerID, lessonID, req.Title, req.Content, req.Position)
	if err != nil {
		writeContentError(c, err)
		return
	}
	c.JSON(200, lesson)
}

func (h *ContentHandler) DeleteLesson(c *gin.Context) {
	callerID, ok := caller(c)
	if !ok {
		return
	}
	lessonID, ok := idParam(c, "lessonId", "lesson")
	if !ok {
		return
	}

	if err := h.Service.DeleteLesson(c.Request.Context(), callerID, lessonID); err != nil {
		writeContentError(c, err)
		return
	}
	c.Status(204)
}

// caller returns the user ID from the verified token (never from the body),
// answering 401 itself when there is none.
func caller(c *gin.Context) (uuid.UUID, bool) {
	id, ok := auth.UserIDFrom(c)
	if !ok {
		c.JSON(401, gin.H{"error": "unauthenticated"})
	}
	return id, ok
}

// idParam parses a UUID path parameter, answering 400 itself when it is invalid.
func idParam(c *gin.Context, param, what string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(param))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid " + what + " id"})
		return uuid.Nil, false
	}
	return id, true
}

func writeContentError(c *gin.Context, err error) {
	var (
		courseNotFound *apperrors.CourseNotFoundError
		moduleNotFound *apperrors.ModuleNotFoundError
		lessonNotFound *apperrors.LessonNotFoundError
		notOwned       *apperrors.CourseNotOwnedError
		notEditable    *apperrors.CourseNotEditableError
		positionTaken  *apperrors.PositionConflictError
		dbErr          *apperrors.DatabaseError
	)
	switch {
	case errors.As(err, &courseNotFound):
		c.JSON(404, gin.H{"error": "course not found"})
	case errors.As(err, &moduleNotFound):
		c.JSON(404, gin.H{"error": "module not found"})
	case errors.As(err, &lessonNotFound):
		c.JSON(404, gin.H{"error": "lesson not found"})
	case errors.As(err, &notOwned):
		c.JSON(403, gin.H{"error": "only the course's instructor can change its content"})
	case errors.As(err, &notEditable):
		c.JSON(409, gin.H{"error": notEditable.Error()})
	case errors.As(err, &positionTaken):
		c.JSON(409, gin.H{"error": positionTaken.Error()})
	case errors.As(err, &dbErr):
		c.JSON(500, gin.H{"error": "internal server error"})
	default:
		c.JSON(400, gin.H{"error": err.Error()})
	}
}
