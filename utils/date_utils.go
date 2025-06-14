package utils

import "time"

func CalculateAge(birthday time.Time) int {
	now := time.Now()
	age := now.Year() - birthday.Year()
	if now.YearDay() < birthday.YearDay() {
		age--
	}
	return age
}
