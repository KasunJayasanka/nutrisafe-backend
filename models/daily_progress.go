package models

import (
    "gorm.io/gorm"
    "time"
)

type DailyProgress struct {
    gorm.Model
    UserID    uint      `gorm:"index;not null"`
    Date      time.Time `gorm:"index;not null"`

    Calories  float64
    Protein   float64
    Carbs     float64
    Fat       float64
    Sodium    float64
    Sugar     float64
    Hydration float64
    Exercise  float64
}
