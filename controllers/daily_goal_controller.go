// controllers/health_goal_controller.go
package controllers

import (
	"backend/config"
	"backend/models"
	"backend/services"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func GetGoals(c *gin.Context) {
    email := c.MustGet("email").(string)
    var user models.User
    if err := config.DB.Where("email = ?", email).First(&user).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
        return
    }

    goal, progress, err := services.GetGoalsAndProgress(user.ID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"goals": goal, "progress": progress})
}

func UpdateGoals(c *gin.Context) {
    email := c.MustGet("email").(string)
    var user models.User
    if err := config.DB.Where("email = ?", email).First(&user).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
        return
    }

    var req struct {
        Calories  float64  `json:"calories"`
        Protein   float64  `json:"protein"`
        Carbs     float64  `json:"carbs"`
        Fat       *float64 `json:"fat"`
        Sodium    *float64 `json:"sodium"`
        Sugar     *float64 `json:"sugar"`
        Hydration *float64 `json:"hydration"`
        Exercise  *float64 `json:"exercise"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // default missing to 0
    fat, sodium, sugar := 0.0, 0.0, 0.0
    if req.Fat != nil    { fat    = *req.Fat }
    if req.Sodium != nil { sodium = *req.Sodium }
    if req.Sugar != nil  { sugar  = *req.Sugar }

    hydration, exercise := 0.0, 0.0
    if req.Hydration != nil { hydration = *req.Hydration }
    if req.Exercise != nil  { exercise  = *req.Exercise  }

    if err := services.UpsertGoals(
        user.ID,
        req.Calories,
        req.Protein,
        req.Carbs,
        fat,
        sodium,
        sugar,
        hydration,
        exercise,
    ); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.Status(http.StatusNoContent)
}

func GetDailyProgressHistory(c *gin.Context) {
    email := c.MustGet("email").(string)
    var user models.User
    if err := config.DB.Where("email = ?", email).First(&user).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
        return
    }

    history, err := services.GetAllDailyProgress(user.ID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, history)
}


func GetGoalsByDate(c *gin.Context) {
	email := c.MustGet("email").(string)
	var user models.User
	if err := config.DB.Where("email = ?", email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	dateStr := c.Query("date") // expected format: YYYY-MM-DD
	if dateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'date' query param"})
		return
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format. Use YYYY-MM-DD"})
		return
	}

	goal, progress, err := services.GetGoalsAndProgressByDate(user.ID, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"date":    dateStr,
		"goals":   goal,
		"progress": progress,
	})
}
