package controllers

import (
	"strconv"
	"time"

	"backend/config"
	"backend/models"
	"backend/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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



func UpdateMeal(c *gin.Context) {
	// parse meal ID
	idParam := c.Param("id")
	mealID, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid meal id"})
		return
	}

	// bind payload
	var body struct {
		Type  string                     `json:"type"`
		AteAt time.Time                  `json:"ate_at"`
		Items []services.MealItemRequest `json:"items"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// lookup user
	email := c.GetString("email")
	var u models.User
	config.DB.First(&u, "email = ?", email)

	// call service
	eda := services.NewEdamamService()
	rek, svcErr := services.NewRekognitionService()
	if svcErr != nil {
		c.JSON(500, gin.H{"error": svcErr.Error()})
		return
	}
	mealSvc := services.NewMealService(services.NewFoodService(eda, rek))
	updatedMeal, err := mealSvc.UpdateMeal(u.ID, uint(mealID), body.Type, body.AteAt, body.Items)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, updatedMeal)
}

func DeleteMeal(c *gin.Context) {
	// parse meal ID
	idParam := c.Param("id")
	mealID, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid meal id"})
		return
	}

	// lookup user
	email := c.GetString("email")
	var u models.User
	config.DB.First(&u, "email = ?", email)

	// call service
	eda := services.NewEdamamService()
	rek, svcErr := services.NewRekognitionService()
	if svcErr != nil {
		c.JSON(500, gin.H{"error": svcErr.Error()})
		return
	}
	mealSvc := services.NewMealService(services.NewFoodService(eda, rek))
	if err := mealSvc.DeleteMeal(u.ID, uint(mealID)); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.Status(204)
}

func GetMealByID(c *gin.Context) {
	// 1) parse :id
	idParam := c.Param("id")
	mealID64, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid meal id"})
		return
	}
	mealID := uint(mealID64)

	// 2) look up current user
	email := c.GetString("email")
	var u models.User
	if err := config.DB.First(&u, "email = ?", email).Error; err != nil {
		c.JSON(500, gin.H{"error": "could not find user"})
		return
	}

	// 3) build foodSvc → mealSvc
	edaSvc := services.NewEdamamService()
	rekSvc, rekErr := services.NewRekognitionService()
	if rekErr != nil {
		c.JSON(500, gin.H{"error": rekErr.Error()})
		return
	}
	foodSvc := services.NewFoodService(edaSvc, rekSvc)
	mealSvc := services.NewMealService(foodSvc)

	// 4) fetch the meal
	meal, err := mealSvc.GetMeal(u.ID, mealID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(404, gin.H{"error": "meal not found"})
		} else {
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}

	// 5) return it
	c.JSON(200, meal)
}