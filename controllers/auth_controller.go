package controllers

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"backend/config"
	"backend/models"
	"backend/services"
	"backend/utils"

	"github.com/gin-gonic/gin"
)

type RegisterInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	FullName string `json:"full_name" binding:"required"`
}

type VerifyInput struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

func Register(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := services.RegisterUser(input.Email, input.Password, input.FullName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "registration successful"})
}

type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func Login(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := services.FindUserByEmail(input.Email)
	if err != nil || !utils.CheckPasswordHash(input.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	if user.MFAEnabled {
		// Seed random for unique code
		rand.Seed(time.Now().UnixNano())
		code := fmt.Sprintf("%06d", rand.Intn(1000000))

		user.MFACode = code
		config.DB.Save(&user)

		err = utils.SendMFAEmail(user.Email, code)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send MFA code"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "MFA code sent to email"})
		return
	}

	// No MFA, generate token directly
	token, err := utils.GenerateJWT(user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func VerifyMFA(c *gin.Context) {
	var input VerifyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := services.FindUserByEmail(input.Email)
	if err != nil || user.MFACode != input.Code {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid MFA code"})
		return
	}

	token, err := utils.GenerateJWT(user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}

	user.MFACode = ""
	config.DB.Save(&user)

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func ForgotPassword(c *gin.Context) {
	var input struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	user, err := services.FindUserByEmail(input.Email)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a reset code has been sent"})
		return
	}

	token := utils.GenerateRandomToken(6) // 6-digit code
	user.ResetToken = token
	user.ResetTokenExp = time.Now().Add(15 * time.Minute)
	config.DB.Save(&user)

	utils.SendResetEmail(user.Email, token)

	c.JSON(http.StatusOK, gin.H{"message": "Password reset code sent to your email"})
}

func ResetPassword(c *gin.Context) {
	var input struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	var user models.User
	result := config.DB.Where("reset_token = ?", input.Token).First(&user)
	if result.Error != nil || time.Now().After(user.ResetTokenExp) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired token"})
		return
	}

	hashed, err := utils.HashPassword(input.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	user.Password = hashed
	user.ResetToken = ""
	user.ResetTokenExp = time.Time{}
	config.DB.Save(&user)

	c.JSON(http.StatusOK, gin.H{"message": "Password has been reset"})
}
