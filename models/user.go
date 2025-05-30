package models

import (
    "gorm.io/gorm"
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
}
