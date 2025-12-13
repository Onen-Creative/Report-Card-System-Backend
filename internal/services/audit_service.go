package services

import (
	"github.com/google/uuid"
	"github.com/school-system/backend/internal/models"
	"gorm.io/gorm"
)

type AuditService struct {
	db *gorm.DB
}

func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{db: db}
}

func (s *AuditService) Log(userID uuid.UUID, action, resourceType string, resourceID uuid.UUID, before, after models.JSONB, ip string) error {
	log := &models.AuditLog{
		ActorUserID:  userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Before:       before,
		After:        after,
		IP:           ip,
	}
	return s.db.Create(log).Error
}
