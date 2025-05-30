package controllers

import (
	"net/http"
	"backend/services"

	"github.com/gin-gonic/gin"
)

type ProfileInput struct {
	HealthConditions string `json:"health_conditions"`
	FitnessGoals     string `json:"fitness_goals"`
}

func GetProfile(c *gin.Context) {
	email := c.GetString("email")
	profile, err := services.GetUserProfile(email)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, profile)
}

func UpdateProfile(c *gin.Context) {
	email := c.GetString("email")
	var input ProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := services.UpdateUserProfile(email, input.HealthConditions, input.FitnessGoals)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "profile updated successfully"})
}
