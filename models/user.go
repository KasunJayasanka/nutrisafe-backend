package models

import (
    "gorm.io/gorm"
	"time"

)
// lib/models/user.go
type User struct {
    gorm.Model
    UserID           string    `gorm:"uniqueIndex"`
    Email            string    `gorm:"uniqueIndex;not null"`
    Password         string    `gorm:"not null"`
    FirstName        string
    LastName         string

    Birthday         time.Time
    Height           float64
    Weight           float64
    HealthConditions string    // comma-list
    FitnessGoals     string    // comma-list
    ProfilePicture   string    `json:"profile_picture"`
    MFAEnabled       bool

    Sex              string    `json:"sex"`

    Onboarded        bool       `gorm:"default:false"`
    MFACode          string
    ResetToken       string
    ResetTokenExp    time.Time
    Disabled         bool      `gorm:"default:false"`
}