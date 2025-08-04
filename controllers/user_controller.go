package controllers

import (
	"net/http"
	"time"
	"backend/services"
	`backend/config`
	

	"github.com/gin-gonic/gin"
)


type ProfileInput struct {
    FirstName        string  `json:"first_name"`
    LastName         string  `json:"last_name"`
    Birthday         string  `json:"birthday"` // sent as YYYY-MM-DD
    Height           float64 `json:"height"`
    Weight           float64 `json:"weight"`
    HealthConditions string  `json:"health_conditions"`
    FitnessGoals     string  `json:"fitness_goals"`
    ProfilePicture   string  `json:"profile_picture"`
    Onboarded        bool    `json:"onboarded"`
}
type OnboardingInput struct {
    Birthday         string   `json:"birthday" binding:"required,datetime=2006-01-02"`
    Height           float64  `json:"height" binding:"required"`
    Weight           float64  `json:"weight" binding:"required"`
    HealthConditions []string `json:"health_conditions"`
    FitnessGoals     []string `json:"fitness_goals"`
    ProfilePicture   string   `json:"profile_picture"`
    MFAEnabled       bool     `json:"mfa_enabled"`
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


func OnboardUser(c *gin.Context) {
    email := c.GetString("email") // set by your auth middleware
    var input OnboardingInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // parse the date
    birthday, err := time.Parse("2006-01-02", input.Birthday)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid birthday format"})
        return
    }

    // persist everything
    if err := services.CompleteUserOnboarding(
        email, birthday,
        input.Height, input.Weight,
        input.HealthConditions,
        input.FitnessGoals,
        input.ProfilePicture,
        input.MFAEnabled,
    ); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "profile updated successfully"})
}


