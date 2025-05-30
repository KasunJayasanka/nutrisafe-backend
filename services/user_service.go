package services

import (
	"errors"
	"backend/config"
	"backend/models"
	"backend/utils"
	"fmt"
)

type ProfileInput struct {
	HealthConditions string `json:"health_conditions"`
	FitnessGoals     string `json:"fitness_goals"`
	ProfilePicture   string `json:"profile_picture"`
}

func GetUserProfile(email string) (*models.User, error) {
	var user models.User
	result := config.DB.Where("email = ? AND disabled = ?", email, false).First(&user)
	if result.Error != nil {
		return nil, errors.New("user not found or disabled")
	}
	return &user, nil
}


func UpdateUserProfile(email string, input ProfileInput) error {
	var user models.User
	result := config.DB.Where("email = ? AND disabled = ?", email, false).First(&user)
	if result.Error != nil {
		return errors.New("user not found or disabled")
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
