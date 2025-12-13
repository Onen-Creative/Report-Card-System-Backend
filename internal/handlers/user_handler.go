package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/school-system/backend/internal/models"
	"github.com/school-system/backend/internal/services"
	"gorm.io/gorm"
)

type UserHandler struct {
	db          *gorm.DB
	authService *services.AuthService
	auditService *services.AuditService
}

func NewUserHandler(db *gorm.DB, authService *services.AuthService) *UserHandler {
	return &UserHandler{
		db: db, 
		authService: authService,
		auditService: services.NewAuditService(db),
	}
}

func (h *UserHandler) List(c *gin.Context) {
	var users []models.User
	if err := h.db.Preload("School").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

func (h *UserHandler) Create(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
		FullName string `json:"full_name" binding:"required"`
		Role     string `json:"role" binding:"required"`
		SchoolID string `json:"school_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate school assignment
	if req.Role != "system_admin" && req.SchoolID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "School assignment required for non-system admin users"})
		return
	}

	user := &models.User{
		Email:    req.Email,
		FullName: req.FullName,
		Role:     req.Role,
		IsActive: true,
	}

	if req.Role == "system_admin" {
		user.SchoolID = nil
	} else {
		schoolID, err := uuid.Parse(req.SchoolID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid school_id"})
			return
		}
		user.SchoolID = &schoolID
	}

	if err := h.authService.CreateUser(user, req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Log audit
	if userID, exists := c.Get("user_id"); exists {
		h.auditService.Log(userID.(uuid.UUID), "CREATE", "user", user.ID, nil, models.JSONB{"name": user.FullName, "role": user.Role}, c.ClientIP())
	}

	c.JSON(http.StatusCreated, user)
}

func (h *UserHandler) Get(c *gin.Context) {
	id := c.Param("id")
	var user models.User
	if err := h.db.Preload("School").First(&user, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var user models.User
	if err := h.db.First(&user, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var req struct {
		Email    string `json:"email"`
		FullName string `json:"full_name"`
		Role     string `json:"role"`
		IsActive *bool  `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Email != "" {
		user.Email = req.Email
	}
	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.Role != "" {
		user.Role = req.Role
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	if err := h.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Log audit
	if userID, exists := c.Get("user_id"); exists {
		h.auditService.Log(userID.(uuid.UUID), "UPDATE", "user", user.ID, nil, models.JSONB{"name": user.FullName}, c.ClientIP())
	}

	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	var user models.User
	h.db.First(&user, "id = ?", id)
	
	if err := h.db.Delete(&models.User{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Log audit
	if userID, exists := c.Get("user_id"); exists {
		h.auditService.Log(userID.(uuid.UUID), "DELETE", "user", uuid.MustParse(id), models.JSONB{"name": user.FullName}, nil, c.ClientIP())
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}
