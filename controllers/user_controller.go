package controllers

import (
	"net/http"
	"backend/services"
	`backend/config`
	

	"github.com/gin-gonic/gin"
)

type ProfileInput struct {
	HealthConditions string `json:"health_conditions"`
	FitnessGoals     string `json:"fitness_goals"`
	ProfilePicture   string `json:"profile_picture"` // base64
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
	var input services.ProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := services.UpdateUserProfile(email, input); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "profile updated successfully"})
}


func ToggleMFA(c *gin.Context) {
	email := c.GetString("email")
	var body struct {
		Enable bool `json:"enable"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := services.FindUserByEmail(email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found"})
		return
	}

	user.MFAEnabled = body.Enable
	config.DB.Save(&user)

	c.JSON(http.StatusOK, gin.H{"message": "MFA status updated", "mfa_enabled": user.MFAEnabled})
}


