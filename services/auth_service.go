package services

import (
    "errors"
    "backend/config"
    "backend/models"
    "backend/utils"
    "fmt"
	"math/rand"
	"strings"
	"time"
)


func RegisterUser(email, password, firstName, lastName string) error {
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return err
	}

	rand.Seed(time.Now().UnixNano())
	base := strings.ToLower(strings.ReplaceAll(firstName, " ", ""))
	userID := fmt.Sprintf("%s%d", base, rand.Intn(100000))

	user := models.User{
		UserID:    userID,
		Email:     email,
		Password:  hashedPassword,
		FirstName: firstName,
		LastName:  lastName,
		Disabled:  false,
	}

	result := config.DB.Create(&user)
	return result.Error
}


func AuthenticateUser(email, password string) (string, error) {
    var user models.User
    result := config.DB.Where("email = ? AND disabled = ?", email, false).First(&user)
    if result.Error != nil {
        return "", errors.New("user not found or disabled")
    }

    if !utils.CheckPasswordHash(password, user.Password) {
        return "", errors.New("incorrect password")
    }

    token, err := utils.GenerateJWT(user.Email)
    if err != nil {
        return "", err
    }

    return token, nil
}

