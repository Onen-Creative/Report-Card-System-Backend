package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TenantMiddleware ensures data isolation by school
func TenantMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole := c.GetString("user_role")
		
		// System admin can access all schools
		if userRole == "system_admin" {
			c.Next()
			return
		}

		// All other users must have a school_id
		schoolIDStr := c.GetString("school_id")
		if schoolIDStr == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: No school assigned"})
			c.Abort()
			return
		}

		// Validate school_id format
		if _, err := uuid.Parse(schoolIDStr); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid school ID"})
			c.Abort()
			return
		}

		c.Set("tenant_school_id", schoolIDStr)
		c.Next()
	}
}