package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/school-system/backend/internal/models"
	"gorm.io/gorm"
)

type SchoolSetupService struct {
	db                    *gorm.DB
	userAssignmentService *UserAssignmentService
}

func NewSchoolSetupService(db *gorm.DB) *SchoolSetupService {
	return &SchoolSetupService{
		db:                    db,
		userAssignmentService: NewUserAssignmentService(db),
	}
}

// SetupSchool configures a school with classes, subjects, users, and default settings
func (s *SchoolSetupService) SetupSchool(school *models.School, levels []string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Create classes for each level
		if err := s.createClasses(tx, school.ID, levels); err != nil {
			return fmt.Errorf("failed to create classes: %w", err)
		}

		// 2. Create subjects for each level
		if err := s.createSubjects(tx, school.ID, levels); err != nil {
			return fmt.Errorf("failed to create subjects: %w", err)
		}

		// 3. Create grading rules
		if err := s.createGradingRules(tx, school.ID, levels); err != nil {
			return fmt.Errorf("failed to create grading rules: %w", err)
		}

		// 4. Create default school admin
		if _, err := s.userAssignmentService.CreateSchoolAdmin(school.ID, school.Name); err != nil {
			return fmt.Errorf("failed to create school admin: %w", err)
		}

		return nil
	})
}

func (s *SchoolSetupService) createClasses(tx *gorm.DB, schoolID uuid.UUID, levels []string) error {
	currentYear := time.Now().Year()
	terms := []string{"Term1", "Term2", "Term3"}

	for _, level := range levels {
		for _, term := range terms {
			// Check if class already exists
			var existing models.Class
			err := tx.Where("school_id = ? AND level = ? AND year = ? AND term = ?", schoolID, level, currentYear, term).First(&existing).Error
			if err == nil {
				// Class already exists, skip
				continue
			}
			if err != gorm.ErrRecordNotFound {
				return err
			}

			class := models.Class{
				SchoolID: schoolID,
				Name:     fmt.Sprintf("%s %s %d", level, term, currentYear),
				Level:    level,
				Year:     currentYear,
				Term:     term,
			}
			if err := tx.Create(&class).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *SchoolSetupService) createSubjects(tx *gorm.DB, schoolID uuid.UUID, levels []string) error {
	// Use standard subject service to create subjects from curriculum
	standardSubjectService := NewStandardSubjectService(tx)
	return standardSubjectService.CreateSchoolSubjectsFromStandard(schoolID, levels)
}

func (s *SchoolSetupService) createSubjectsOld(tx *gorm.DB, schoolID uuid.UUID, levels []string) error {
	// Check if subjects already exist for this school
	var count int64
	tx.Model(&models.Subject{}).Where("school_id = ?", schoolID).Count(&count)
	if count > 0 {
		return nil // Subjects already exist
	}
	subjectsByLevel := map[string][]SubjectConfig{
		"ECCE": {
			{"English", "ENG", true, 1},
			{"Mathematics", "MATH", true, 1},
			{"Science", "SCI", true, 1},
			{"Social Studies", "SST", true, 1},
			{"Art & Craft", "ART", false, 1},
			{"Music", "MUS", false, 1},
			{"Physical Education", "PE", false, 1},
		},
		"P1": {
			{"English", "ENG", true, 1},
			{"Mathematics", "MATH", true, 1},
			{"Science", "SCI", true, 1},
			{"Social Studies", "SST", true, 1},
			{"Religious Education", "RE", true, 1},
		},
		"P2": {
			{"English", "ENG", true, 1},
			{"Mathematics", "MATH", true, 1},
			{"Science", "SCI", true, 1},
			{"Social Studies", "SST", true, 1},
			{"Religious Education", "RE", true, 1},
		},
		"P3": {
			{"English", "ENG", true, 1},
			{"Mathematics", "MATH", true, 1},
			{"Science", "SCI", true, 1},
			{"Social Studies", "SST", true, 1},
			{"Religious Education", "RE", true, 1},
		},
		"P4": {
			{"English", "ENG", true, 1},
			{"Mathematics", "MATH", true, 1},
			{"Science", "SCI", true, 1},
			{"Social Studies", "SST", true, 1},
			{"Religious Education", "RE", true, 1},
		},
		"P5": {
			{"English", "ENG", true, 1},
			{"Mathematics", "MATH", true, 1},
			{"Science", "SCI", true, 1},
			{"Social Studies", "SST", true, 1},
			{"Religious Education", "RE", true, 1},
		},
		"P6": {
			{"English", "ENG", true, 1},
			{"Mathematics", "MATH", true, 1},
			{"Science", "SCI", true, 1},
			{"Social Studies", "SST", true, 1},
			{"Religious Education", "RE", true, 1},
		},
		"P7": {
			{"English", "ENG", true, 1},
			{"Mathematics", "MATH", true, 1},
			{"Science", "SCI", true, 1},
			{"Social Studies", "SST", true, 1},
			{"Religious Education", "RE", true, 1},
		},
		"S1": {
			{"English Language", "ENG", true, 1},
			{"Mathematics", "MATH", true, 1},
			{"Biology", "BIO", true, 1},
			{"Chemistry", "CHEM", true, 1},
			{"Physics", "PHY", true, 1},
			{"History", "HIST", true, 1},
			{"Geography", "GEO", true, 1},
			{"Religious Education", "RE", true, 1},
			{"Computer Studies", "COMP", false, 1},
		},
		"S2": {
			{"English Language", "ENG", true, 1},
			{"Mathematics", "MATH", true, 1},
			{"Biology", "BIO", true, 1},
			{"Chemistry", "CHEM", true, 1},
			{"Physics", "PHY", true, 1},
			{"History", "HIST", true, 1},
			{"Geography", "GEO", true, 1},
			{"Religious Education", "RE", true, 1},
			{"Computer Studies", "COMP", false, 1},
		},
		"S3": {
			{"English Language", "ENG", true, 1},
			{"Mathematics", "MATH", true, 1},
			{"Biology", "BIO", false, 1},
			{"Chemistry", "CHEM", false, 1},
			{"Physics", "PHY", false, 1},
			{"History", "HIST", false, 1},
			{"Geography", "GEO", false, 1},
			{"Economics", "ECON", false, 1},
			{"Literature", "LIT", false, 1},
			{"Computer Studies", "COMP", false, 1},
		},
		"S4": {
			{"English Language", "ENG", true, 1},
			{"Mathematics", "MATH", true, 1},
			{"Biology", "BIO", false, 1},
			{"Chemistry", "CHEM", false, 1},
			{"Physics", "PHY", false, 1},
			{"History", "HIST", false, 1},
			{"Geography", "GEO", false, 1},
			{"Economics", "ECON", false, 1},
			{"Literature", "LIT", false, 1},
			{"Computer Studies", "COMP", false, 1},
		},
		"S5": {
			{"General Paper", "GP", true, 1},
			{"Mathematics", "MATH", false, 2},
			{"Physics", "PHY", false, 2},
			{"Chemistry", "CHEM", false, 2},
			{"Biology", "BIO", false, 2},
			{"History", "HIST", false, 1},
			{"Geography", "GEO", false, 1},
			{"Economics", "ECON", false, 1},
			{"Literature", "LIT", false, 1},
			{"Computer Science", "COMP", false, 2},
		},
		"S6": {
			{"General Paper", "GP", true, 1},
			{"Mathematics", "MATH", false, 2},
			{"Physics", "PHY", false, 2},
			{"Chemistry", "CHEM", false, 2},
			{"Biology", "BIO", false, 2},
			{"History", "HIST", false, 1},
			{"Geography", "GEO", false, 1},
			{"Economics", "ECON", false, 1},
			{"Literature", "LIT", false, 1},
			{"Computer Science", "COMP", false, 2},
		},
	}

	for _, level := range levels {
		if subjects, exists := subjectsByLevel[level]; exists {
			for _, subjectConfig := range subjects {
				subject := models.Subject{
					SchoolID:     schoolID,
					Name:         subjectConfig.Name,
					Code:         subjectConfig.Code,
					Level:        level,
					IsCompulsory: subjectConfig.IsCompulsory,
					Papers:       subjectConfig.Papers,
				}
				if err := tx.Create(&subject).Error; err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s *SchoolSetupService) createGradingRules(tx *gorm.DB, schoolID uuid.UUID, levels []string) error {
	for _, level := range levels {
		// Check if grading rule already exists
		var existing models.GradingRule
		err := tx.Where("school_id = ? AND level = ?", schoolID, level).First(&existing).Error
		if err == nil {
			// Rule already exists, skip
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}
		var ruleVersion string
		var rules models.JSONB

		if isPrimaryLevel(level) {
			ruleVersion = "NCDC_PRIMARY_2023"
			rules = models.JSONB{
				"type": "primary",
				"grades": map[string]interface{}{
					"1": map[string]interface{}{"min": 80, "max": 100, "description": "Excellent"},
					"2": map[string]interface{}{"min": 70, "max": 79, "description": "Very Good"},
					"3": map[string]interface{}{"min": 60, "max": 69, "description": "Good"},
					"4": map[string]interface{}{"min": 50, "max": 59, "description": "Satisfactory"},
					"5": map[string]interface{}{"min": 0, "max": 49, "description": "Needs Improvement"},
				},
			}
		} else if isSecondaryLevel(level) {
			if level == "S5" || level == "S6" {
				ruleVersion = "UNEB_ADVANCED_2023"
				rules = models.JSONB{
					"type": "advanced",
					"grades": map[string]interface{}{
						"A": map[string]interface{}{"min": 80, "max": 100, "points": 6},
						"B": map[string]interface{}{"min": 70, "max": 79, "points": 5},
						"C": map[string]interface{}{"min": 60, "max": 69, "points": 4},
						"D": map[string]interface{}{"min": 50, "max": 59, "points": 3},
						"E": map[string]interface{}{"min": 40, "max": 49, "points": 2},
						"O": map[string]interface{}{"min": 35, "max": 39, "points": 1},
						"F": map[string]interface{}{"min": 0, "max": 34, "points": 0},
					},
				}
			} else {
				ruleVersion = "UNEB_ORDINARY_2023"
				rules = models.JSONB{
					"type": "ordinary",
					"grades": map[string]interface{}{
						"D1": map[string]interface{}{"min": 80, "max": 100, "points": 1},
						"D2": map[string]interface{}{"min": 70, "max": 79, "points": 2},
						"C3": map[string]interface{}{"min": 65, "max": 69, "points": 3},
						"C4": map[string]interface{}{"min": 60, "max": 64, "points": 4},
						"C5": map[string]interface{}{"min": 55, "max": 59, "points": 5},
						"C6": map[string]interface{}{"min": 50, "max": 54, "points": 6},
						"P7": map[string]interface{}{"min": 45, "max": 49, "points": 7},
						"P8": map[string]interface{}{"min": 40, "max": 44, "points": 8},
						"F9": map[string]interface{}{"min": 0, "max": 39, "points": 9},
					},
				}
			}
		} else {
			// ECCE level
			ruleVersion = "NCDC_ECCE_2023"
			rules = models.JSONB{
				"type": "ecce",
				"grades": map[string]interface{}{
					"E": map[string]interface{}{"description": "Excellent"},
					"VG": map[string]interface{}{"description": "Very Good"},
					"G": map[string]interface{}{"description": "Good"},
					"S": map[string]interface{}{"description": "Satisfactory"},
					"NI": map[string]interface{}{"description": "Needs Improvement"},
				},
			}
		}

		gradingRule := models.GradingRule{
			SchoolID:    schoolID,
			Level:       level,
			RuleVersion: ruleVersion,
			Rules:       rules,
		}
		if err := tx.Create(&gradingRule).Error; err != nil {
			return err
		}
	}
	return nil
}

type SubjectConfig struct {
	Name         string
	Code         string
	IsCompulsory bool
	Papers       int
}

func isPrimaryLevel(level string) bool {
	primaryLevels := []string{"P1", "P2", "P3", "P4", "P5", "P6", "P7"}
	for _, p := range primaryLevels {
		if p == level {
			return true
		}
	}
	return false
}

func isSecondaryLevel(level string) bool {
	secondaryLevels := []string{"S1", "S2", "S3", "S4", "S5", "S6"}
	for _, s := range secondaryLevels {
		if s == level {
			return true
		}
	}
	return false
}

// SetupNewLevels adds classes and subjects for new levels added to existing school
func (s *SchoolSetupService) SetupNewLevels(schoolID uuid.UUID, newLevels []string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Only create classes and subjects for new levels
		if err := s.createClasses(tx, schoolID, newLevels); err != nil {
			return fmt.Errorf("failed to create classes for new levels: %w", err)
		}

		if err := s.createSubjects(tx, schoolID, newLevels); err != nil {
			return fmt.Errorf("failed to create subjects for new levels: %w", err)
		}

		if err := s.createGradingRules(tx, schoolID, newLevels); err != nil {
			return fmt.Errorf("failed to create grading rules for new levels: %w", err)
		}

		return nil
	})
}