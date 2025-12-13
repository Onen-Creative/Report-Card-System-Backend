package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/school-system/backend/internal/models"
	"gorm.io/gorm"
)

type ClassHandler struct {
	db *gorm.DB
}

func NewClassHandler(db *gorm.DB) *ClassHandler {
	return &ClassHandler{db: db}
}

func (h *ClassHandler) List(c *gin.Context) {
	year := c.Query("year")
	term := c.Query("term")
	schoolID := c.GetString("tenant_school_id")

	var classes []models.Class
	query := h.db.Preload("Teacher")

	// Filter by school for non-system admins
	if schoolID != "" {
		query = query.Where("school_id = ?", schoolID)
	}

	if year != "" {
		query = query.Where("year = ?", year)
	}
	if term != "" {
		query = query.Where("term = ?", term)
	}

	if err := query.Find(&classes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, classes)
}

func (h *ClassHandler) Create(c *gin.Context) {
	var class models.Class
	if err := c.ShouldBindJSON(&class); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Auto-assign class to user's school
	userRole := c.GetString("user_role")
	if userRole != "system_admin" {
		tenantSchoolID := c.GetString("tenant_school_id")
		if tenantSchoolID == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "No school assigned to user"})
			return
		}
		schoolID, _ := uuid.Parse(tenantSchoolID)
		class.SchoolID = schoolID
	}

	if err := h.db.Create(&class).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, class)
}

func (h *ClassHandler) Get(c *gin.Context) {
	id := c.Param("id")
	var class models.Class
	if err := h.db.Preload("Teacher").Preload("School").First(&class, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Class not found"})
		return
	}
	c.JSON(http.StatusOK, class)
}

func (h *ClassHandler) GetStudents(c *gin.Context) {
	classID := c.Param("id")
	year, _ := strconv.Atoi(c.Query("year"))
	term := c.Query("term")

	var enrollments []models.Enrollment
	query := h.db.Preload("Student").Where("class_id = ?", classID)
	if year > 0 {
		query = query.Where("year = ?", year)
	}
	if term != "" {
		query = query.Where("term = ?", term)
	}

	if err := query.Find(&enrollments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	students := make([]models.Student, len(enrollments))
	for i, e := range enrollments {
		students[i] = *e.Student
	}

	c.JSON(http.StatusOK, students)
}

func (h *ClassHandler) GetLevels(c *gin.Context) {
	schoolID := c.GetString("tenant_school_id")

	type LevelResult struct {
		Level string `json:"level"`
	}

	var levels []LevelResult
	query := h.db.Model(&models.Class{}).Select("DISTINCT level").Order("level")

	// Filter by school for non-system admins
	if schoolID != "" {
		query = query.Where("school_id = ?", schoolID)
	}

	if err := query.Scan(&levels).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, levels)
}
