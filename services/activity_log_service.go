package services

import (
	"time"
	"backend/config"
	"backend/models"
)

func UpsertDailyActivity(userID uint, hydration, exercise float64) error {
	date := time.Now().Truncate(24 * time.Hour)

	var log models.DailyActivityLog
	err := config.DB.Where("user_id = ? AND date = ?", userID, date).First(&log).Error
	if err != nil {
		log = models.DailyActivityLog{
			UserID:    userID,
			Date:      date,
			Hydration: hydration,
			Exercise:  exercise,
		}
		return config.DB.Create(&log).Error
	}

	log.Hydration = hydration
	log.Exercise = exercise
	return config.DB.Save(&log).Error
}

func GetDailyActivity(userID uint) (hydration, exercise float64, err error) {
	date := time.Now().Truncate(24 * time.Hour)
	var log models.DailyActivityLog
	err = config.DB.Where("user_id = ? AND date = ?", userID, date).First(&log).Error
	if err != nil {
		return 0, 0, nil // treat as zero if not found
	}
	return log.Hydration, log.Exercise, nil
}

func GetDailyActivityByDate(userID uint, date time.Time) (hydration float64, exercise float64, err error) {
	var log models.DailyActivityLog

	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local)
	end := start.Add(24 * time.Hour)

	err = config.DB.
		Where("user_id = ? AND date >= ? AND date < ?", userID, start, end).
		First(&log).Error

	if err != nil {
		// If no record found, return 0s with nil error
		return 0, 0, nil
	}

	return log.Hydration, log.Exercise, nil
}