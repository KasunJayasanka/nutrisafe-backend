package models

import (
    "time"

    "gorm.io/gorm"
)

// One Meal (breakfast/lunch/…)
type Meal struct {
    gorm.Model
    UserID uint      // FK → users.id
    Type   string    // “Breakfast”|“Lunch”|…
    AteAt  time.Time // timestamp of the meal
    Items  []MealItem
}

// Each MealItem stores the nutrition snapshot & safety flag
type MealItem struct {
    gorm.Model
    MealID     uint
    Meal       Meal

    FoodID     string    `gorm:"type:varchar(255);not null"`
    // Food       FoodItem  `gorm:"foreignKey:FoodID;references:EdamamFoodID"`
    FoodLabel     string          // human label
    Quantity      float64         // e.g. 200
    MeasureURI    string          // e.g. “http://www.edamam.com/ontologies/edamam.owl#Measure_gram”
    Calories      float64
    Protein       float64
    Carbs         float64
    Fat           float64
    Sodium        float64
    Sugar         float64
    // etc. add micro if desired
    Safe          bool            // safety assessment
    Warnings      string          // comma-sep warnings
}
