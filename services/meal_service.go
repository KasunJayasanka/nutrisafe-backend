package services

import (
	"backend/config"
	"backend/models"
	"backend/utils"
	"strings"
	"time"
)

type MealService struct {
	foodSvc *FoodService
}

func NewMealService(fs *FoodService) *MealService {
	return &MealService{foodSvc: fs}
}

type MealItemRequest struct {
	FoodID     string  `json:"food_id"`     // EdamamFoodID
	MeasureURI string  `json:"measure_uri"` // Edamam measure URI
	Quantity   float64 `json:"quantity"`
}

func (s *MealService) AddMeal(
	userID uint,
	mealType string,
	ateAt time.Time,
	items []MealItemRequest,
) (*models.Meal, error) {
	meal := &models.Meal{UserID: userID, Type: mealType, AteAt: ateAt}
	if err := config.DB.Create(meal).Error; err != nil {
		return nil, err
	}

	for _, it := range items {
		nut, err := s.foodSvc.Analyze(it.FoodID, it.MeasureURI, it.Quantity)
		if err != nil {
			return nil, err
		}
		warnings := utils.AssessFoodSafety(it.FoodID, nut)
		mi := &models.MealItem{
			MealID:     meal.ID,
			FoodLabel:  it.FoodID,
			Quantity:   it.Quantity,
			MeasureURI: it.MeasureURI,
			Calories:   nut["ENERC_KCAL"],
			Protein:    nut["PROCNT"],
			Carbs:      nut["CHOCDF"],
			Fat:        nut["FAT"],
			Sodium:     nut["NA"],
			Sugar:      nut["SUGAR"],
			Safe:       len(warnings) == 0,
			Warnings:   strings.Join(warnings, "; "),
		}
		config.DB.Create(mi)
	}
	return meal, nil
}

func (s *MealService) ListMeals(userID uint) ([]models.Meal, error) {
	var meals []models.Meal
	err := config.DB.
		Preload("Items").
		Where("user_id = ?", userID).
		Order("ate_at DESC").
		Find(&meals).Error
	return meals, err
}
