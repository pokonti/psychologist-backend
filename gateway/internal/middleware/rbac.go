package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequireRoles accepts a list of allowed roles
func RequireRoles(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get role from context
		userRole, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		roleStr := userRole.(string)

		// Check if user's role is in the allowed list
		for _, role := range allowedRoles {
			if role == roleStr {
				c.Next()
				return
			}
		}

		// If loop finishes, role was not found
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "You do not have permission to access this resource",
		})
	}
}
