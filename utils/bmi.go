package utils

import "errors"

// CalculateBMI expects height in centimeters and weight in kilograms.
func CalculateBMI(heightCm, weightKg float64) (float64, error) {
	if heightCm <= 0 || weightKg <= 0 {
		return 0, errors.New("height and weight must be positive")
	}
	// Sanity checks to avoid garbage input
	if heightCm < 50 || heightCm > 250 || weightKg < 10 || weightKg > 400 {
		return 0, errors.New("height/weight out of plausible range")
	}

	h := heightCm / 100.0 // to meters
	bmi := weightKg / (h * h)
	return bmi, nil
}

func BMICategory(bmi float64) string {
	switch {
	case bmi < 18.5:
		return "Underweight"
	case bmi < 25.0:
		return "Normal weight"
	case bmi < 30.0:
		return "Overweight"
	case bmi < 35.0:
		return "Obesity class I"
	case bmi < 40.0:
		return "Obesity class II"
	default:
		return "Obesity class III"
	}
}
