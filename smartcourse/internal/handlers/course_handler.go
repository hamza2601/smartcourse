package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"smartcourse/internal/services"
)

type CourseHandler struct {
	Service *services.CourseService
}

func NewCourseHandler(service *services.CourseService) *CourseHandler {
	return &CourseHandler{Service: service}
}

type CreateCourseRequest struct {
	Name         string `json:"name" binding:"required"`
	InstructorID uint   `json:"instructor_id" binding:"required"`
}

func (ch *CourseHandler) CreateCourse(c *gin.Context) {
	var req CreateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	course, err := ch.Service.CreateCourse(req.Name, req.InstructorID)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(201, course)
}

func (ch *CourseHandler) GetCourse(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid course id"})
		return
	}

	course, err := ch.Service.GetCourse(uint(id))
	if err != nil {
		c.JSON(404, gin.H{"error": "course not found"})
		return
	}

	c.JSON(200, course)
}

func (ch *CourseHandler) ListCourses(c *gin.Context) {
	courses, err := ch.Service.ListCourses()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, courses)
}
