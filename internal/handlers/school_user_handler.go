package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/school-system/backend/internal/services"
	"gorm.io/gorm"
)

type SchoolUserHandler struct {
	db                    *gorm.DB
	userAssignmentService *services.UserAssignmentService
}

func NewSchoolUserHandler(db *gorm.DB) *SchoolUserHandler {
	return &SchoolUserHandler{
		db:                    db,
		userAssignmentService: services.NewUserAssignmentService(db),
	}
}

// GetSchoolUsers returns all users for a specific school
func (h *SchoolUserHandler) GetSchoolUsers(c *gin.Context) {
	schoolIDStr := c.Param("schoolId")
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid school ID"})
		return
	}

	users, err := h.userAssignmentService.GetSchoolUsers(schoolID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

// CreateTeacher creates a new teacher for a school
func (h *SchoolUserHandler) CreateTeacher(c *gin.Context) {
	schoolIDStr := c.Param("schoolId")
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid school ID"})
		return
	}

	var req struct {
		FullName string `json:"full_name" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	teacher, err := h.userAssignmentService.CreateTeacher(schoolID, req.FullName, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, teacher)
}

// AssignTeacherToClass assigns a teacher to a specific class
func (h *SchoolUserHandler) AssignTeacherToClass(c *gin.Context) {
	var req struct {
		TeacherID string `json:"teacher_id" binding:"required"`
		ClassID   string `json:"class_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	teacherID, err := uuid.Parse(req.TeacherID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid teacher ID"})
		return
	}

	classID, err := uuid.Parse(req.ClassID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid class ID"})
		return
	}

	if err := h.userAssignmentService.AssignTeacherToClass(teacherID, classID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Teacher assigned to class successfully"})
}

// UpdateUserRole updates a user's role within the school
func (h *SchoolUserHandler) UpdateUserRole(c *gin.Context) {
	userIDStr := c.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.userAssignmentService.UpdateUserRole(userID, req.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User role updated successfully"})
}