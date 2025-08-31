package controllers

import (
	"backend/config"
	"backend/services"
	"net/http"
	"strconv"
	"time"

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
	Sex              string  `json:"sex"` 
}
type OnboardingInput struct {
    Birthday         string   `json:"birthday" binding:"required,datetime=2006-01-02"`
    Height           float64  `json:"height" binding:"required"`
    Weight           float64  `json:"weight" binding:"required"`
    HealthConditions []string `json:"health_conditions"`
    FitnessGoals     []string `json:"fitness_goals"`
    ProfilePicture   string   `json:"profile_picture"`
    MFAEnabled       bool     `json:"mfa_enabled"`
	Sex              string   `json:"sex"`
}

type changePasswordInput struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
	ConfirmPassword string `json:"confirm_password"`
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
		input.Sex,
    ); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "profile updated successfully"})
}


func GetBMI(c *gin.Context) {
	email := c.GetString("email")

	var (
		overrideH *float64
		overrideW *float64
	)

	if h := c.Query("height_cm"); h != "" {
		if v, err := strconv.ParseFloat(h, 64); err == nil && v > 0 {
			overrideH = &v
		} else if h != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid height_cm"})
			return
		}
	}
	if w := c.Query("weight_kg"); w != "" {
		if v, err := strconv.ParseFloat(w, 64); err == nil && v > 0 {
			overrideW = &v
		} else if w != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid weight_kg"})
			return
		}
	}

	result, err := services.GetUserBMI(email, overrideH, overrideW)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}


func ChangePassword(c *gin.Context) {
	email := c.GetString("email")

	var in changePasswordInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if in.ConfirmPassword != "" && in.NewPassword != in.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error": "new_password and confirm_password do not match"})
		return
	}

	if err := services.ChangePassword(email, in.CurrentPassword, in.NewPassword); err != nil {
		// Map common errors to friendly codes/messages
		switch err.Error() {
		case "user not found or disabled":
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case "current password is incorrect":
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password updated successfully"})
}
