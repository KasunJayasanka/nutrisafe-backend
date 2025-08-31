package models

import "time"

type UserDevice struct {
	ID          uint      `gorm:"primaryKey"`
	UserID      uint      `gorm:"index"`
	Platform    string    `gorm:"size:16"` // "android" | "ios"
	TokenHash   string    `gorm:"size:64"`
	EndpointARN string    `gorm:"size:256"`
	Enabled     bool      `gorm:"default:true"`
	UpdatedAt   time.Time
	CreatedAt   time.Time
}
