package controllers

import (
	"time"

	"backend/config"
	"backend/models"
	"backend/services"

	"github.com/gin-gonic/gin"
)

func LogMeal(c *gin.Context) {
	var body struct {
		Type  string                     `json:"type"`
		AteAt time.Time                  `json:"ate_at"`
		Items []services.MealItemRequest `json:"items"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	email := c.GetString("email")
	var u models.User
	config.DB.First(&u, "email=?", email)

	eda := services.NewEdamamService()
	rek, err := services.NewRekognitionService()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	foodSvc := services.NewFoodService(eda, rek)
	mealSvc := services.NewMealService(foodSvc)
	meal, err := mealSvc.AddMeal(u.ID, body.Type, body.AteAt, body.Items)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, meal)
}

func ListMeals(c *gin.Context) {
	email := c.GetString("email")
	var u models.User
	config.DB.First(&u, "email=?", email)

	eda := services.NewEdamamService()
	rek, err := services.NewRekognitionService()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	foodSvc := services.NewFoodService(eda, rek)
	mealSvc := services.NewMealService(foodSvc)

	meals, err := mealSvc.ListMeals(u.ID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, meals)
}
