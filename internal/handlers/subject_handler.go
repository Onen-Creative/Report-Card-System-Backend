package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/school-system/backend/internal/models"
	"github.com/school-system/backend/internal/services"
	"gorm.io/gorm"
)

type SubjectHandler struct {
	db                     *gorm.DB
	standardSubjectService *services.StandardSubjectService
}

func NewSubjectHandler(db *gorm.DB) *SubjectHandler {
	return &SubjectHandler{
		db:                     db,
		standardSubjectService: services.NewStandardSubjectService(db),
	}
}

func (h *SubjectHandler) List(c *gin.Context) {
	level := c.Query("level")

	if level != "" {
		// Always return standardized subjects for the level to ensure consistency
		standardSubjects, err := h.standardSubjectService.GetSubjectsForLevel(level)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Convert to regular subject format for API compatibility
		var subjects []models.Subject
		for _, std := range standardSubjects {
			subjects = append(subjects, models.Subject{
				BaseModel:    models.BaseModel{ID: std.ID},
				Name:         std.Name,
				Code:         std.Code,
				Level:        std.Level,
				IsCompulsory: std.IsCompulsory,
				Papers:       std.Papers,
			})
		}
		c.JSON(http.StatusOK, subjects)
		return
	}

	// If no level specified, return all standard subjects grouped by level
	var standardSubjects []models.StandardSubject
	if err := h.db.Find(&standardSubjects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to regular subject format for API compatibility
	var subjects []models.Subject
	for _, std := range standardSubjects {
		subjects = append(subjects, models.Subject{
			BaseModel:    models.BaseModel{ID: std.ID},
			Name:         std.Name,
			Code:         std.Code,
			Level:        std.Level,
			IsCompulsory: std.IsCompulsory,
			Papers:       std.Papers,
		})
	}
	c.JSON(http.StatusOK, subjects)
}

func (h *SubjectHandler) Create(c *gin.Context) {
	// Schools cannot create custom subjects - only standard subjects are allowed
	c.JSON(http.StatusForbidden, gin.H{
		"error":   "Custom subjects not allowed",
		"message": "All schools must use standardized curriculum subjects. Contact system administrator to add new standard subjects.",
	})
}

func (h *SubjectHandler) Update(c *gin.Context) {
	// Schools cannot modify standard subjects
	c.JSON(http.StatusForbidden, gin.H{
		"error":   "Cannot modify standard subjects",
		"message": "Standard curriculum subjects cannot be modified. Contact system administrator for curriculum changes.",
	})
}

func (h *SubjectHandler) Delete(c *gin.Context) {
	// Schools cannot delete standard subjects
	c.JSON(http.StatusForbidden, gin.H{
		"error":   "Cannot delete standard subjects",
		"message": "Standard curriculum subjects cannot be deleted. Contact system administrator for curriculum changes.",
	})
}

func (h *SubjectHandler) GetLevels(c *gin.Context) {
	levels, err := h.standardSubjectService.GetAllLevels()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"levels": levels})
}

func (h *SubjectHandler) CreateStandardSubject(c *gin.Context) {
	var standardSubject models.StandardSubject
	if err := c.ShouldBindJSON(&standardSubject); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existing models.StandardSubject
	err := h.db.Where("name = ? AND level = ?", standardSubject.Name, standardSubject.Level).First(&existing).Error
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Standard subject already exists for this level"})
		return
	}

	if err := h.db.Create(&standardSubject).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, standardSubject)
}

func (h *SubjectHandler) UpdateStandardSubject(c *gin.Context) {
	id := c.Param("id")
	var standardSubject models.StandardSubject
	if err := h.db.First(&standardSubject, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Standard subject not found"})
		return
	}

	if err := c.ShouldBindJSON(&standardSubject); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.Save(&standardSubject).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, standardSubject)
}

func (h *SubjectHandler) DeleteStandardSubject(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&models.StandardSubject{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Standard subject deleted"})
}

func (h *SubjectHandler) ListStandardSubjects(c *gin.Context) {
	level := c.Query("level")
	var standardSubjects []models.StandardSubject

	query := h.db
	if level != "" {
		query = query.Where("level = ?", level)
	}

	if err := query.Find(&standardSubjects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, standardSubjects)
}