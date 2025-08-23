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

func ListRecentMeals(c *gin.Context) {
	limit := 3
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			if n < 1 {
				n = 1
			}
			if n > 20 {
				n = 20
			}
			limit = n
		}
	}

	email := c.GetString("email")
	var u models.User
	if err := config.DB.First(&u, "email = ?", email).Error; err != nil {
		c.JSON(401, gin.H{"error": "user not found"})
		return
	}

	eda := services.NewEdamamService()
	rek, err := services.NewRekognitionService()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	mealSvc := services.NewMealService(services.NewFoodService(eda, rek))

	meals, err := mealSvc.ListRecentMeals(u.ID, limit)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"limit": limit, "meals": meals})
}

func ListRecentMealItems(c *gin.Context) {
	limit := 3
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			if n < 1 {
				n = 1
			}
			if n > 20 {
				n = 20
			}
			limit = n
		}
	}

	email := c.GetString("email")
	var u models.User
	if err := config.DB.First(&u, "email = ?", email).Error; err != nil {
		c.JSON(401, gin.H{"error": "user not found"})
		return
	}

	eda := services.NewEdamamService()
	rek, err := services.NewRekognitionService()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	mealSvc := services.NewMealService(services.NewFoodService(eda, rek))

	items, err := mealSvc.ListRecentMealItems(u.ID, limit)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"limit": limit, "items": items})
}

type mealWarningsQuery struct {
	From string `form:"from"` // RFC3339 (e.g., 2025-08-01T00:00:00+05:30)
	To   string `form:"to"`   // RFC3339
}

// GET /meals/warnings?from=...&to=...
func ListMealWarnings(c *gin.Context) {
	email := c.GetString("email")

	var u models.User
	if err := config.DB.First(&u, "email = ?", email).Error; err != nil {
		c.AbortWithStatusJSON(401, gin.H{"error": "user not found"})
		return
	}

	// Bind query params using Gin
	var q mealWarningsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.AbortWithStatusJSON(400, gin.H{"error": "invalid query parameters"})
		return
	}

	// Parse RFC3339 if provided
	var fromPtr, toPtr *time.Time
	if q.From != "" {
		if t, err := time.Parse(time.RFC3339, q.From); err == nil {
			fromPtr = &t
		} else {
			c.AbortWithStatusJSON(400, gin.H{"error": "invalid 'from' (must be RFC3339)"})
			return
		}
	}
	if q.To != "" {
		if t, err := time.Parse(time.RFC3339, q.To); err == nil {
			toPtr = &t
		} else {
			c.AbortWithStatusJSON(400, gin.H{"error": "invalid 'to' (must be RFC3339)"})
			return
		}
	}

	eda := services.NewEdamamService()
	rek, err := services.NewRekognitionService()
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		return
	}

	mealSvc := services.NewMealService(services.NewFoodService(eda, rek))
	res, err := mealSvc.ListMealsWithWarnings(u.ID, fromPtr, toPtr)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"meals": res})
}

// GET /meals/:id/warnings
func GetMealWarnings(c *gin.Context) {
	idParam := c.Param("id")
	mealID64, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{"error": "invalid meal id"})
		return
	}
	mealID := uint(mealID64)

	email := c.GetString("email")
	var u models.User
	if err := config.DB.First(&u, "email = ?", email).Error; err != nil {
		c.AbortWithStatusJSON(401, gin.H{"error": "user not found"})
		return
	}

	eda := services.NewEdamamService()
	rek, rerr := services.NewRekognitionService()
	if rerr != nil {
		c.AbortWithStatusJSON(500, gin.H{"error": rerr.Error()})
		return
	}

	mealSvc := services.NewMealService(services.NewFoodService(eda, rek))
	out, svcErr := mealSvc.GetMealWarnings(u.ID, mealID)
	if svcErr != nil {
		c.AbortWithStatusJSON(500, gin.H{"error": svcErr.Error()})
		return
	}

	c.JSON(200, out)
}
