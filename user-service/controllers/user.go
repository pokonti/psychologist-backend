package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetProfile(c *gin.Context) {
	userID := c.GetHeader("X-USER-ID")
	role := c.GetHeader("X-User-Role")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user id"})
		return
	}

	profile, err := profileRepo

	c.JSON(http.StatusOK, gin.H{
		"id":   userID,
		"role": role,
	})
}
