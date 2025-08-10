package models

import (
	"time"
	"gorm.io/gorm"
)

type DailyActivityLog struct {
	gorm.Model
	UserID    uint      `gorm:"index;not null"`
	Date      time.Time `gorm:"index;not null"` // truncate to YYYY-MM-DD
	Hydration float64   // e.g. 4 (glasses)
	Exercise  float64   // e.g. 30 (minutes)
}
