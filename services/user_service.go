package services

import (
	"errors"
	"backend/config"
	"backend/models"
)

func GetUserProfile(email string) (*models.User, error) {
	var user models.User
	result := config.DB.First(&user, "email = ?", email)
	if result.Error != nil {
		return nil, errors.New("user not found")
	}
	return &user, nil
}

func UpdateUserProfile(email, healthConditions, fitnessGoals string) error {
	var user models.User
	result := config.DB.First(&user, "email = ?", email)
	if result.Error != nil {
		return errors.New("user not found")
	}

	user.HealthConditions = healthConditions
	user.FitnessGoals = fitnessGoals
	config.DB.Save(&user)
	return nil
}
