package database

import (
	"fmt"
	"log"

	"github.com/school-system/backend/internal/config"
	"github.com/school-system/backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.Config) (*gorm.DB, error) {
	var logLevel logger.LogLevel
	if cfg.Server.Env == "development" {
		logLevel = logger.Info
	} else {
		logLevel = logger.Silent
	}

	// Debug: Log connection attempt (without password)
	log.Printf("Attempting database connection with DSN: %s", maskPassword(cfg.Database.DSN))

	db, err := gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Database connection successful")
	return db, nil
}

func maskPassword(dsn string) string {
	// Simple password masking for logging
	if len(dsn) > 20 {
		return dsn[:20] + "...***..."
	}
	return "***"
}

func Migrate(db *gorm.DB) error {
	log.Println("Running migrations...")
	
	return db.AutoMigrate(
		&models.School{},
		&models.User{},
		&models.Class{},
		&models.Student{},
		&models.Enrollment{},
		&models.Subject{},
		&models.StandardSubject{},
		&models.Assessment{},
		&models.Mark{},
		&models.SubjectResult{},
		&models.ReportCard{},
		&models.AuditLog{},
		&models.Job{},
		&models.GradingRule{},
		&models.RefreshToken{},
	)
}
