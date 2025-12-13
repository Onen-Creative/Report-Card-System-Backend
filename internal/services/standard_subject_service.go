package services

import (
	"github.com/google/uuid"
	"github.com/school-system/backend/internal/models"
	"gorm.io/gorm"
)

type StandardSubjectService struct {
	db *gorm.DB
}

func NewStandardSubjectService(db *gorm.DB) *StandardSubjectService {
	return &StandardSubjectService{db: db}
}

// GetSubjectsForLevel returns standard subjects for a specific education level
func (s *StandardSubjectService) GetSubjectsForLevel(level string) ([]models.StandardSubject, error) {
	var subjects []models.StandardSubject
	err := s.db.Where("level = ?", level).Find(&subjects).Error
	return subjects, err
}

// CreateSchoolSubjectsFromStandard creates school-specific subjects based on standard subjects
func (s *StandardSubjectService) CreateSchoolSubjectsFromStandard(schoolID uuid.UUID, levels []string) error {
	for _, level := range levels {
		standardSubjects, err := s.GetSubjectsForLevel(level)
		if err != nil {
			return err
		}

		for _, std := range standardSubjects {
			subject := models.Subject{
				SchoolID:     schoolID,
				Name:         std.Name,
				Code:         std.Code,
				Level:        std.Level,
				IsCompulsory: std.IsCompulsory,
				Papers:       std.Papers,
			}
			
			// Check if subject already exists for this school and level
			var existing models.Subject
			err := s.db.Where("school_id = ? AND name = ? AND level = ?", schoolID, std.Name, std.Level).First(&existing).Error
			if err == gorm.ErrRecordNotFound {
				if err := s.db.Create(&subject).Error; err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// GetAllLevels returns all available education levels
func (s *StandardSubjectService) GetAllLevels() ([]string, error) {
	var levels []string
	err := s.db.Model(&models.StandardSubject{}).Distinct("level").Pluck("level", &levels).Error
	return levels, err
}