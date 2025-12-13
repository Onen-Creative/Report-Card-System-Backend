package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/school-system/backend/internal/models"
	"gorm.io/gorm"
)

type ResultHandler struct {
	db *gorm.DB
}

func NewResultHandler(db *gorm.DB) *ResultHandler {
	return &ResultHandler{db: db}
}

func (h *ResultHandler) GetByStudent(c *gin.Context) {
	studentID := c.Param("id")
	term := c.Query("term")
	year := c.Query("year")
	schoolID := c.GetString("tenant_school_id")
	
	// Verify student belongs to the same school
	var student models.Student
	if err := h.db.Where("id = ? AND school_id = ?", studentID, schoolID).First(&student).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Student not found or access denied"})
		return
	}
	
	type ResultWithSubject struct {
		models.SubjectResult
		SubjectName string `json:"subject_name"`
		SubjectCode string `json:"subject_code"`
	}
	
	var results []ResultWithSubject
	// Join with standard_subjects instead of subjects to ensure consistency
	query := h.db.Table("subject_results").
		Select("subject_results.*, standard_subjects.name as subject_name, standard_subjects.code as subject_code").
		Joins("LEFT JOIN standard_subjects ON subject_results.subject_id = standard_subjects.id").
		Where("subject_results.student_id = ? AND subject_results.school_id = ?", studentID, schoolID)
	
	if term != "" {
		query = query.Where("subject_results.term = ?", term)
	}
	if year != "" {
		query = query.Where("subject_results.year = ?", year)
	}
	
	if err := query.Scan(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, results)
}

func (h *ResultHandler) CreateOrUpdate(c *gin.Context) {
	userRole := c.GetString("user_role")
	
	var req struct {
		StudentID   string                 `json:"student_id" binding:"required"`
		SubjectID   string                 `json:"subject_id" binding:"required"`
		ClassID     string                 `json:"class_id"`
		Term        string                 `json:"term" binding:"required"`
		Year        int                    `json:"year" binding:"required"`
		FinalGrade  string                 `json:"final_grade"`
		RawMarks    models.JSONB           `json:"raw_marks"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	studentID, err := uuid.Parse(req.StudentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid student ID"})
		return
	}
	
	subjectID, err := uuid.Parse(req.SubjectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subject ID"})
		return
	}
	
	schoolID := c.GetString("tenant_school_id")
	if schoolID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "School ID required"})
		return
	}
	
	// Verify student belongs to the same school
	var student models.Student
	if err := h.db.Where("id = ? AND school_id = ?", studentID, schoolID).First(&student).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Student not found or access denied"})
		return
	}
	
	// Get student's class from enrollment
	var enrollment models.Enrollment
	if err := h.db.Where("student_id = ?", studentID).Order("created_at DESC").First(&enrollment).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Student not enrolled in any class"})
		return
	}
	classID := enrollment.ClassID
	
	// Verify that the subject is a valid standard subject
	var standardSubject models.StandardSubject
	if err := h.db.First(&standardSubject, "id = ?", subjectID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subject - only standard curriculum subjects are allowed"})
		return
	}
	
	// Check if result already exists
	var result models.SubjectResult
	err = h.db.Where("student_id = ? AND subject_id = ? AND term = ? AND year = ?",
		studentID, subjectID, req.Term, req.Year).First(&result).Error
	
	// Teachers can only create new results, not edit existing ones
	if userRole == "teacher" && err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusForbidden, gin.H{"error": "Teachers cannot edit existing marks"})
		return
	}
	
	// Calculate grade from total marks
	total := 0.0
	if req.RawMarks != nil {
		if t, ok := req.RawMarks["total"].(float64); ok {
			total = t
		}
	}
	grade := ""
	if total >= 80 {
		grade = "A"
	} else if total >= 65 {
		grade = "B"
	} else if total >= 50 {
		grade = "C"
	} else if total >= 35 {
		grade = "D"
	} else {
		grade = "E"
	}
	
	if err == gorm.ErrRecordNotFound {
		result = models.SubjectResult{
			StudentID:  studentID,
			SubjectID:  subjectID,
			ClassID:    classID,
			Term:       req.Term,
			Year:       req.Year,
			SchoolID:   uuid.MustParse(schoolID),
			FinalGrade: grade,
			RawMarks:   req.RawMarks,
		}
		if err := h.db.Create(&result).Error; err != nil {
			log.Printf("Error creating result: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	} else {
		// Only school admins can update existing results
		result.FinalGrade = grade
		result.RawMarks = req.RawMarks
		if err := h.db.Save(&result).Error; err != nil {
			log.Printf("Error saving result: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	
	c.JSON(http.StatusOK, result)
}

func (h *ResultHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&models.SubjectResult{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Result deleted"})
}
