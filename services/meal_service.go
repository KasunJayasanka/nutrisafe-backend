// services/meal_service.go
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

// exactly your existing request type
type MealItemRequest struct {
	FoodID     string  `json:"food_id"`     // EdamamFoodID
	MeasureURI string  `json:"measure_uri"` // Edamam measure URI
	Quantity   float64 `json:"quantity"`
}

type ItemWarning struct {
	MealItemID uint   `json:"meal_item_id"`
	FoodLabel  string `json:"food_label"`
	Safe       bool   `json:"safe"`
	Warnings   string `json:"warnings"`
	Calories   float64 `json:"calories,omitempty"` // optional extras
}

type MealWarnings struct {
	MealID    uint          `json:"meal_id"`
	Type      string        `json:"type"`
	AteAt     time.Time     `json:"ate_at"`
	MealSafe  bool          `json:"meal_safe"`         // true if no unsafe items
	Warnings  []ItemWarning `json:"warnings_by_item"`  // only items that have warnings/unsafe
}

// lookupLabel will call your FoodService.SearchFoods(q)
// and try to match the returned EdamamFoodID back to the
// one we passed in.  If we find it, return its Label.
// Otherwise fall back to the raw ID.
func (s *MealService) lookupLabel(foodID string) string {
	// SearchFoods should already exist in your food_service.go
	foods, err := s.foodSvc.Search(foodID)
	if err != nil {
		return foodID
	}
	for _, f := range foods {
		if f.EdamamFoodID == foodID {
			return f.Label
		}
	}
	if len(foods) > 0 {
		return foods[0].Label
	}
	return foodID
}

func (s *MealService) AddMeal(
	userID uint,
	mealType string,
	ateAt time.Time,
	items []MealItemRequest,
) (*models.Meal, error) {
	// create the parent meal
	meal := &models.Meal{UserID: userID, Type: mealType, AteAt: ateAt}
	if err := config.DB.Create(meal).Error; err != nil {
		return nil, err
	}

	// for each requested item, Analyze nutrition, then lookup the Label
	for _, it := range items {
		nut, err := s.foodSvc.Analyze(it.FoodID, it.MeasureURI, it.Quantity)
		if err != nil {
			return nil, err
		}
		warnings := utils.AssessFoodSafety(it.FoodID, nut)

		// here’s the only change: instead of using it.FoodID as the label,
		// we do a quick search and pull out the human name
		label := s.lookupLabel(it.FoodID)

		mi := &models.MealItem{
			MealID:     meal.ID,
			FoodID:     it.FoodID,
			FoodLabel:  label,               // ← human name now
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
		if err := config.DB.Create(mi).Error; err != nil {
			return nil, err
		}
	}

	// reload with items
	var populatedMeal models.Meal
	if err := config.DB.Preload("Items").
		First(&populatedMeal, meal.ID).Error; err != nil {
		return nil, err
	}
	return &populatedMeal, nil
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

func (s *MealService) UpdateMeal(
	userID, mealID uint,
	mealType string,
	ateAt time.Time,
	items []MealItemRequest,
) (*models.Meal, error) {
	// fetch & update the parent meal
	var meal models.Meal
	if err := config.DB.
		Where("id = ? AND user_id = ?", mealID, userID).
		First(&meal).Error; err != nil {
		return nil, err
	}
	meal.Type = mealType
	meal.AteAt = ateAt
	if err := config.DB.Save(&meal).Error; err != nil {
		return nil, err
	}

	// delete old items
	if err := config.DB.
		Where("meal_id = ?", meal.ID).
		Delete(&models.MealItem{}).Error; err != nil {
		return nil, err
	}

	// re-create new items, again using lookupLabel
	for _, it := range items {
		nut, err := s.foodSvc.Analyze(it.FoodID, it.MeasureURI, it.Quantity)
		if err != nil {
			return nil, err
		}
		warnings := utils.AssessFoodSafety(it.FoodID, nut)
		label := s.lookupLabel(it.FoodID)

		mi := &models.MealItem{
			MealID:     meal.ID,
			FoodID:     it.FoodID,
			FoodLabel:  label,
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
		if err := config.DB.Create(mi).Error; err != nil {
			return nil, err
		}
	}

	// reload
	var updated models.Meal
	if err := config.DB.
		Preload("Items").
		First(&updated, meal.ID).Error; err != nil {
		return nil, err
	}
	return &updated, nil
}

func (s *MealService) DeleteMeal(userID, mealID uint) error {
	if err := config.DB.
		Where("meal_id = ?", mealID).
		Delete(&models.MealItem{}).Error; err != nil {
		return err
	}
	return config.DB.
		Where("id = ? AND user_id = ?", mealID, userID).
		Delete(&models.Meal{}).Error
}

func (s *MealService) GetMeal(userID, mealID uint) (*models.Meal, error) {
	var meal models.Meal
	err := config.DB.
		Preload("Items").
		Where("id = ? AND user_id = ?", mealID, userID).
		First(&meal).Error
	if err != nil {
		return nil, err  // could be ErrRecordNotFound
	}
	return &meal, nil
}

func (s *MealService) ListMealsByDateRange(userID uint, from, to time.Time) ([]models.Meal, error) {
    var meals []models.Meal
    err := config.DB.
        Preload("Items").
        Where("user_id = ? AND ate_at >= ? AND ate_at < ?", userID, from, to).
        Order("ate_at DESC").
        Find(&meals).Error
    return meals, err
}

func (s *MealService) ListRecentMeals(userID uint, limit int) ([]models.Meal, error) {
	if limit <= 0 {
		limit = 3
	}
	var meals []models.Meal
	q := config.DB.
		Preload("Items").
		Where("user_id = ?", userID).
		Order("ate_at DESC").
		Limit(limit)

	err := q.Find(&meals).Error
	return meals, err
}

// “Recent meal items” (flat list) – handy for a simple card UI
type RecentMealItem struct {
	ID        uint      `json:"id"`
	MealID    uint      `json:"meal_id"`
	FoodLabel string    `json:"food_label"`
	Calories  float64   `json:"calories"`
	Safe      bool      `json:"safe"`
	AteAt     time.Time `json:"ate_at"`
}

func (s *MealService) ListRecentMealItems(userID uint, limit int) ([]RecentMealItem, error) {
	if limit <= 0 {
		limit = 3
	}
	var rows []RecentMealItem
	err := config.DB.
		Table("meal_items").
		Select("meal_items.id, meal_items.meal_id, meal_items.food_label, meal_items.calories, meal_items.safe, meals.ate_at").
		Joins("JOIN meals ON meals.id = meal_items.meal_id").
		Where("meals.user_id = ?", userID).
		Order("meals.ate_at DESC, meal_items.created_at DESC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}



func (s *MealService) ListMealsWithWarnings(userID uint, from, to *time.Time) ([]MealWarnings, error) {
	var meals []models.Meal
	q := config.DB.
		Where("user_id = ?", userID).
		// preload only items that are unsafe OR have non-empty warnings
		Preload("Items", "safe = ? OR warnings <> ''", false).
		Order("ate_at DESC")

	if from != nil && to != nil {
		q = q.Where("ate_at >= ? AND ate_at < ?", *from, *to)
	}

	if err := q.Find(&meals).Error; err != nil {
		return nil, err
	}

	out := make([]MealWarnings, 0, len(meals))
	for _, m := range meals {
		mw := MealWarnings{
			MealID:   m.ID,
			Type:     m.Type,
			AteAt:    m.AteAt,
			MealSafe: true, // assume safe until we see an unsafe item
		}

		for _, it := range m.Items {
			// (Preload already filtered, but double-check just in case)
			if !it.Safe || strings.TrimSpace(it.Warnings) != "" {
				mw.Warnings = append(mw.Warnings, ItemWarning{
					MealItemID: it.ID,
					FoodLabel:  it.FoodLabel,
					Safe:       it.Safe,
					Warnings:   it.Warnings,
					Calories:   it.Calories,
				})
				if !it.Safe {
					mw.MealSafe = false
				}
			}
		}

		out = append(out, mw)
	}
	return out, nil
}

// Optional: single-meal variant
func (s *MealService) GetMealWarnings(userID, mealID uint) (*MealWarnings, error) {
	var meal models.Meal
	if err := config.DB.
		Where("id = ? AND user_id = ?", mealID, userID).
		Preload("Items", "safe = ? OR warnings <> ''", false).
		First(&meal).Error; err != nil {
		return nil, err
	}

	mw := MealWarnings{
		MealID:   meal.ID,
		Type:     meal.Type,
		AteAt:    meal.AteAt,
		MealSafe: true,
	}
	for _, it := range meal.Items {
		if !it.Safe || strings.TrimSpace(it.Warnings) != "" {
			mw.Warnings = append(mw.Warnings, ItemWarning{
				MealItemID: it.ID,
				FoodLabel:  it.FoodLabel,
				Safe:       it.Safe,
				Warnings:   it.Warnings,
				Calories:   it.Calories,
			})
			if !it.Safe {
				mw.MealSafe = false
			}
		}
	}
	return &mw, nil
}
