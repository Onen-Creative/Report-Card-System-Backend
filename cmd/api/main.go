package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/school-system/backend/internal/config"
	"github.com/school-system/backend/internal/database"
	"github.com/school-system/backend/internal/handlers"
	"github.com/school-system/backend/internal/middleware"
	"github.com/school-system/backend/internal/models"
	"github.com/school-system/backend/internal/services"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// @title School Management System API
// @version 1.0
// @description Production-ready School Management & Report Card System for Ugandan schools
// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	if len(os.Args) > 1 {
		handleCommand(os.Args[1])
		return
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	if cfg.Server.Env == "development" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())

	// CORS
	r.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allowedOrigins := []string{"http://localhost:5173", "http://localhost:5174", "https://your-app.netlify.app"}
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				break
			}
		}
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Static files
	r.Static("/logos", "./public/logos")

	// Health check
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Metrics
	if cfg.Monitoring.PrometheusEnabled {
		r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}

	// Swagger
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Services
	authService := services.NewAuthService(db, cfg)

	// Handlers
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(db, authService)
	schoolHandler := handlers.NewSchoolHandler(db)
	classHandler := handlers.NewClassHandler(db)
	studentHandler := handlers.NewStudentHandler(db)
	subjectHandler := handlers.NewSubjectHandler(db)
	resultHandler := handlers.NewResultHandler(db)
	uploadHandler := handlers.NewUploadHandler(db)
	auditHandler := handlers.NewAuditHandler(db)

	// Routes
	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.Refresh)
			auth.POST("/logout", authHandler.Logout)
		}

		// Protected routes
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(authService))
		protected.Use(middleware.TenantMiddleware())
		{
			// System Admin only routes
			sysAdmin := protected.Group("")
			sysAdmin.Use(middleware.RequireSystemAdmin())
			{
				sysAdmin.GET("/users", userHandler.List)
				sysAdmin.POST("/users", userHandler.Create)
				sysAdmin.GET("/users/:id", userHandler.Get)
				sysAdmin.PUT("/users/:id", userHandler.Update)
				sysAdmin.DELETE("/users/:id", userHandler.Delete)

				sysAdmin.POST("/schools", schoolHandler.Create)
				sysAdmin.PUT("/schools/:id", schoolHandler.Update)
				sysAdmin.DELETE("/schools/:id", schoolHandler.Delete)
				sysAdmin.GET("/stats", schoolHandler.GetStats)

				// Standard subject management
				sysAdmin.GET("/standard-subjects", subjectHandler.ListStandardSubjects)
				sysAdmin.POST("/standard-subjects", subjectHandler.CreateStandardSubject)
				sysAdmin.PUT("/standard-subjects/:id", subjectHandler.UpdateStandardSubject)
				sysAdmin.DELETE("/standard-subjects/:id", subjectHandler.DeleteStandardSubject)

				// Audit logs
				sysAdmin.GET("/audit/recent", auditHandler.GetRecentActivity)
			}

			// School Admin routes
			schoolAdmin := protected.Group("")
			schoolAdmin.Use(middleware.RequireSchoolAdmin())
			{
				schoolAdmin.POST("/students", studentHandler.Create)
				schoolAdmin.PUT("/students/:id", studentHandler.Update)
				schoolAdmin.DELETE("/students/:id", studentHandler.Delete)
				// Note: Subject creation/modification removed - only standard subjects allowed
				schoolAdmin.DELETE("/results/:id", resultHandler.Delete)
			}

			// Teacher routes (all authenticated users)
			protected.GET("/schools", schoolHandler.List)
			protected.GET("/schools/:id", schoolHandler.Get)
			protected.GET("/classes", classHandler.List)
			protected.GET("/classes/levels", classHandler.GetLevels)
			protected.GET("/classes/:id", classHandler.Get)
			protected.GET("/classes/:id/students", classHandler.GetStudents)
			protected.GET("/students", studentHandler.List)
			protected.GET("/students/:id", studentHandler.Get)
			protected.GET("/students/:id/results", resultHandler.GetByStudent)
			protected.GET("/subjects", subjectHandler.List)
			protected.GET("/subjects/levels", subjectHandler.GetLevels)
			protected.POST("/results", resultHandler.CreateOrUpdate)
			protected.POST("/upload/logo", uploadHandler.UploadLogo)
		}
	}

	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func handleCommand(cmd string) {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	switch cmd {
	case "migrate":
		if err := database.Migrate(db); err != nil {
			log.Fatal("Migration failed:", err)
		}
		log.Println("Migration completed successfully")

	case "seed-admin":
		seedAdmin(db, cfg)

	case "seed-subjects":
		seedSubjects(db)

	case "seed-sample":
		log.Println("Seeding sample data - TODO")

	case "cleanup-duplicates":
		cleanupDuplicates(db)

	case "standardize-classes":
		standardizeClassNames(db)

	case "standardize-subjects":
		standardizeSubjects(db)

	case "seed-standard-subjects":
		seedStandardSubjects(db)

	case "migrate-to-standard-subjects":
		migrateToStandardSubjects(db)

	case "fix-foreign-keys":
		fixForeignKeys(db)

	case "cleanup-orphaned-data":
		cleanupOrphanedData(db)

	default:
		log.Printf("Unknown command: %s", cmd)
	}
}

func seedAdmin(db *gorm.DB, cfg *config.Config) {
	authService := services.NewAuthService(db, cfg)

	var count int64
	db.Model(&models.User{}).Where("role = ?", "system_admin").Count(&count)
	if count > 0 {
		log.Println("System admin already exists")
		return
	}

	// Create system admin without school assignment
	sysAdmin := &models.User{
		SchoolID: nil,
		Email:    "sysadmin@school.ug",
		FullName: "System Administrator",
		Role:     "system_admin",
		IsActive: true,
	}

	if err := authService.CreateUser(sysAdmin, "Admin@123"); err != nil {
		log.Fatal("Failed to create system admin:", err)
	}

	log.Println("System Admin: sysadmin@school.ug / Admin@123")

	// Create default school
	var school models.School
	if err := db.First(&school).Error; err != nil {
		school = models.School{
			Name:         "Nabumali Secondary School",
			Type:         "Secondary",
			Address:      "Nabumali, Mbale District",
			Country:      "Uganda",
			Region:       "Eastern",
			ContactEmail: "nabumalisecondaryschool@gmail.com",
			Phone:        "+256-782-390-592",
			LogoURL:      "https://picsum.photos/150",
			Motto:        "Excellence Through Discipline",
			Config:       models.JSONB{"levels": []string{"S1", "S2", "S3", "S4", "S5", "S6"}},
		}
		db.Create(&school)
	}

	// Create school admin assigned to the school
	schoolAdmin := &models.User{
		SchoolID: &school.ID,
		Email:    "schooladmin@school.ug",
		FullName: "School Administrator",
		Role:     "school_admin",
		IsActive: true,
	}

	if err := authService.CreateUser(schoolAdmin, "Admin@123"); err != nil {
		log.Fatal("Failed to create school admin:", err)
	}

	log.Println("School Admin: schooladmin@school.ug / Admin@123")

	// Create teacher assigned to the school
	teacher := &models.User{
		SchoolID: &school.ID,
		Email:    "teacher@school.ug",
		FullName: "Teacher",
		Role:     "teacher",
		IsActive: true,
	}

	if err := authService.CreateUser(teacher, "Teacher@123"); err != nil {
		log.Fatal("Failed to create teacher:", err)
	}

	log.Println("Teacher: teacher@school.ug / Teacher@123")
}

func seedSubjects(db *gorm.DB) {
	var school models.School
	if err := db.Where("name = ?", "Nabumali Secondary School").First(&school).Error; err != nil {
		log.Fatal("School not found")
	}

	db.Where("school_id = ?", school.ID).Delete(&models.Subject{})

	subjects := []models.Subject{}

	// Nursery subjects
	nurserySubjects := []models.Subject{
		{SchoolID: school.ID, Name: "Language & Early Literacy", Code: "LIT", Level: "Nursery", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "Early Numeracy", Code: "NUM", Level: "Nursery", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "Social & Emotional Development", Code: "SED", Level: "Nursery", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "Creative Arts", Code: "ART", Level: "Nursery", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "Physical & Motor Skills", Code: "PMS", Level: "Nursery", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "Health, Hygiene & Nutrition", Code: "HHN", Level: "Nursery", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "Play & Environmental Awareness", Code: "PEA", Level: "Nursery", IsCompulsory: true, Papers: 1},
	}

	// P1-P3 subjects (thematic)
	p13Subjects := []models.Subject{
		{SchoolID: school.ID, Name: "Literacy", Code: "LIT", Level: "P1", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "Numeracy", Code: "NUM", Level: "P1", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "Life Skills", Code: "LS", Level: "P1", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "Creative Arts", Code: "ART", Level: "P1", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "Environment", Code: "ENV", Level: "P1", IsCompulsory: true, Papers: 1},
	}

	// P4-P7 subjects
	p47Subjects := []models.Subject{
		{SchoolID: school.ID, Name: "English Language", Code: "ENG", Level: "P4", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "Mathematics", Code: "MATH", Level: "P4", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "Integrated Science", Code: "SCI", Level: "P4", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "Social Studies", Code: "SST", Level: "P4", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "Local Language", Code: "LL", Level: "P4", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "Creative Arts", Code: "ART", Level: "P4", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "Physical Education", Code: "PE", Level: "P4", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "Agriculture", Code: "AGR", Level: "P4", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "ICT", Code: "ICT", Level: "P4", IsCompulsory: false, Papers: 1},
	}

	// S1-S4 compulsory subjects
	s14Compulsory := []models.Subject{
		{SchoolID: school.ID, Name: "English Language", Code: "ENG", Level: "S1", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "Mathematics", Code: "MATH", Level: "S1", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "Physics", Code: "PHY", Level: "S1", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "Chemistry", Code: "CHEM", Level: "S1", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "Biology", Code: "BIO", Level: "S1", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "Geography", Code: "GEO", Level: "S1", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "History & Political Education", Code: "HIST", Level: "S1", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "Religious Education", Code: "RE", Level: "S1", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "Entrepreneurship Education", Code: "ENT", Level: "S1", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "Kiswahili", Code: "KIS", Level: "S1", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "Physical Education", Code: "PE", Level: "S1", IsCompulsory: true, Papers: 1},
	}

	// S1-S4 electives
	s14Electives := []models.Subject{
		{SchoolID: school.ID, Name: "Literature in English", Code: "LIT", Level: "S1", IsCompulsory: false, Papers: 1},
		{SchoolID: school.ID, Name: "ICT / Computer Studies", Code: "ICT", Level: "S1", IsCompulsory: false, Papers: 1},
		{SchoolID: school.ID, Name: "Agriculture", Code: "AGR", Level: "S1", IsCompulsory: false, Papers: 1},
		{SchoolID: school.ID, Name: "Performing Arts", Code: "PA", Level: "S1", IsCompulsory: false, Papers: 1},
		{SchoolID: school.ID, Name: "Art & Design", Code: "AD", Level: "S1", IsCompulsory: false, Papers: 1},
		{SchoolID: school.ID, Name: "Technology & Design", Code: "TD", Level: "S1", IsCompulsory: false, Papers: 1},
		{SchoolID: school.ID, Name: "Nutrition & Food Tech", Code: "NFT", Level: "S1", IsCompulsory: false, Papers: 1},
		{SchoolID: school.ID, Name: "Local Language", Code: "LL", Level: "S1", IsCompulsory: false, Papers: 1},
	}

	// S5-S6 Principal subjects
	s56Principal := []models.Subject{
		{SchoolID: school.ID, Name: "Mathematics", Code: "MATH", Level: "S5", IsCompulsory: false, Papers: 2},
		{SchoolID: school.ID, Name: "Physics", Code: "PHY", Level: "S5", IsCompulsory: false, Papers: 3},
		{SchoolID: school.ID, Name: "Chemistry", Code: "CHEM", Level: "S5", IsCompulsory: false, Papers: 3},
		{SchoolID: school.ID, Name: "Biology", Code: "BIO", Level: "S5", IsCompulsory: false, Papers: 3},
		{SchoolID: school.ID, Name: "Geography", Code: "GEO", Level: "S5", IsCompulsory: false, Papers: 3},
		{SchoolID: school.ID, Name: "History & Political Education", Code: "HIST", Level: "S5", IsCompulsory: false, Papers: 3},
		{SchoolID: school.ID, Name: "Religious Education", Code: "RE", Level: "S5", IsCompulsory: false, Papers: 2},
		{SchoolID: school.ID, Name: "Entrepreneurship Education", Code: "ENT", Level: "S5", IsCompulsory: false, Papers: 2},
		{SchoolID: school.ID, Name: "Agriculture", Code: "AGR", Level: "S5", IsCompulsory: false, Papers: 3},
		{SchoolID: school.ID, Name: "Economics", Code: "ECON", Level: "S5", IsCompulsory: false, Papers: 2},
		{SchoolID: school.ID, Name: "Luganda", Code: "LUG", Level: "S5", IsCompulsory: false, Papers: 2},
		{SchoolID: school.ID, Name: "Art and Design", Code: "AD", Level: "S5", IsCompulsory: false, Papers: 2},
		{SchoolID: school.ID, Name: "Literature", Code: "LIT", Level: "S5", IsCompulsory: false, Papers: 2},
	}

	// S5-S6 Subsidiary subjects
	s56Subsidiary := []models.Subject{
		{SchoolID: school.ID, Name: "General Paper", Code: "GP", Level: "S5", IsCompulsory: true, Papers: 1},
		{SchoolID: school.ID, Name: "ICT", Code: "ICT", Level: "S5", IsCompulsory: false, Papers: 1},
		{SchoolID: school.ID, Name: "Subsidiary Mathematics", Code: "SUBMATH", Level: "S5", IsCompulsory: false, Papers: 1},
	}

	// Add all subjects
	subjects = append(subjects, nurserySubjects...)
	for _, level := range []string{"P1", "P2", "P3"} {
		for _, s := range p13Subjects {
			s.Level = level
			subjects = append(subjects, s)
		}
	}
	for _, level := range []string{"P4", "P5", "P6", "P7"} {
		for _, s := range p47Subjects {
			s.Level = level
			subjects = append(subjects, s)
		}
	}
	for _, level := range []string{"S1", "S2", "S3", "S4"} {
		for _, s := range s14Compulsory {
			s.Level = level
			subjects = append(subjects, s)
		}
		for _, s := range s14Electives {
			s.Level = level
			subjects = append(subjects, s)
		}
	}
	for _, level := range []string{"S5", "S6"} {
		for _, s := range s56Principal {
			s.Level = level
			subjects = append(subjects, s)
		}
		for _, s := range s56Subsidiary {
			s.Level = level
			subjects = append(subjects, s)
		}
	}

	for _, subject := range subjects {
		db.Create(&subject)
	}

	log.Printf("Seeded %d subjects for all levels", len(subjects))
}

func cleanupDuplicates(db *gorm.DB) {
	// Find Tanna Memorial school
	var school models.School
	if err := db.Where("name LIKE ?", "%Tanna%").First(&school).Error; err != nil {
		log.Println("Tanna Memorial school not found")
		return
	}

	log.Printf("Cleaning up duplicates for school: %s", school.Name)

	// Remove duplicate classes - keep only the first occurrence of each unique combination
	var classes []models.Class
	db.Where("school_id = ?", school.ID).Find(&classes)

	seen := make(map[string]uuid.UUID)
	var toDelete []uuid.UUID

	for _, class := range classes {
		key := fmt.Sprintf("%s-%s-%d-%s", class.SchoolID, class.Level, class.Year, class.Term)
		if _, exists := seen[key]; exists {
			// This is a duplicate, mark for deletion
			toDelete = append(toDelete, class.ID)
			log.Printf("Marking duplicate class for deletion: %s %s %d %s", class.Level, class.Term, class.Year, class.ID)
		} else {
			// First occurrence, keep it
			seen[key] = class.ID
		}
	}

	// Delete duplicates
	if len(toDelete) > 0 {
		result := db.Where("id IN ?", toDelete).Delete(&models.Class{})
		log.Printf("Deleted %d duplicate classes", result.RowsAffected)
	} else {
		log.Println("No duplicate classes found")
	}

	// Remove duplicate subjects
	var subjects []models.Subject
	db.Where("school_id = ?", school.ID).Find(&subjects)

	subjectSeen := make(map[string]uuid.UUID)
	var subjectsToDelete []uuid.UUID

	for _, subject := range subjects {
		key := fmt.Sprintf("%s-%s-%s", subject.SchoolID, subject.Name, subject.Level)
		if _, exists := subjectSeen[key]; exists {
			subjectsToDelete = append(subjectsToDelete, subject.ID)
			log.Printf("Marking duplicate subject for deletion: %s %s %s", subject.Name, subject.Level, subject.ID)
		} else {
			subjectSeen[key] = subject.ID
		}
	}

	if len(subjectsToDelete) > 0 {
		result := db.Where("id IN ?", subjectsToDelete).Delete(&models.Subject{})
		log.Printf("Deleted %d duplicate subjects", result.RowsAffected)
	} else {
		log.Println("No duplicate subjects found")
	}

	// Remove duplicate grading rules
	var rules []models.GradingRule
	db.Where("school_id = ?", school.ID).Find(&rules)

	ruleSeen := make(map[string]uuid.UUID)
	var rulesToDelete []uuid.UUID

	for _, rule := range rules {
		key := fmt.Sprintf("%s-%s", rule.SchoolID, rule.Level)
		if _, exists := ruleSeen[key]; exists {
			rulesToDelete = append(rulesToDelete, rule.ID)
			log.Printf("Marking duplicate grading rule for deletion: %s %s", rule.Level, rule.ID)
		} else {
			ruleSeen[key] = rule.ID
		}
	}

	if len(rulesToDelete) > 0 {
		result := db.Where("id IN ?", rulesToDelete).Delete(&models.GradingRule{})
		log.Printf("Deleted %d duplicate grading rules", result.RowsAffected)
	} else {
		log.Println("No duplicate grading rules found")
	}

	log.Println("Cleanup completed")
}

func standardizeClassNames(db *gorm.DB) {
	log.Println("Standardizing class names...")

	var classes []models.Class
	db.Find(&classes)

	for _, class := range classes {
		standardName := fmt.Sprintf("%s %s %d", class.Level, class.Term, class.Year)
		if class.Name != standardName {
			log.Printf("Updating class name from '%s' to '%s'", class.Name, standardName)
			db.Model(&class).Update("name", standardName)
		}
	}

	log.Println("Class name standardization completed")
}

func standardizeSubjects(db *gorm.DB) {
	log.Println("Standardizing subjects across all schools...")

	// Don't delete existing subjects due to foreign key constraints
	// Instead, we'll add missing subjects from standards

	// Get all schools
	var schools []models.School
	db.Find(&schools)

	for _, school := range schools {
		log.Printf("Creating subjects for school: %s", school.Name)

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

		// Create subjects for each level using standard subject service
		standardSubjectService := services.NewStandardSubjectService(db)
		err := standardSubjectService.CreateSchoolSubjectsFromStandard(school.ID, levels)
		if err != nil {
			log.Printf("Error creating subjects for school %s: %v", school.Name, err)
		}
	}

	log.Println("Subject standardization completed")
}

func seedStandardSubjects(db *gorm.DB) {
	log.Println("Seeding standard subjects from curriculum...")

	// Clear existing standard subjects
	db.Exec("DELETE FROM standard_subjects")

	// Pre-Primary (Nursery) subjects
	nurserySubjects := []models.StandardSubject{
		{Name: "Language & Early Literacy", Code: "LIT", Level: "Nursery", IsCompulsory: true, Papers: 1, GradingType: "developmental", Description: "Language development and early literacy skills"},
		{Name: "Early Numeracy", Code: "NUM", Level: "Nursery", IsCompulsory: true, Papers: 1, GradingType: "developmental", Description: "Basic number concepts and mathematical thinking"},
		{Name: "Social & Emotional Development", Code: "SED", Level: "Nursery", IsCompulsory: true, Papers: 1, GradingType: "developmental", Description: "Social skills and emotional regulation"},
		{Name: "Creative Arts", Code: "ART", Level: "Nursery", IsCompulsory: true, Papers: 1, GradingType: "developmental", Description: "Music, drama, and art activities"},
		{Name: "Physical & Motor Skills", Code: "PMS", Level: "Nursery", IsCompulsory: true, Papers: 1, GradingType: "developmental", Description: "Gross and fine motor skill development"},
		{Name: "Health, Hygiene & Nutrition", Code: "HHN", Level: "Nursery", IsCompulsory: true, Papers: 1, GradingType: "developmental", Description: "Health awareness and hygiene practices"},
		{Name: "Play & Environmental Awareness", Code: "PEA", Level: "Nursery", IsCompulsory: true, Papers: 1, GradingType: "developmental", Description: "Environmental awareness through play"},
	}

	// P1-P3 thematic subjects
	p13Subjects := []models.StandardSubject{
		{Name: "Literacy", Code: "LIT", Level: "P1", IsCompulsory: true, Papers: 1, GradingType: "descriptive", Description: "Reading, writing, and communication skills"},
		{Name: "Numeracy", Code: "NUM", Level: "P1", IsCompulsory: true, Papers: 1, GradingType: "descriptive", Description: "Basic mathematical concepts and problem solving"},
		{Name: "Life Skills", Code: "LS", Level: "P1", IsCompulsory: true, Papers: 1, GradingType: "descriptive", Description: "Personal and social life skills"},
		{Name: "Creative Arts", Code: "ART", Level: "P1", IsCompulsory: true, Papers: 1, GradingType: "descriptive", Description: "Creative expression through arts"},
		{Name: "Environment", Code: "ENV", Level: "P1", IsCompulsory: true, Papers: 1, GradingType: "descriptive", Description: "Environmental awareness and science concepts"},
	}

	// P4-P7 subjects
	p47Subjects := []models.StandardSubject{
		{Name: "English Language", Code: "ENG", Level: "P4", IsCompulsory: true, Papers: 1, GradingType: "standard", Description: "English language skills and communication"},
		{Name: "Mathematics", Code: "MATH", Level: "P4", IsCompulsory: true, Papers: 1, GradingType: "standard", Description: "Mathematical concepts and problem solving"},
		{Name: "Integrated Science", Code: "SCI", Level: "P4", IsCompulsory: true, Papers: 1, GradingType: "standard", Description: "Basic science concepts and investigations"},
		{Name: "Social Studies", Code: "SST", Level: "P4", IsCompulsory: true, Papers: 1, GradingType: "standard", Description: "Social studies including CRE/IRE"},
		{Name: "Local Language", Code: "LL", Level: "P4", IsCompulsory: true, Papers: 1, GradingType: "standard", Description: "Local language (Acoli, Luganda, Lango, or Lumasaba)"},
		{Name: "Creative Arts", Code: "ART", Level: "P4", IsCompulsory: true, Papers: 1, GradingType: "standard", Description: "Creative and performing arts"},
		{Name: "Physical Education", Code: "PE", Level: "P4", IsCompulsory: true, Papers: 1, GradingType: "standard", Description: "Physical fitness and sports"},
		{Name: "Agriculture", Code: "AGR", Level: "P4", IsCompulsory: true, Papers: 1, GradingType: "standard", Description: "Agricultural practices and environmental education"},
		{Name: "ICT", Code: "ICT", Level: "P4", IsCompulsory: false, Papers: 1, GradingType: "standard", Description: "Information and Communication Technology"},
	}

	// S1-S4 compulsory subjects
	s14Compulsory := []models.StandardSubject{
		{Name: "English Language", Code: "ENG", Level: "S1", IsCompulsory: true, Papers: 1, GradingType: "ncdc", Description: "English language and communication"},
		{Name: "Mathematics", Code: "MATH", Level: "S1", IsCompulsory: true, Papers: 1, GradingType: "ncdc", Description: "Mathematical concepts and problem solving"},
		{Name: "Physics", Code: "PHY", Level: "S1", IsCompulsory: true, Papers: 1, GradingType: "ncdc", Description: "Physical science and physics concepts"},
		{Name: "Chemistry", Code: "CHEM", Level: "S1", IsCompulsory: true, Papers: 1, GradingType: "ncdc", Description: "Chemical science and laboratory work"},
		{Name: "Biology", Code: "BIO", Level: "S1", IsCompulsory: true, Papers: 1, GradingType: "ncdc", Description: "Biological science and life processes"},
		{Name: "Geography", Code: "GEO", Level: "S1", IsCompulsory: true, Papers: 1, GradingType: "ncdc", Description: "Physical and human geography"},
		{Name: "History & Political Education", Code: "HIST", Level: "S1", IsCompulsory: true, Papers: 1, GradingType: "ncdc", Description: "Historical studies and political education"},
		{Name: "Christian Religious Education", Code: "CRE", Level: "S1", IsCompulsory: true, Papers: 1, GradingType: "ncdc", Description: "Christian religious studies"},
		{Name: "Islamic Religious Education", Code: "IRE", Level: "S1", IsCompulsory: true, Papers: 1, GradingType: "ncdc", Description: "Islamic religious studies"},
		{Name: "Entrepreneurship Education", Code: "ENT", Level: "S1", IsCompulsory: true, Papers: 1, GradingType: "ncdc", Description: "Business and entrepreneurship skills"},
		{Name: "Kiswahili", Code: "KIS", Level: "S1", IsCompulsory: true, Papers: 1, GradingType: "ncdc", Description: "Kiswahili language"},
		{Name: "Physical Education", Code: "PE", Level: "S1", IsCompulsory: true, Papers: 1, GradingType: "ncdc", Description: "Physical fitness and sports"},
	}

	// S1-S4 electives
	s14Electives := []models.StandardSubject{
		{Name: "ICT / Computer Studies", Code: "ICT", Level: "S1", IsCompulsory: false, Papers: 1, GradingType: "ncdc", Description: "Information and Communication Technology"},
		{Name: "Agriculture", Code: "AGR", Level: "S1", IsCompulsory: false, Papers: 1, GradingType: "ncdc", Description: "Agricultural science and practices"},
		{Name: "Literature in English", Code: "LIT", Level: "S1", IsCompulsory: false, Papers: 1, GradingType: "ncdc", Description: "English literature and literary analysis"},
		{Name: "Art and Design", Code: "AD", Level: "S1", IsCompulsory: false, Papers: 1, GradingType: "ncdc", Description: "Visual arts and design"},
	}

	// S5-S6 Principal subjects
	s56Principal := []models.StandardSubject{
		{Name: "Mathematics", Code: "MATH", Level: "S5", IsCompulsory: false, Papers: 2, GradingType: "uneb", Description: "Advanced mathematics"},
		{Name: "Physics", Code: "PHY", Level: "S5", IsCompulsory: false, Papers: 3, GradingType: "uneb", Description: "Advanced physics"},
		{Name: "Chemistry", Code: "CHEM", Level: "S5", IsCompulsory: false, Papers: 3, GradingType: "uneb", Description: "Advanced chemistry"},
		{Name: "Biology", Code: "BIO", Level: "S5", IsCompulsory: false, Papers: 3, GradingType: "uneb", Description: "Advanced biology"},
		{Name: "Geography", Code: "GEO", Level: "S5", IsCompulsory: false, Papers: 3, GradingType: "uneb", Description: "Advanced geography"},
		{Name: "History & Political Education", Code: "HIST", Level: "S5", IsCompulsory: false, Papers: 3, GradingType: "uneb", Description: "Advanced history and political education"},
		{Name: "Religious Education", Code: "RE", Level: "S5", IsCompulsory: false, Papers: 2, GradingType: "uneb", Description: "Advanced religious studies"},
		{Name: "Entrepreneurship Education", Code: "ENT", Level: "S5", IsCompulsory: false, Papers: 2, GradingType: "uneb", Description: "Advanced entrepreneurship"},
		{Name: "Agriculture", Code: "AGR", Level: "S5", IsCompulsory: false, Papers: 3, GradingType: "uneb", Description: "Advanced agriculture"},
		{Name: "Economics", Code: "ECON", Level: "S5", IsCompulsory: false, Papers: 2, GradingType: "uneb", Description: "Economic theory and practice"},
		{Name: "Luganda", Code: "LUG", Level: "S5", IsCompulsory: false, Papers: 2, GradingType: "uneb", Description: "Luganda language and literature"},
		{Name: "Art and Design", Code: "AD", Level: "S5", IsCompulsory: false, Papers: 2, GradingType: "uneb", Description: "Advanced art and design"},
		{Name: "Literature", Code: "LIT", Level: "S5", IsCompulsory: false, Papers: 2, GradingType: "uneb", Description: "Advanced literature studies"},
	}

	// S5-S6 Subsidiary subjects
	s56Subsidiary := []models.StandardSubject{
		{Name: "General Paper", Code: "GP", Level: "S5", IsCompulsory: true, Papers: 1, GradingType: "uneb_subsidiary", Description: "General knowledge and critical thinking"},
		{Name: "Information Communication Technology", Code: "ICT", Level: "S5", IsCompulsory: false, Papers: 1, GradingType: "uneb_subsidiary", Description: "Advanced ICT skills"},
		{Name: "Subsidiary Mathematics", Code: "SUBMATH", Level: "S5", IsCompulsory: false, Papers: 1, GradingType: "uneb_subsidiary", Description: "Basic mathematics for non-math students"},
	}

	// Create all standard subjects
	var allSubjects []models.StandardSubject
	allSubjects = append(allSubjects, nurserySubjects...)

	// Add P1-P3 subjects for each level
	for _, level := range []string{"P1", "P2", "P3"} {
		for _, s := range p13Subjects {
			s.Level = level
			allSubjects = append(allSubjects, s)
		}
	}

	// Add P4-P7 subjects for each level
	for _, level := range []string{"P4", "P5", "P6", "P7"} {
		for _, s := range p47Subjects {
			s.Level = level
			allSubjects = append(allSubjects, s)
		}
	}

	// Add S1-S4 subjects for each level
	for _, level := range []string{"S1", "S2", "S3", "S4"} {
		for _, s := range s14Compulsory {
			s.Level = level
			allSubjects = append(allSubjects, s)
		}
		for _, s := range s14Electives {
			s.Level = level
			allSubjects = append(allSubjects, s)
		}
	}

	// Add S5-S6 subjects for each level
	for _, level := range []string{"S5", "S6"} {
		for _, s := range s56Principal {
			s.Level = level
			allSubjects = append(allSubjects, s)
		}
		for _, s := range s56Subsidiary {
			s.Level = level
			allSubjects = append(allSubjects, s)
		}
	}

	// Insert all subjects
	for _, subject := range allSubjects {
		db.Create(&subject)
	}

	log.Printf("Seeded %d standard subjects from curriculum", len(allSubjects))
}

func migrateToStandardSubjects(db *gorm.DB) {
	log.Println("Migrating existing data to use standard subjects...")

	// First, ensure standard subjects are seeded
	var count int64
	db.Model(&models.StandardSubject{}).Count(&count)
	if count == 0 {
		log.Println("No standard subjects found. Seeding first...")
		seedStandardSubjects(db)
	}

	// Update subject_results to reference standard_subjects
	log.Println("Updating subject results to reference standard subjects...")

	// Get all subject results and their associated school subjects
	type ResultWithSubject struct {
		models.SubjectResult
		SubjectName  string `json:"subject_name"`
		SubjectLevel string `json:"subject_level"`
	}

	var results []ResultWithSubject
	db.Table("subject_results").
		Select("subject_results.*, subjects.name as subject_name, subjects.level as subject_level").
		Joins("LEFT JOIN subjects ON subject_results.subject_id = subjects.id").
		Where("subjects.id IS NOT NULL").
		Scan(&results)

	for _, result := range results {
		if result.SubjectName != "" {
			// Find matching standard subject
			var standardSubject models.StandardSubject
			err := db.Where("name = ? AND level = ?", result.SubjectName, result.SubjectLevel).First(&standardSubject).Error
			if err == nil {
				// Update the result to reference the standard subject
				db.Model(&models.SubjectResult{}).Where("id = ?", result.ID).Update("subject_id", standardSubject.ID)
				log.Printf("Updated result %s to use standard subject %s", result.ID, standardSubject.Name)
			} else {
				log.Printf("Warning: Could not find standard subject for %s %s", result.SubjectName, result.SubjectLevel)
			}
		}
	}

	// Update assessments to reference standard_subjects
	log.Println("Updating assessments to reference standard subjects...")

	type AssessmentWithSubject struct {
		models.Assessment
		SubjectName  string `json:"subject_name"`
		SubjectLevel string `json:"subject_level"`
	}

	var assessments []AssessmentWithSubject
	db.Table("assessments").
		Select("assessments.*, subjects.name as subject_name, subjects.level as subject_level").
		Joins("LEFT JOIN subjects ON assessments.subject_id = subjects.id").
		Where("subjects.id IS NOT NULL").
		Scan(&assessments)

	for _, assessment := range assessments {
		if assessment.SubjectName != "" {
			// Find matching standard subject
			var standardSubject models.StandardSubject
			err := db.Where("name = ? AND level = ?", assessment.SubjectName, assessment.SubjectLevel).First(&standardSubject).Error
			if err == nil {
				// Update the assessment to reference the standard subject
				db.Model(&models.Assessment{}).Where("id = ?", assessment.ID).Update("subject_id", standardSubject.ID)
				log.Printf("Updated assessment %s to use standard subject %s", assessment.ID, standardSubject.Name)
			} else {
				log.Printf("Warning: Could not find standard subject for %s %s", assessment.SubjectName, assessment.SubjectLevel)
			}
		}
	}

	log.Println("Migration to standard subjects completed")
	log.Println("Note: Old school-specific subjects are preserved but no longer used for new data")
}

func fixForeignKeys(db *gorm.DB) {
	log.Println("Fixing foreign key constraints...")

	// Drop existing foreign key constraints
	db.Exec("ALTER TABLE subject_results DROP FOREIGN KEY fk_subject_results_subject")
	db.Exec("ALTER TABLE assessments DROP FOREIGN KEY fk_assessments_subject")

	// Add new foreign key constraints pointing to standard_subjects
	db.Exec("ALTER TABLE subject_results ADD CONSTRAINT fk_subject_results_standard_subject FOREIGN KEY (subject_id) REFERENCES standard_subjects(id)")
	db.Exec("ALTER TABLE assessments ADD CONSTRAINT fk_assessments_standard_subject FOREIGN KEY (subject_id) REFERENCES standard_subjects(id)")

	log.Println("Foreign key constraints fixed")
}

func cleanupOrphanedData(db *gorm.DB) {
	log.Println("Cleaning up orphaned data...")

	// Delete users with non-existent school_id (except system admins)
	result := db.Exec("DELETE FROM users WHERE school_id IS NOT NULL AND role != 'system_admin' AND school_id NOT IN (SELECT id FROM schools)")
	log.Printf("Deleted %d orphaned users", result.RowsAffected)

	// Delete students with non-existent school_id
	result = db.Exec("DELETE FROM students WHERE school_id NOT IN (SELECT id FROM schools)")
	log.Printf("Deleted %d orphaned students", result.RowsAffected)

	// Delete classes with non-existent school_id
	result = db.Exec("DELETE FROM classes WHERE school_id NOT IN (SELECT id FROM schools)")
	log.Printf("Deleted %d orphaned classes", result.RowsAffected)

	// Delete subjects with non-existent school_id
	result = db.Exec("DELETE FROM subjects WHERE school_id NOT IN (SELECT id FROM schools)")
	log.Printf("Deleted %d orphaned subjects", result.RowsAffected)

	// Delete subject_results with non-existent school_id
	result = db.Exec("DELETE FROM subject_results WHERE school_id NOT IN (SELECT id FROM schools)")
	log.Printf("Deleted %d orphaned subject results", result.RowsAffected)

	// Delete assessments with non-existent school_id
	result = db.Exec("DELETE FROM assessments WHERE school_id NOT IN (SELECT id FROM schools)")
	log.Printf("Deleted %d orphaned assessments", result.RowsAffected)

	// Delete grading_rules with non-existent school_id
	result = db.Exec("DELETE FROM grading_rules WHERE school_id NOT IN (SELECT id FROM schools)")
	log.Printf("Deleted %d orphaned grading rules", result.RowsAffected)

	log.Println("Orphaned data cleanup completed")
}
