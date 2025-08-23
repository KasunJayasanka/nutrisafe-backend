package models

import "time"

type Alert struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"index"`
	Type      string    `gorm:"size:20"` // "warning" | "info"
	Message   string    `gorm:"type:text"`
	CreatedAt time.Time
}
