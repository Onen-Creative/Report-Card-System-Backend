package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/school-system/backend/internal/models"
	"github.com/school-system/backend/internal/services"
	"gorm.io/gorm"
)

type SchoolHandler struct {
	db           *gorm.DB
	setupService *services.SchoolSetupService
	auditService *services.AuditService
}

func NewSchoolHandler(db *gorm.DB) *SchoolHandler {
	return &SchoolHandler{
		db:           db,
		setupService: services.NewSchoolSetupService(db),
		auditService: services.NewAuditService(db),
	}
}

func (h *SchoolHandler) List(c *gin.Context) {
	var schools []models.School
	if err := h.db.Find(&schools).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, schools)
}

func (h *SchoolHandler) Create(c *gin.Context) {
	var req struct {
		models.School
		Levels []string `json:"levels"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Store levels in config
	if req.School.Config == nil {
		req.School.Config = make(models.JSONB)
	}
	req.School.Config["levels"] = req.Levels

	if err := h.db.Create(&req.School).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Log audit
	if userID, exists := c.Get("user_id"); exists {
		h.auditService.Log(userID.(uuid.UUID), "CREATE", "school", req.School.ID, nil, models.JSONB{"name": req.School.Name}, c.ClientIP())
	}

	// Setup school with classes, subjects, and grading rules
	if err := h.setupService.SetupSchool(&req.School, req.Levels); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to setup school: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, req.School)
}

func (h *SchoolHandler) Get(c *gin.Context) {
	id := c.Param("id")
	var school models.School
	if err := h.db.First(&school, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "School not found"})
		return
	}
	c.JSON(http.StatusOK, school)
}

func (h *SchoolHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var school models.School
	if err := h.db.First(&school, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "School not found"})
		return
	}

	var updateData models.School
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	school.Name = updateData.Name
	school.Type = updateData.Type
	school.Address = updateData.Address
	school.Country = updateData.Country
	school.Region = updateData.Region
	school.ContactEmail = updateData.ContactEmail
	school.Phone = updateData.Phone
	school.LogoURL = updateData.LogoURL
	school.Motto = updateData.Motto
	school.Config = updateData.Config

	if err := h.db.Save(&school).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Setup additional levels if added
	if updateData.Config != nil {
		if newLevels, ok := updateData.Config["levels"].([]interface{}); ok {
			var levels []string
			for _, lvl := range newLevels {
				if level, ok := lvl.(string); ok {
					levels = append(levels, level)
				}
			}
			if err := h.setupService.SetupNewLevels(school.ID, levels); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to setup new levels: " + err.Error()})
				return
			}
		}
	}

	c.JSON(http.StatusOK, school)
}

// SetupSchool sets up classes and subjects for existing schools
func (h *SchoolHandler) SetupSchool(c *gin.Context) {
	id := c.Param("id")
	var school models.School
	if err := h.db.First(&school, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "School not found"})
		return
	}

	// Get levels from school config
	var levels []string
	if school.Config != nil {
		if configLevels, ok := school.Config["levels"].([]interface{}); ok {
			for _, lvl := range configLevels {
				if level, ok := lvl.(string); ok {
					levels = append(levels, level)
				}
			}
		}
	}

	// If no levels in config, use default based on school type
	if len(levels) == 0 {
		switch school.Type {
		case "Primary":
			levels = []string{"P1", "P2", "P3", "P4", "P5", "P6", "P7"}
		case "Secondary":
			levels = []string{"S1", "S2", "S3", "S4", "S5", "S6"}
		case "Nursery":
			levels = []string{"Baby", "Middle", "Top"}
		}
		// Update school config with levels
		if school.Config == nil {
			school.Config = make(models.JSONB)
		}
		school.Config["levels"] = levels
		h.db.Save(&school)
	}

	if err := h.setupService.SetupSchool(&school, levels); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to setup school: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "School setup completed successfully"})
}

func (h *SchoolHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	var school models.School
	if err := h.db.First(&school, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "School not found"})
		return
	}

	// Cascade delete all related data in proper order
	err := h.db.Transaction(func(tx *gorm.DB) error {
		// Delete marks first (depends on assessments)
		if err := tx.Exec("DELETE FROM marks WHERE assessment_id IN (SELECT id FROM assessments WHERE school_id = ?)", id).Error; err != nil {
			return err
		}
		// Delete subject results
		if err := tx.Where("school_id = ?", id).Delete(&models.SubjectResult{}).Error; err != nil {
			return err
		}
		// Delete assessments
		if err := tx.Where("school_id = ?", id).Delete(&models.Assessment{}).Error; err != nil {
			return err
		}
		// Delete report cards
		if err := tx.Exec("DELETE FROM report_cards WHERE student_id IN (SELECT id FROM students WHERE school_id = ?)", id).Error; err != nil {
			return err
		}
		// Delete enrollments
		if err := tx.Exec("DELETE FROM enrollments WHERE student_id IN (SELECT id FROM students WHERE school_id = ?)", id).Error; err != nil {
			return err
		}
		// Delete students
		if err := tx.Where("school_id = ?", id).Delete(&models.Student{}).Error; err != nil {
			return err
		}
		// Delete classes
		if err := tx.Where("school_id = ?", id).Delete(&models.Class{}).Error; err != nil {
			return err
		}
		// Delete subjects
		if err := tx.Where("school_id = ?", id).Delete(&models.Subject{}).Error; err != nil {
			return err
		}
		// Delete grading rules
		if err := tx.Where("school_id = ?", id).Delete(&models.GradingRule{}).Error; err != nil {
			return err
		}
		// Delete users assigned to this school (but not system admins)
		if err := tx.Where("school_id = ? AND role != 'system_admin'", id).Delete(&models.User{}).Error; err != nil {
			return err
		}
		// Finally delete the school
		if err := tx.Delete(&school).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete school: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "School and all related data deleted successfully"})
}

func (h *SchoolHandler) GetStats(c *gin.Context) {
	type Stats struct {
		SchoolsByType    map[string]int64 `json:"schools_by_type"`
		UsersByRole      map[string]int64 `json:"users_by_role"`
		UsersBySchool    []struct {
			SchoolName string `json:"school_name"`
			UserCount  int64  `json:"user_count"`
		} `json:"users_by_school"`
		TotalStudents    int64 `json:"total_students"`
		TotalSchools     int64 `json:"total_schools"`
		TotalUsers       int64 `json:"total_users"`
		StudentsBySchool []struct {
			SchoolName    string `json:"school_name"`
			StudentCount  int64  `json:"student_count"`
		} `json:"students_by_school"`
	}

	stats := Stats{
		SchoolsByType: make(map[string]int64),
		UsersByRole:   make(map[string]int64),
	}

	// Schools by type
	var schoolTypeResults []struct {
		Type  string
		Count int64
	}
	h.db.Model(&models.School{}).Select("type, COUNT(*) as count").Group("type").Scan(&schoolTypeResults)
	for _, result := range schoolTypeResults {
		stats.SchoolsByType[result.Type] = result.Count
	}

	// Users by role
	var userRoleResults []struct {
		Role  string
		Count int64
	}
	h.db.Model(&models.User{}).Select("role, COUNT(*) as count").Group("role").Scan(&userRoleResults)
	for _, result := range userRoleResults {
		stats.UsersByRole[result.Role] = result.Count
	}

	// Users by school (show all schools with their user counts, including zero)
	h.db.Model(&models.School{}).
		Select("schools.name as school_name, COUNT(DISTINCT users.id) as user_count").
		Joins("LEFT JOIN users ON schools.id = users.school_id AND users.deleted_at IS NULL").
		Group("schools.id, schools.name").
		Scan(&stats.UsersBySchool)

	// Students by school (show all schools with their student counts, including zero)
	h.db.Model(&models.School{}).
		Select("schools.name as school_name, COUNT(DISTINCT students.id) as student_count").
		Joins("LEFT JOIN students ON schools.id = students.school_id AND students.deleted_at IS NULL").
		Group("schools.id, schools.name").
		Scan(&stats.StudentsBySchool)

	// Total counts
	h.db.Model(&models.School{}).Count(&stats.TotalSchools)
	h.db.Model(&models.User{}).Count(&stats.TotalUsers)
	h.db.Model(&models.Student{}).Count(&stats.TotalStudents)

	c.JSON(http.StatusOK, stats)
}

