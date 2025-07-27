package models

import "gorm.io/gorm"

// A catalog entry from Edamam
type FoodItem struct {
    gorm.Model
    EdamamFoodID string `gorm:"uniqueIndex;not null"` // e.g. "food_bnbh4ycaqj9as0ahtjwyxbmz4eqx"
    Label        string `gorm:"not null"`
    Category     string
    // Optionally store thumbnail or metadata…
}
