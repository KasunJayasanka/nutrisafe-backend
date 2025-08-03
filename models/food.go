package models

import "gorm.io/gorm"

// A catalog entry from Edamam
type FoodItem struct {
    gorm.Model
    EdamamFoodID string `gorm:"type:varchar(255);uniqueIndex;not null"`
    Label        string `gorm:"not null"`
    Category     string
    // Optionally store thumbnail or metadata…
}
