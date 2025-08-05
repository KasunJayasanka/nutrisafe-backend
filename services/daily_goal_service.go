// services/health_goal_service.go
package services

import (
    "time"
    "errors"

    "backend/config"
    "backend/models"
    "gorm.io/gorm"
)

func GetGoalsAndProgress(userID uint) (*models.DailyGoal, map[string]interface{}, error) {
    var goal models.DailyGoal
    err := config.DB.Where("user_id = ?", userID).First(&goal).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            goal = models.DailyGoal{UserID: userID}
        } else {
            return nil, nil, err
        }
    }

    // build today’s window
    now := time.Now()
    start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
    end := start.Add(24 * time.Hour)

    // inject your existing MealService here...
    edaSvc := NewEdamamService()
    rekSvc, _ := NewRekognitionService()
    foodSvc := NewFoodService(edaSvc, rekSvc)
    mealSvc := NewMealService(foodSvc)

    meals, err := mealSvc.ListMealsByDateRange(userID, start, end)
    if err != nil {
        return &goal, nil, err
    }

    var cals, prot, carbs, fat, sodium, sugar float64
    for _, m := range meals {
        for _, it := range m.Items {
            cals += it.Calories
            prot += it.Protein
            carbs += it.Carbs
            fat += it.Fat
            sodium += it.Sodium
            sugar += it.Sugar
        }
    }

    pct := func(consumed, target float64) float64 {
        if target <= 0 {
            return 0
        }
        p := consumed / target
        if p > 1 {
            return 1
        }
        return p
    }

	hydration, exercise, err := GetDailyActivity(userID)
	if err != nil {
		return &goal, nil, err
	}

	dp := models.DailyProgress{
		UserID:    userID,
		Date:      start, // beginning of the day
		Calories:  cals,
		Protein:   prot,
		Carbs:     carbs,
		Fat:       fat,
		Sodium:    sodium,
		Sugar:     sugar,
		Hydration: hydration,
		Exercise:  exercise,
	}

	config.DB.
		Where("user_id = ? AND date = ?", userID, start).
		Assign(dp).
		FirstOrCreate(&dp)

    progress := map[string]interface{}{
        "calories":  map[string]float64{"consumed": cals,   "goal": goal.Calories,  "percent": pct(cals,   goal.Calories)},
        "protein":   map[string]float64{"consumed": prot,   "goal": goal.Protein,   "percent": pct(prot,   goal.Protein)},
        "carbs":     map[string]float64{"consumed": carbs,  "goal": goal.Carbs,     "percent": pct(carbs,  goal.Carbs)},
        "fat":       map[string]float64{"consumed": fat,    "goal": goal.Fat,       "percent": pct(fat,    goal.Fat)},
        "sodium":    map[string]float64{"consumed": sodium, "goal": goal.Sodium,    "percent": pct(sodium, goal.Sodium)},
        "sugar":     map[string]float64{"consumed": sugar,  "goal": goal.Sugar,     "percent": pct(sugar,  goal.Sugar)},
       "hydration": map[string]float64{"consumed": hydration, "goal": goal.Hydration, "percent": pct(hydration, goal.Hydration)},
       "exercise":  map[string]float64{"consumed": exercise,  "goal": goal.Exercise,  "percent": pct(exercise,  goal.Exercise)},
    }

    return &goal, progress, nil
}

// **Note**: first arg is uint, rest are float64
func UpsertGoals(
    userID uint,
    calories, protein, carbs, fat, sodium, sugar, hydration, exercise float64,
) error {
    var goal models.DailyGoal
    err := config.DB.Where("user_id = ?", userID).First(&goal).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        goal = models.DailyGoal{
            UserID:    userID,
            Calories:  calories,
            Protein:   protein,
            Carbs:     carbs,
            Fat:       fat,
            Sodium:    sodium,
            Sugar:     sugar,
            Hydration: hydration,
            Exercise:  exercise,
        }
        return config.DB.Create(&goal).Error
    }
    if err != nil {
        return err
    }

    // update
    goal.Calories  = calories
    goal.Protein   = protein
    goal.Carbs     = carbs
    goal.Fat       = fat
    goal.Sodium    = sodium
    goal.Sugar     = sugar
    goal.Hydration = hydration
    goal.Exercise  = exercise

    return config.DB.Save(&goal).Error
}


func GetAllDailyProgress(userID uint) ([]models.DailyProgress, error) {
    var logs []models.DailyProgress
    err := config.DB.
        Where("user_id = ?", userID).
        Order("date desc").
        Find(&logs).Error
    return logs, err
}


func GetGoalsAndProgressByDate(userID uint, date time.Time) (*models.DailyGoal, map[string]interface{}, error) {
	var goal models.DailyGoal
	err := config.DB.Where("user_id = ?", userID).First(&goal).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, err
	}

	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local)
	end := start.Add(24 * time.Hour)

	edaSvc := NewEdamamService()
	rekSvc, _ := NewRekognitionService()
	foodSvc := NewFoodService(edaSvc, rekSvc)
	mealSvc := NewMealService(foodSvc)

	meals, err := mealSvc.ListMealsByDateRange(userID, start, end)
	if err != nil {
		return &goal, nil, err
	}

	var cals, prot, carbs, fat, sodium, sugar float64
	for _, m := range meals {
		for _, it := range m.Items {
			cals += it.Calories
			prot += it.Protein
			carbs += it.Carbs
			fat += it.Fat
			sodium += it.Sodium
			sugar += it.Sugar
		}
	}

	hydration, exercise, _ := GetDailyActivityByDate(userID, start) // implement this if needed

	pct := func(consumed, target float64) float64 {
		if target <= 0 {
			return 0
		}
		p := consumed / target
		if p > 1 {
			return 1
		}
		return p
	}

	progress := map[string]interface{}{
		"calories":  map[string]float64{"consumed": cals, "goal": goal.Calories, "percent": pct(cals, goal.Calories)},
		"protein":   map[string]float64{"consumed": prot, "goal": goal.Protein, "percent": pct(prot, goal.Protein)},
		"carbs":     map[string]float64{"consumed": carbs, "goal": goal.Carbs, "percent": pct(carbs, goal.Carbs)},
		"fat":       map[string]float64{"consumed": fat, "goal": goal.Fat, "percent": pct(fat, goal.Fat)},
		"sodium":    map[string]float64{"consumed": sodium, "goal": goal.Sodium, "percent": pct(sodium, goal.Sodium)},
		"sugar":     map[string]float64{"consumed": sugar, "goal": goal.Sugar, "percent": pct(sugar, goal.Sugar)},
		"hydration": map[string]float64{"consumed": hydration, "goal": goal.Hydration, "percent": pct(hydration, goal.Hydration)},
		"exercise":  map[string]float64{"consumed": exercise, "goal": goal.Exercise, "percent": pct(exercise, goal.Exercise)},
	}

	return &goal, progress, nil
}
