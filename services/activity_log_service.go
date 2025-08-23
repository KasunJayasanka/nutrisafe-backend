package services

import (
	"backend/config"
	"backend/models"
	"time"

	"gorm.io/gorm"
)

func dayStartLocal(t time.Time) time.Time {
	loc := time.Local
	tt := t.In(loc)
	return time.Date(tt.Year(), tt.Month(), tt.Day(), 0, 0, 0, 0, loc)
}

func UpsertDailyActivity(userID uint, hydration, exercise float64) error {
	start := dayStartLocal(time.Now())

	log := models.DailyActivityLog{
		UserID:    userID,
		Date:      start,
		Hydration: hydration,
		Exercise:  exercise,
	}

	// Upsert by (user_id, date @ local midnight)
	return config.DB.
		Where("user_id = ? AND date = ?", userID, start).
		Assign(log).
		FirstOrCreate(&log).Error
}

func GetDailyActivity(userID uint) (hydration, exercise float64, err error) {
	start := dayStartLocal(time.Now())

	var log models.DailyActivityLog
	err = config.DB.
		Where("user_id = ? AND date = ?", userID, start).
		First(&log).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, 0, nil
		}
		return 0, 0, err
	}
	return log.Hydration, log.Exercise, nil
}

func GetDailyActivityByDate(userID uint, date time.Time) (hydration, exercise float64, err error) {
	start := dayStartLocal(date)

	var log models.DailyActivityLog
	err = config.DB.
		Where("user_id = ? AND date = ?", userID, start).
		First(&log).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, 0, nil
		}
		return 0, 0, err
	}
	return log.Hydration, log.Exercise, nil
}