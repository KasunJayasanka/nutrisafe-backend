package services

import (
    "errors"
    "backend/config"
    "backend/models"
    "backend/utils"
)

func RegisterUser(email, password, fullName string) error {
    hashedPassword, err := utils.HashPassword(password)
    if err != nil {
        return err
    }

    user := models.User{
        Email:    email,
        Password: hashedPassword,
        FullName: fullName,
        Disabled: false, 
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

