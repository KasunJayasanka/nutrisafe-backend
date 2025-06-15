package services

import (
	"errors"
	"backend/config"
	"backend/models"
	"backend/utils"
	"fmt"
	"time"
	"strings"
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

func GetUserProfile(email string) (map[string]interface{}, error) {
	var user models.User
	result := config.DB.Where("email = ? AND disabled = ?", email, false).First(&user)
	if result.Error != nil {
		return nil, errors.New("user not found or disabled")
	}

	age := 0
	if !user.Birthday.IsZero() {
		age = utils.CalculateAge(user.Birthday)
	}

	return map[string]interface{}{
		"id":               user.ID,
		"user_id":          user.UserID,
		"email":            user.Email,
		"first_name":       user.FirstName,
		"last_name":        user.LastName,
		"birthday":         user.Birthday.Format("2006-01-02"),
		"age":              age,
		"height":           user.Height,
		"weight":           user.Weight,
		"health_conditions": user.HealthConditions,
		"fitness_goals":     user.FitnessGoals,
		"profile_picture":   user.ProfilePicture,
		"mfa_enabled":       user.MFAEnabled,
		"onboarded":         user.Onboarded,
	}, nil
}



func UpdateUserProfile(email string, input ProfileInput) error {
    var user models.User
    result := config.DB.Where("email = ? AND disabled = ?", email, false).First(&user)
    if result.Error != nil {
        return errors.New("user not found or disabled")
    }

    if input.FirstName != "" {
        user.FirstName = input.FirstName
    }
    if input.LastName != "" {
        user.LastName = input.LastName
    }

    if input.Birthday != "" {
        birthday, err := time.Parse("2006-01-02", input.Birthday)
        if err == nil {
            user.Birthday = birthday
        }
    }

    if input.Height > 0 {
        user.Height = input.Height
    }
    if input.Weight > 0 {
        user.Weight = input.Weight
    }
    if input.HealthConditions != "" {
        user.HealthConditions = input.HealthConditions
    }
    if input.FitnessGoals != "" {
        user.FitnessGoals = input.FitnessGoals
    }
    if input.ProfilePicture != "" {
        url, err := utils.UploadBase64ImageToS3(input.ProfilePicture, user.Email)
        if err != nil {
            return fmt.Errorf("failed to upload image: %v", err)
        }
        user.ProfilePicture = url
    }

    user.Onboarded = input.Onboarded

    return config.DB.Save(&user).Error
}




func FindUserByEmail(email string) (*models.User, error) {
	var user models.User
	result := config.DB.First(&user, "email = ?", email)
	if result.Error != nil {
		return nil, errors.New("user not found")
	}
	return &user, nil
}

func DeleteUser(email string) error {
	var user models.User
	result := config.DB.First(&user, "email = ?", email)
	if result.Error != nil {
		return result.Error
	}
	user.Disabled = true
	return config.DB.Save(&user).Error
}


func CompleteUserOnboarding(
    email string,
    birthday time.Time,
    height, weight float64,
    healthConditions, fitnessGoals []string,
    profilePictureBase64 string,
    mfaEnabled bool,
) error {
    var user models.User
    if err := config.DB.
        Where("email = ? AND disabled = ?", email, false).
        First(&user).Error; err != nil {
        return errors.New("user not found or disabled")
    }

    // Update fields
    user.Birthday = birthday
    user.Height = height
    user.Weight = weight
    user.HealthConditions = strings.Join(healthConditions, ",")
    user.FitnessGoals = strings.Join(fitnessGoals, ",")
    user.MFAEnabled = mfaEnabled

    if profilePictureBase64 != "" {
        url, err := utils.UploadBase64ImageToS3(profilePictureBase64, "onboarding/"+user.Email)
        if err != nil {
            return fmt.Errorf("failed to upload profile picture: %w", err)
        }
        user.ProfilePicture = url
    }

    user.Onboarded = true  // ← This line enables the flag

    return config.DB.Save(&user).Error
}

