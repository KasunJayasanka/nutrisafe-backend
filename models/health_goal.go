package models

import (
    "gorm.io/gorm"
)

// DailyGoal holds each user’s daily nutrient‐intake targets.
type DailyGoal struct {
    gorm.Model
    UserID   uint    `gorm:"index;not null"`
    Calories float64 // e.g. 2200 kcal
    Protein  float64 // e.g. 120 g
    Carbs    float64 // e.g. 275 g
	Fat      float64 // e.g. 70 g
    Sodium   float64 // e.g. 2300 mg
    Sugar    float64 // e.g. 50 g

	Hydration float64 // e.g. 8 glasses
    Exercise  float64 // e.g. 60 minutes
}
