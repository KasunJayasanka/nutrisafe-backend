package models

import (
    "gorm.io/gorm"
	"time"

)

type User struct {
    gorm.Model
    Email           string `gorm:"uniqueIndex;not null"`
    Password        string `gorm:"not null"`
    FullName        string
    HealthConditions string
    FitnessGoals    string
    MFAEnabled      bool
    MFACode         string
	ResetToken     string
	ResetTokenExp  time.Time
	ProfilePicture string `json:"profile_picture"`
	Disabled bool `gorm:"default:false"`


}
    