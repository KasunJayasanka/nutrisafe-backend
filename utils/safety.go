package utils

import (
	"fmt"
	"math"
	"strings"

	"backend/models"
)

type AssessmentContext struct {
	AgeYears      int
	Sex           string
	CalorieTarget float64 // if 0, engine will assume 2000 kcal for % of day conversions
}

// WarningSeverity categorizes how serious the flag is.
type WarningSeverity string

const (
	Info    WarningSeverity = "info"
	Caution WarningSeverity = "caution"
	High    WarningSeverity = "high"
)

// Warning is a structured finding you can show in your API / UI.
type Warning struct {
	Code           string          `json:"code"`
	Severity       WarningSeverity `json:"severity"`
	Message        string          `json:"message"`
	Metric         string          `json:"metric,omitempty"`
	Value          float64         `json:"value,omitempty"`
	Limit          float64         `json:"limit,omitempty"`
	PercentOfLimit float64         `json:"percent_of_limit,omitempty"`
	Reference      string          `json:"reference,omitempty"`
}

// -----------------------------
// Context builders (from models)
// -----------------------------

// BuildAssessmentContext creates an AssessmentContext from a User and optional DailyGoal.
// - AgeYears: derived from user.Birthday (0 if missing)
// - Sex: user.Sex (lowercased/trimmed)
// - CalorieTarget: goal.Calories if >0, else 0 (engine will default to 2000 internally)
func BuildAssessmentContext(user *models.User, goal *models.DailyGoal) AssessmentContext {
	age := 0
	if user != nil && !user.Birthday.IsZero() {
		age = CalculateAge(user.Birthday)
	}
	sex := ""
	if user != nil {
		sex = strings.ToLower(strings.TrimSpace(user.Sex))
	}
	var kcalTarget float64
	if goal != nil && goal.Calories > 0 {
		kcalTarget = goal.Calories
	}
	return AssessmentContext{
		AgeYears:      age,
		Sex:           sex,
		CalorieTarget: kcalTarget,
	}
}

// Convenience wrappers if you want to call the engine directly with user/goal.
func AssessFoodSafetyForUser(foodName string, nutrients map[string]float64, user *models.User, goal *models.DailyGoal) []Warning {
	ctx := BuildAssessmentContext(user, goal)
	return AssessFoodSafetyDGA(foodName, nutrients, ctx)
}
func AssessFoodSafetyMessagesForUser(foodName string, nutrients map[string]float64, user *models.User, goal *models.DailyGoal) []string {
	ws := AssessFoodSafetyForUser(foodName, nutrients, user, goal)
	out := make([]string, 0, len(ws))
	for _, w := range ws {
		out = append(out, w.Message)
	}
	return out
}

// -----------------------------
// Main API-compatible helpers
// -----------------------------

// AssessFoodSafety keeps the original simple signature (strings only).
func AssessFoodSafety(foodName string, nutrients map[string]float64) []string {
	ws := AssessFoodSafetyDGA(foodName, nutrients, AssessmentContext{})
	out := make([]string, 0, len(ws))
	for _, w := range ws {
		out = append(out, w.Message)
	}
	return out
}

// AssessFoodSafetyDGA — DGA 2020–2025 aligned, rule-based engine.
// Only emits findings when inputs are present (no “missing” notes).
func AssessFoodSafetyDGA(foodName string, nutrients map[string]float64, ctx AssessmentContext) []Warning {
	warnings := []Warning{}

	// --- Inputs (common keys from Edamam/USDA-like feeds) ---
	kcal := pick(nutrients, "ENERC_KCAL", "Energy", "kcal", "Calories")
	addedSugarG := pick(nutrients, "SUGAR.added", "SUGAR_ADDED", "Sugar.added")
	totalSugarG := pick(nutrients, "SUGAR", "Sugar")
	satFatG := pick(nutrients, "FASAT", "FattyAcids,Saturated", "FAT_SAT")
	transFatG := pick(nutrients, "FATRN", "TransFattyAcids", "FAT_TRANS")
	sodiumMg := pick(nutrients, "NA", "SODIUM", "Na", "Sodium")
	potassiumMg := pick(nutrients, "K", "POTASSIUM", "Potassium")
	fiberG := pick(nutrients, "FIBTG", "Fiber", "Dietary fiber")

	// Optional macros (for AMDR and kcal reconstruction)
	carbG := pick(nutrients, "CHOCDF", "Carbohydrate, by difference", "CARBS", "Carbs")
	protG := pick(nutrients, "PROCNT", "Protein", "PROTEIN")
	fatG := pick(nutrients, "FAT", "Total lipid (fat)", "Fat")
	alcoholG := pick(nutrients, "ALC", "Alcohol")

	// Optional serving weight for energy density
	servingG := pick(nutrients, "SERVING_SIZE_G", "serving_weight_g", "ServingWeightGrams", "weight_g")

	// If calories missing, reconstruct quietly (no user-facing note)
	if kcal <= 0 {
		kcal = energyFromMacros(carbG, protG, fatG, alcoholG)
	}

	// Age-specific sodium daily limit (CDRR)
	sodLimit := sodiumLimitByAge(ctx.AgeYears)

	// Calorie target for %-of-day conversions (added sugars & sat fat)
	kcalTarget := ctx.CalorieTarget
	if kcalTarget <= 0 {
		kcalTarget = 2000
	}
	addedSugarDailyLimitG := (0.10 * kcalTarget) / 4.0 // <10% kcal/day
	satFatDailyLimitG := (0.10 * kcalTarget) / 9.0     // <10% kcal/day

	// ---------------------------------------------------------
	// 1) Added sugars — <10% kcal/day (avoid added sugars <2y)
	// ---------------------------------------------------------
	if ctx.AgeYears > 0 && ctx.AgeYears < 2 {
		if addedSugarG > 0 {
			warnings = append(warnings, Warning{
				Code:      "added_sugars_infants",
				Severity:  High,
				Message:   "Under age 2: avoid added sugars.",
				Metric:    "added_sugar_g",
				Value:     round2(addedSugarG),
				Reference: dgaRef("Added sugars: avoid for <2y"),
			})
		}
	} else {
		// % of item kcal screen (prefer added; use total as proxy if added missing)
		if kcal > 0 {
			if addedSugarG > 0 {
				pct := (addedSugarG * 4.0) / kcal
				if pct >= 0.10 {
					warnings = append(warnings, Warning{
						Code:      "added_sugars_high_item",
						Severity:  High,
						Message:   fmt.Sprintf("High added sugars for this item (%.0f%% of its calories).", pct*100),
						Metric:    "added_sugar_%_of_item_kcal",
						Value:     round2(pct * 100),
						Limit:     10,
						Reference: dgaRef("Added sugars ≤10% kcal"),
					})
				}
			} else if totalSugarG > 0 {
				pct := (totalSugarG * 4.0) / kcal
				if pct >= 0.10 {
					warnings = append(warnings, Warning{
						Code:      "total_sugars_proxy_high",
						Severity:  Caution,
						Message:   fmt.Sprintf("High sugars for this item (%.0f%% of its calories) — may include added sugars.", pct*100),
						Metric:    "total_sugar_%_of_item_kcal",
						Value:     round2(pct * 100),
						Limit:     10,
						Reference: dgaRef("Added sugars ≤10% kcal"),
					})
				}
			}
		}
		// Per-serving share of daily limit (only if added sugar reported)
		if addedSugarG > 0 && addedSugarDailyLimitG > 0 {
			share := addedSugarG / addedSugarDailyLimitG
			switch {
			case share >= 0.40:
				warnings = append(warnings, Warning{
					Code:           "added_sugars_very_high_daily_share",
					Severity:       High,
					Message:        fmt.Sprintf("This serving provides ~%.0f%% of the daily added-sugar limit.", share*100),
					Metric:         "added_sugar_%_of_daily_limit",
					Value:          round2(share * 100),
					Limit:          100,
					PercentOfLimit: round2(share * 100),
					Reference:      dgaRef("<10% kcal/day from added sugars"),
				})
			case share >= 0.20:
				warnings = append(warnings, Warning{
					Code:           "added_sugars_high_daily_share",
					Severity:       Caution,
					Message:        fmt.Sprintf("High share of daily added-sugar limit from one serving (~%.0f%%).", share*100),
					Metric:         "added_sugar_%_of_daily_limit",
					Value:          round2(share * 100),
					Limit:          100,
					PercentOfLimit: round2(share * 100),
					Reference:      dgaRef("<10% kcal/day from added sugars"),
				})
			}
		}
	}

	// ---------------------------------------------------------
	// 2) Saturated fat — <10% kcal/day (age ≥2)
	// ---------------------------------------------------------
	if (ctx.AgeYears == 0 || ctx.AgeYears >= 2) && kcal > 0 && satFatG > 0 {
		pct := (satFatG * 9.0) / kcal
		if pct >= 0.10 {
			warnings = append(warnings, Warning{
				Code:      "sat_fat_high_item",
				Severity:  High,
				Message:   fmt.Sprintf("High saturated fat for this item (%.0f%% of its calories).", pct*100),
				Metric:    "saturated_fat_%_of_item_kcal",
				Value:     round2(pct * 100),
				Limit:     10,
				Reference: dgaRef("Saturated fat ≤10% kcal"),
			})
		}
	}
	// Per-serving share of daily limit (if sat fat reported)
	if satFatG > 0 && satFatDailyLimitG > 0 {
		share := satFatG / satFatDailyLimitG
		if share >= 0.40 {
			warnings = append(warnings, Warning{
				Code:           "sat_fat_very_high_daily_share",
				Severity:       High,
				Message:        fmt.Sprintf("This serving provides ~%.0f%% of the daily saturated-fat limit.", share*100),
				Metric:         "sat_fat_%_of_daily_limit",
				Value:          round2(share * 100),
				Limit:          100,
				PercentOfLimit: round2(share * 100),
				Reference:      dgaRef("<10% kcal/day from saturated fat"),
			})
		} else if share >= 0.20 {
			warnings = append(warnings, Warning{
				Code:           "sat_fat_high_daily_share",
				Severity:       Caution,
				Message:        fmt.Sprintf("High share of daily saturated-fat limit from one serving (~%.0f%%).", share*100),
				Metric:         "sat_fat_%_of_daily_limit",
				Value:          round2(share * 100),
				Limit:          100,
				PercentOfLimit: round2(share * 100),
				Reference:      dgaRef("<10% kcal/day from saturated fat"),
			})
		}
	}
	// Heuristic nudge when sat fat missing but name suggests high-sat sources
	if satFatG <= 0 && looksHighSatSource(strings.ToLower(foodName)) {
		warnings = append(warnings, Warning{
			Code:      "satfat_source_heuristic",
			Severity:  Info,
			Message:   "Likely high in saturated fat (e.g., butter/cream/fatty meats)—consider leaner cuts or plant oils.",
			Reference: dgaRef("Shift from saturated to unsaturated fats"),
		})
	}

	// ---------------------------------------------------------
	// 3) Sodium — age-aware CDRR; “high” gates ~20%/40% of day
	// ---------------------------------------------------------
	if sodiumMg > 0 && sodLimit > 0 {
		share := sodiumMg / sodLimit
		if share >= 0.40 {
			warnings = append(warnings, Warning{
				Code:           "sodium_very_high",
				Severity:       High,
				Message:        fmt.Sprintf("Very high sodium for one serving (≈%.0f%% of the daily limit).", share*100),
				Metric:         "sodium_%_of_daily_limit_per_serving",
				Value:          round2(share * 100),
				Limit:          100,
				PercentOfLimit: round2(share * 100),
				Reference:      dgaRef("Limit sodium (CDRR)"),
			})
		} else if share >= 0.20 {
			warnings = append(warnings, Warning{
				Code:           "sodium_high",
				Severity:       Caution,
				Message:        fmt.Sprintf("High sodium for one serving (≈%.0f%% of the daily limit).", share*100),
				Metric:         "sodium_%_of_daily_limit_per_serving",
				Value:          round2(share * 100),
				Limit:          100,
				PercentOfLimit: round2(share * 100),
				Reference:      dgaRef("Limit sodium (CDRR)"),
			})
		}
		// Sodium density (mg/100 kcal) — higher density is harder to fit
		if kcal > 0 {
			naPer100kcal := (sodiumMg / kcal) * 100.0
			if naPer100kcal >= 400 {
				warnings = append(warnings, Warning{
					Code:      "sodium_dense",
					Severity:  Info,
					Message:   "High sodium density relative to calories — consider lower-sodium alternatives.",
					Metric:    "sodium_mg_per_100kcal",
					Value:     round2(naPer100kcal),
					Reference: dgaRef("Reduce sodium; choose lower-sodium options"),
				})
			}
		}
	}
	// Sodium–potassium balance (if both present)
	if sodiumMg > 0 && potassiumMg > 0 {
		ratio := sodiumMg / potassiumMg
		if ratio > 1.5 {
			warnings = append(warnings, Warning{
				Code:      "sodium_potassium_ratio_high",
				Severity:  Info,
				Message:   "Higher sodium relative to potassium — add potassium-rich foods (fruits, vegetables, legumes).",
				Metric:    "na_to_k_ratio",
				Value:     round2(ratio),
				Reference: dgaRef("Shift to potassium-rich foods while reducing sodium"),
			})
		}
	}

	// ---------------------------------------------------------
	// 4) Trans fat — keep as low as possible
	// ---------------------------------------------------------
	if transFatG > 0 {
		severity := Caution
		if transFatG >= 0.5 {
			severity = High
		}
		warnings = append(warnings, Warning{
			Code:      "trans_fat_present",
			Severity:  severity,
			Message:   fmt.Sprintf("Contains trans fat (%.2fg); keep intake as low as possible.", transFatG),
			Metric:    "trans_fat_g",
			Value:     round2(transFatG),
			Reference: dgaRef("Avoid trans fat"),
		})
	}

	// ---------------------------------------------------------
	// 5) AMDR (macronutrient distribution) — per item macro kcal
	// ---------------------------------------------------------
	if kcal > 0 && (carbG > 0 || protG > 0 || fatG > 0) {
		totalFromMacros := 4*carbG + 4*protG + 9*fatG
		if totalFromMacros > 0 {
			cPct := (4 * carbG) / totalFromMacros
			pPct := (4 * protG) / totalFromMacros
			fPct := (9 * fatG) / totalFromMacros

			if cPct < 0.45 || cPct > 0.65 {
				warnings = append(warnings, Warning{
					Code:      "amdr_carbs_out_of_range",
					Severity:  Info,
					Message:   fmt.Sprintf("Carbohydrates ~%.0f%% of macro calories (AMDR 45–65%%).", cPct*100),
					Metric:    "carb_%_of_macro_kcal",
					Value:     round2(cPct * 100),
					Reference: dgaRef("AMDR: Carbs 45–65% kcal"),
				})
			}
			if pPct < 0.10 || pPct > 0.35 {
				warnings = append(warnings, Warning{
					Code:      "amdr_protein_out_of_range",
					Severity:  Info,
					Message:   fmt.Sprintf("Protein ~%.0f%% of macro calories (AMDR 10–35%%).", pPct*100),
					Metric:    "protein_%_of_macro_kcal",
					Value:     round2(pPct * 100),
					Reference: dgaRef("AMDR: Protein 10–35% kcal"),
				})
			}
			if fPct < 0.20 || fPct > 0.35 {
				warnings = append(warnings, Warning{
					Code:      "amdr_fat_out_of_range",
					Severity:  Info,
					Message:   fmt.Sprintf("Fat ~%.0f%% of macro calories (AMDR 20–35%%).", fPct*100),
					Metric:    "fat_%_of_macro_kcal",
					Value:     round2(fPct * 100),
					Reference: dgaRef("AMDR: Fat 20–35% kcal"),
				})
			}
		}
	}

	// ---------------------------------------------------------
	// 6) Fiber density (nudges for underconsumed dietary fiber)
	// ---------------------------------------------------------
	if kcal > 0 && carbG >= 15 && fiberG > 0 {
		fiberPer100kcal := (fiberG / kcal) * 100.0
		if fiberPer100kcal < 1.0 {
			warnings = append(warnings, Warning{
				Code:      "fiber_low_nudge",
				Severity:  Info,
				Message:   "Low dietary fiber for a carbohydrate food — consider whole grains, fruits, or vegetables.",
				Metric:    "fiber_g_per_100kcal",
				Value:     round2(fiberPer100kcal),
				Reference: dgaRef("Fiber is underconsumed; emphasize fiber-rich foods"),
			})
		} else if fiberPer100kcal >= 2.5 {
			warnings = append(warnings, Warning{
				Code:      "fiber_high_positive",
				Severity:  Info,
				Message:   "Good fiber density — supports a healthy dietary pattern.",
				Metric:    "fiber_g_per_100kcal",
				Value:     round2(fiberPer100kcal),
				Reference: dgaRef("Emphasize fiber-rich foods"),
			})
		}
	}

	// ---------------------------------------------------------
	// 7) Whole vs. refined grains (name heuristics)
	// ---------------------------------------------------------
	lower := strings.ToLower(foodName)
	if isLikelyWholeGrain(lower) {
		warnings = append(warnings, Warning{
			Code:      "whole_grain_positive",
			Severity:  Info,
			Message:   "Whole-grain choice supports fiber and nutrient density.",
			Reference: dgaRef("Make at least half of grains whole"),
		})
	} else if isLikelyRefinedGrain(lower) {
		warnings = append(warnings, Warning{
			Code:      "refined_grain_nudge",
			Severity:  Info,
			Message:   "Refined-grain item — consider swapping for whole-grain options (≥½ of grains should be whole).",
			Reference: dgaRef("Make at least half of grains whole"),
		})
	}

	// ---------------------------------------------------------
	// 8) Energy density (when serving weight is known)
	// ---------------------------------------------------------
	if servingG > 0 && kcal > 0 {
		kcalPer100g := (kcal / servingG) * 100.0
		switch {
		case kcalPer100g >= 275:
			warnings = append(warnings, Warning{
				Code:      "energy_density_very_high",
				Severity:  Info,
				Message:   "Very energy-dense food — mindful portions can help fit it into a healthy pattern.",
				Metric:    "kcal_per_100g",
				Value:     round2(kcalPer100g),
				Reference: dgaRef("Focus on nutrient density; moderate high-energy-density foods"),
			})
		case kcalPer100g >= 150:
			warnings = append(warnings, Warning{
				Code:      "energy_density_high",
				Severity:  Info,
				Message:   "High energy density — balance with lower-calorie, nutrient-dense sides (vegetables/fruits).",
				Metric:    "kcal_per_100g",
				Value:     round2(kcalPer100g),
				Reference: dgaRef("Emphasize nutrient-dense foods"),
			})
		}
	}

	return warnings
}

// -----------------------------
// Helpers
// -----------------------------

func pick(n map[string]float64, keys ...string) float64 {
	for _, k := range keys {
		if v, ok := n[k]; ok {
			return v
		}
		// also try case-insensitive and dot/underscore variants
		lk := strings.ToLower(k)
		for nk, v := range n {
			if strings.EqualFold(nk, k) || strings.EqualFold(nk, lk) {
				return v
			}
			// handle "SUGAR.added" vs "SUGAR_ADDED"
			if strings.EqualFold(strings.ReplaceAll(nk, "_", "."), strings.ReplaceAll(k, "_", ".")) {
				return v
			}
		}
	}
	return 0
}

func energyFromMacros(carbG, protG, fatG, alcoholG float64) float64 {
	if carbG <= 0 && protG <= 0 && fatG <= 0 && alcoholG <= 0 {
		return 0
	}
	return 4.0*carbG + 4.0*protG + 9.0*fatG + 7.0*alcoholG
}

func sodiumLimitByAge(age int) float64 {
	switch {
	case age > 0 && age <= 3:
		return 1200 // mg/day
	case age >= 4 && age <= 8:
		return 1500
	case age >= 9 && age <= 13:
		return 1800
	default:
		return 2300
	}
}

func dgaRef(where string) string {
	return "Dietary Guidelines for Americans, 2020–2025 — " + where
}

func round2(f float64) float64 {
	return math.Round(f*100) / 100
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// Whole/refined grain heuristics
func isLikelyWholeGrain(name string) bool {
	return containsAny(name, "whole wheat", "whole-grain", "whole grain", "brown rice", "oat", "oats", "quinoa", "bulgur", "rye", "wholemeal")
}
func isLikelyRefinedGrain(name string) bool {
	return containsAny(name, "white bread", "white rice", "refined flour", "all-purpose flour", "maida", "cake", "pastry", "cracker", "biscuit")
}

// High-sat-fat source heuristics when satFat not reported
func looksHighSatSource(name string) bool {
	return containsAny(name,
		"butter", "ghee", "cream", "cheese", "bacon", "sausage", "shortening",
		"palm oil", "palm kernel", "coconut oil", "lard")
}
