package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/school-system/backend/internal/models"
	"gorm.io/gorm"
)

type StudentHandler struct {
	db *gorm.DB
}

func NewStudentHandler(db *gorm.DB) *StudentHandler {
	return &StudentHandler{db: db}
}

func (h *StudentHandler) List(c *gin.Context) {
	classID := c.Query("class_id")
	classLevel := c.Query("class_level")
	term := c.Query("term")
	year := c.Query("year")
	schoolID := c.GetString("tenant_school_id")

	type StudentWithClass struct {
		models.Student
		ClassName string    `json:"class_name"`
		ClassID   uuid.UUID `json:"class_id"`
	}

	var results []StudentWithClass

	query := h.db.Table("students").Select("students.*, classes.level as class_name, classes.id as class_id").
		Joins("LEFT JOIN enrollments ON students.id = enrollments.student_id").
		Joins("LEFT JOIN classes ON enrollments.class_id = classes.id").
		Where("students.deleted_at IS NULL")

	// Filter by school for non-system admins
	if schoolID != "" {
		query = query.Where("students.school_id = ?", schoolID)
	}

	if classID != "" {
		query = query.Where("enrollments.class_id = ?", classID)
	} else if classLevel != "" {
		// Filter by class level, term, and year
		subQuery := h.db.Table("classes").Select("id").Where("level = ?", classLevel)
		if term != "" {
			subQuery = subQuery.Where("term = ?", term)
		}
		if year != "" {
			subQuery = subQuery.Where("year = ?", year)
		}
		if schoolID != "" {
			subQuery = subQuery.Where("school_id = ?", schoolID)
		}
		query = query.Where("enrollments.class_id IN (?)", subQuery)
	}

	if err := query.Scan(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, results)
}

func (h *StudentHandler) Create(c *gin.Context) {
	var req struct {
		FirstName  string `json:"first_name" binding:"required"`
		LastName   string `json:"last_name" binding:"required"`
		Gender     string `json:"gender"`
		ClassLevel string `json:"class_level" binding:"required"`
		Term       string `json:"term" binding:"required"`
		Year       int    `json:"year" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Auto-assign student to the same school as the user
	userRole := c.GetString("user_role")
	var school models.School
	if userRole != "system_admin" {
		tenantSchoolID := c.GetString("tenant_school_id")
		if tenantSchoolID == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "No school assigned to user"})
			return
		}
		if err := h.db.First(&school, "id = ?", tenantSchoolID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "School not found"})
			return
		}
	} else {
		// System admin must specify school through class selection
		var class models.Class
		if err := h.db.Preload("School").Where("level = ? AND term = ? AND year = ?", req.ClassLevel, req.Term, req.Year).First(&class).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Class not found"})
			return
		}
		school = *class.School
	}

	// Find the specific class for this school, level, term, and year
	var class models.Class
	if err := h.db.Where("school_id = ? AND level = ? AND term = ? AND year = ?", school.ID, req.ClassLevel, req.Term, req.Year).First(&class).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Class not found for the specified level, term, and year"})
		return
	}

	// Count students in this class
	var count int64
	h.db.Table("students").Joins("JOIN enrollments ON students.id = enrollments.student_id").
		Where("enrollments.class_id = ?", class.ID).Count(&count)

	// Generate admission number
	schoolInitial := string(school.Name[0])
	var schoolType string
	switch school.Type {
	case "Nursery":
		schoolType = "NS"
	case "Primary":
		schoolType = "PS"
	default:
		schoolType = "SS"
	}
	sequence := int(count) + 1
	admissionNo := fmt.Sprintf("%s%s/%s/%d/%03d", schoolInitial, schoolType, class.Level, req.Year, sequence)

	student := models.Student{
		SchoolID:    school.ID,
		AdmissionNo: admissionNo,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Gender:      req.Gender,
	}

	if err := h.db.Create(&student).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	enrollment := models.Enrollment{
		StudentID:  student.ID,
		ClassID:    class.ID,
		Year:       req.Year,
		Term:       req.Term,
		Status:     "active",
		EnrolledOn: time.Now(),
	}
	h.db.Create(&enrollment)

	c.JSON(http.StatusCreated, student)
}

func (h *StudentHandler) Get(c *gin.Context) {
	id := c.Param("id")
	schoolID := c.GetString("tenant_school_id")

	var student models.Student
	query := h.db.Where("id = ?", id)

	// Filter by school for non-system admins
	if schoolID != "" {
		query = query.Where("school_id = ?", schoolID)
	}

	if err := query.First(&student).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Student not found"})
		return
	}
	c.JSON(http.StatusOK, student)
}

func (h *StudentHandler) Update(c *gin.Context) {
	id := c.Param("id")
	schoolID := c.GetString("tenant_school_id")

	var student models.Student
	query := h.db.Where("id = ?", id)

	// Filter by school for non-system admins
	if schoolID != "" {
		query = query.Where("school_id = ?", schoolID)
	}

	if err := query.First(&student).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Student not found"})
		return
	}

	if err := c.ShouldBindJSON(&student); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.Save(&student).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, student)
}

func (h *StudentHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	schoolID := c.GetString("tenant_school_id")

	query := h.db.Where("id = ?", id)

	// Filter by school for non-system admins
	if schoolID != "" {
		query = query.Where("school_id = ?", schoolID)
	}

	if err := query.Delete(&models.Student{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Student deleted"})
}
