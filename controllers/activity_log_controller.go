// controllers/activity_log_controller.go
package controllers

import (
	"net/http"

	"backend/config"
	"backend/models"
	"backend/services"
	"github.com/gin-gonic/gin"
)

// UpdateDailyActivity handles manual updates for hydration and exercise intake for the current day
func UpdateDailyActivity(c *gin.Context) {
	email := c.MustGet("email").(string)

	var user models.User
	if err := config.DB.Where("email = ?", email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	var body struct {
		Hydration float64 `json:"hydration"`
		Exercise  float64 `json:"exercise"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := services.UpsertDailyActivity(user.ID, body.Hydration, body.Exercise); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
