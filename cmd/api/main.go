package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
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

	// Health check - simple endpoint that doesn't require DB
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "school-system-api"})
	})
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "School Management System API", "status": "running"})
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
			protected.GET("/subjects", subjectHandler.ListStandardSubjects)
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

	case "seed-standard-subjects":
		seedStandardSubjects(db)

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

func seedStandardSubjects(db *gorm.DB) {
	log.Println("Seeding standard subjects...")

	var count int64
	db.Model(&models.StandardSubject{}).Count(&count)
	if count > 0 {
		log.Println("Standard subjects already exist")
		return
	}

	// Pre-Primary (Nursery) Learning Domains
	nurseryDomains := []models.StandardSubject{
		{Name: "Language & Early Literacy", Code: "LEL", Level: "Nursery", IsCompulsory: true, Papers: 1, GradingType: "nursery", Description: "Language development and early literacy skills"},
		{Name: "Early Numeracy", Code: "EN", Level: "Nursery", IsCompulsory: true, Papers: 1, GradingType: "nursery", Description: "Basic number concepts and mathematical thinking"},
		{Name: "Social & Emotional Development", Code: "SED", Level: "Nursery", IsCompulsory: true, Papers: 1, GradingType: "nursery", Description: "Social skills and emotional development"},
		{Name: "Creative Arts", Code: "CA", Level: "Nursery", IsCompulsory: true, Papers: 1, GradingType: "nursery", Description: "Music, drama, and art activities"},
		{Name: "Physical & Motor Skills", Code: "PMS", Level: "Nursery", IsCompulsory: true, Papers: 1, GradingType: "nursery", Description: "Physical development and motor skills"},
		{Name: "Health, Hygiene & Nutrition", Code: "HHN", Level: "Nursery", IsCompulsory: true, Papers: 1, GradingType: "nursery", Description: "Health awareness and hygiene practices"},
		{Name: "Play & Environmental Awareness", Code: "PEA", Level: "Nursery", IsCompulsory: true, Papers: 1, GradingType: "nursery", Description: "Environmental awareness through play"},
	}

	// P1-P3 Thematic Subjects
	p13Subjects := []models.StandardSubject{
		{Name: "Literacy", Code: "LIT", Level: "P1", IsCompulsory: true, Papers: 1, GradingType: "primary_lower", Description: "Reading, writing, and communication skills"},
		{Name: "Numeracy", Code: "NUM", Level: "P1", IsCompulsory: true, Papers: 1, GradingType: "primary_lower", Description: "Basic mathematical concepts and problem solving"},
		{Name: "Life Skills", Code: "LS", Level: "P1", IsCompulsory: true, Papers: 1, GradingType: "primary_lower", Description: "Personal and social life skills"},
		{Name: "Creative Arts", Code: "CA", Level: "P1", IsCompulsory: true, Papers: 1, GradingType: "primary_lower", Description: "Creative expression through arts"},
		{Name: "Environment", Code: "ENV", Level: "P1", IsCompulsory: true, Papers: 1, GradingType: "primary_lower", Description: "Environmental awareness and science concepts"},
	}

	// P4-P7 Subjects
	p47Subjects := []models.StandardSubject{
		{Name: "English Language", Code: "ENG", Level: "P4", IsCompulsory: true, Papers: 1, GradingType: "primary_upper", Description: "English language and communication"},
		{Name: "Mathematics", Code: "MATH", Level: "P4", IsCompulsory: true, Papers: 1, GradingType: "primary_upper", Description: "Mathematical concepts and problem solving"},
		{Name: "Integrated Science", Code: "SCI", Level: "P4", IsCompulsory: true, Papers: 1, GradingType: "primary_upper", Description: "Scientific concepts and practical work"},
		{Name: "Social Studies", Code: "SST", Level: "P4", IsCompulsory: true, Papers: 1, GradingType: "primary_upper", Description: "Social studies including CRE/IRE"},
		{Name: "Local Language", Code: "LL", Level: "P4", IsCompulsory: true, Papers: 1, GradingType: "primary_upper", Description: "Acoli, Luganda, Lango, or Lumasaba"},
		{Name: "Creative Arts", Code: "CA", Level: "P4", IsCompulsory: true, Papers: 1, GradingType: "primary_upper", Description: "Creative and performing arts"},
		{Name: "Physical Education", Code: "PE", Level: "P4", IsCompulsory: true, Papers: 1, GradingType: "primary_upper", Description: "Physical fitness and sports"},
		{Name: "Agriculture / Environmental Education", Code: "AGR", Level: "P4", IsCompulsory: true, Papers: 1, GradingType: "primary_upper", Description: "Agricultural practices and environmental education"},
		{Name: "ICT", Code: "ICT", Level: "P4", IsCompulsory: false, Papers: 1, GradingType: "primary_upper", Description: "Information and Communication Technology"},
	}

	// S1-S4 Compulsory Subjects (S1-S2)
	s12Compulsory := []models.StandardSubject{
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

	// S1-S4 Electives
	s14Electives := []models.StandardSubject{
		{Name: "ICT / Computer Studies", Code: "ICT", Level: "S1", IsCompulsory: false, Papers: 1, GradingType: "ncdc", Description: "Information and Communication Technology"},
		{Name: "Agriculture", Code: "AGR", Level: "S1", IsCompulsory: false, Papers: 1, GradingType: "ncdc", Description: "Agricultural science and practices"},
		{Name: "Literature in English", Code: "LIT", Level: "S1", IsCompulsory: false, Papers: 1, GradingType: "ncdc", Description: "English literature and literary analysis"},
		{Name: "Art and Design", Code: "AD", Level: "S1", IsCompulsory: false, Papers: 1, GradingType: "ncdc", Description: "Visual arts and design"},
	}

	// S5-S6 Principal Subjects
	s56Principal := []models.StandardSubject{
		{Name: "Mathematics", Code: "MATH", Level: "S5", IsCompulsory: false, Papers: 2, GradingType: "uneb", Description: "Advanced mathematics"},
		{Name: "Physics", Code: "PHY", Level: "S5", IsCompulsory: false, Papers: 3, GradingType: "uneb", Description: "Advanced physics"},
		{Name: "Chemistry", Code: "CHEM", Level: "S5", IsCompulsory: false, Papers: 3, GradingType: "uneb", Description: "Advanced chemistry"},
		{Name: "Biology", Code: "BIO", Level: "S5", IsCompulsory: false, Papers: 3, GradingType: "uneb", Description: "Advanced biology"},
		{Name: "Geography", Code: "GEO", Level: "S5", IsCompulsory: false, Papers: 3, GradingType: "uneb", Description: "Advanced geography"},
		{Name: "History & Political Education", Code: "HIST", Level: "S5", IsCompulsory: false, Papers: 3, GradingType: "uneb", Description: "Advanced history and political education"},
		{Name: "Religious Education", Code: "RE", Level: "S5", IsCompulsory: false, Papers: 2, GradingType: "uneb", Description: "Advanced religious studies (CRE or IRE)"},
		{Name: "Entrepreneurship Education", Code: "ENT", Level: "S5", IsCompulsory: false, Papers: 2, GradingType: "uneb", Description: "Advanced entrepreneurship"},
		{Name: "Agriculture", Code: "AGR", Level: "S5", IsCompulsory: false, Papers: 3, GradingType: "uneb", Description: "Advanced agriculture"},
		{Name: "Economics", Code: "ECON", Level: "S5", IsCompulsory: false, Papers: 2, GradingType: "uneb", Description: "Economic theory and practice"},
		{Name: "Luganda", Code: "LUG", Level: "S5", IsCompulsory: false, Papers: 2, GradingType: "uneb", Description: "Luganda language and literature"},
		{Name: "Art and Design", Code: "AD", Level: "S5", IsCompulsory: false, Papers: 2, GradingType: "uneb", Description: "Advanced art and design"},
		{Name: "Literature", Code: "LIT", Level: "S5", IsCompulsory: false, Papers: 2, GradingType: "uneb", Description: "Advanced English literature"},
	}

	// S5-S6 Subsidiary Subjects
	s56Subsidiary := []models.StandardSubject{
		{Name: "General Paper", Code: "GP", Level: "S5", IsCompulsory: true, Papers: 1, GradingType: "uneb", Description: "General knowledge and current affairs"},
		{Name: "Information Communication Technology", Code: "ICT", Level: "S5", IsCompulsory: false, Papers: 1, GradingType: "uneb", Description: "Information and Communication Technology"},
		{Name: "Subsidiary Mathematics", Code: "SUBMATH", Level: "S5", IsCompulsory: false, Papers: 1, GradingType: "uneb", Description: "Subsidiary level mathematics"},
	}

	// Combine all subjects
	allSubjects := []models.StandardSubject{}
	allSubjects = append(allSubjects, nurseryDomains...)

	// Replicate P1-P3 subjects for P2 and P3
	for _, subject := range p13Subjects {
		allSubjects = append(allSubjects, subject)
		for _, level := range []string{"P2", "P3"} {
			subjectCopy := subject
			subjectCopy.Level = level
			allSubjects = append(allSubjects, subjectCopy)
		}
	}

	// Replicate P4-P7 subjects for P5, P6, and P7
	for _, subject := range p47Subjects {
		allSubjects = append(allSubjects, subject)
		for _, level := range []string{"P5", "P6", "P7"} {
			subjectCopy := subject
			subjectCopy.Level = level
			allSubjects = append(allSubjects, subjectCopy)
		}
	}

	// Replicate S1-S2 compulsory subjects for S2, S3, and S4
	for _, subject := range s12Compulsory {
		allSubjects = append(allSubjects, subject)
		for _, level := range []string{"S2", "S3", "S4"} {
			subjectCopy := subject
			subjectCopy.Level = level
			allSubjects = append(allSubjects, subjectCopy)
		}
	}

	// Replicate S1-S4 electives for S2, S3, and S4
	for _, subject := range s14Electives {
		allSubjects = append(allSubjects, subject)
		for _, level := range []string{"S2", "S3", "S4"} {
			subjectCopy := subject
			subjectCopy.Level = level
			allSubjects = append(allSubjects, subjectCopy)
		}
	}

	// Replicate S5-S6 subjects for S6
	for _, subject := range s56Principal {
		allSubjects = append(allSubjects, subject)
		subjectCopy := subject
		subjectCopy.Level = "S6"
		allSubjects = append(allSubjects, subjectCopy)
	}

	for _, subject := range s56Subsidiary {
		allSubjects = append(allSubjects, subject)
		subjectCopy := subject
		subjectCopy.Level = "S6"
		allSubjects = append(allSubjects, subjectCopy)
	}

	// Batch insert all subjects
	if err := db.CreateInBatches(allSubjects, 100).Error; err != nil {
		log.Fatal("Failed to seed standard subjects:", err)
	}

	log.Printf("Successfully seeded %d standard subjects", len(allSubjects))
}