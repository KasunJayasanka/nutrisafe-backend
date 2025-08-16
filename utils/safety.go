package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
)

type AssessmentContext struct {
	AgeYears      int     
	Sex           string  
	CalorieTarget float64 
	IsBeverage    bool    
	EnableLLM     bool    
}

// WarningSeverity categorizes how serious the flag is.
type WarningSeverity string

const (
	Info    WarningSeverity = "info"
	Caution WarningSeverity = "caution"
	High    WarningSeverity = "high"
)

// Warning is a structured finding you can show in your API.
type Warning struct {
	Code           string          `json:"code"`
	Severity       WarningSeverity `json:"severity"`
	Message        string          `json:"message"`
	Metric         string          `json:"metric,omitempty"` // e.g., "added_sugar_%_of_item_kcal"
	Value          float64         `json:"value,omitempty"`  // numeric value of metric
	Limit          float64         `json:"limit,omitempty"`  // numeric limit used (if any)
	PercentOfLimit float64         `json:"percent_of_limit,omitempty"`
	Reference      string          `json:"reference,omitempty"` // short DGA cite string for UI
}

// AssessFoodSafety keeps the original simple signature.
func AssessFoodSafety(foodName string, nutrients map[string]float64) []string {
	ws := AssessFoodSafetyDGA(foodName, nutrients, AssessmentContext{})
	out := make([]string, 0, len(ws))
	for _, w := range ws {
		out = append(out, w.Message)
	}
	return out
}

// AssessFoodSafetyDGA applies DGA-aligned rule-based checks and (optionally) your LLM allergen check.
func AssessFoodSafetyDGA(foodName string, nutrients map[string]float64, ctx AssessmentContext) []Warning {
	warnings := []Warning{}

	// --- Extract nutrient inputs (common keys from Edamam & friends) ---
	kcal := pick(nutrients, "ENERC_KCAL", "Energy", "kcal")                         // kcal
	addedSugarG := pick(nutrients, "SUGAR.added", "SUGAR_ADDED", "Sugar.added")     // grams
	totalSugarG := pick(nutrients, "SUGAR", "Sugar")                                // grams
	satFatG := pick(nutrients, "FASAT", "FattyAcids,Saturated", "FAT_SAT")          // grams
	transFatG := pick(nutrients, "FATRN", "TransFattyAcids", "FAT_TRANS")           // grams
	sodiumMg := pick(nutrients, "NA", "SODIUM", "Na")                               // mg
	caffeineMg := pick(nutrients, "CAFFEINE", "Caffeine", "CAFNE")                  // mg (rarely provided) // kept for future use

	// --- Life-stage constants (DGA CDRR sodium by age) ---
	sodLimit := sodiumLimitByAge(ctx.AgeYears) // mg per day (CDRR)

	// -----------------------------
	// 1) Added sugars
	// -----------------------------
	// DGA: <10% of calories per day starting at age 2; avoid added sugars for <2.
	// Also: if a food’s added sugars exceed ~10% of its calories, it’s hard to fit within a healthy pattern.
	if ctx.AgeYears > 0 && ctx.AgeYears < 2 {
		if addedSugarG > 0 {
			warnings = append(warnings, Warning{
				Code:      "added_sugars_infants",
				Severity:  High,
				Message:   "For children under 2 years, avoid added sugars.",
				Metric:    "added_sugar_g",
				Value:     round2(addedSugarG),
				Reference: dgaRef("Ch.1, p.19 (limits)"),
			})
		}
	} else {
		// Age 2+ (or unknown): apply the 10%-of-item-calorie screen.
		// Prefer added sugars; fall back to total sugars only if added not available (with gentler severity).
		if kcal > 0 {
			if addedSugarG > 0 {
				addedPctOfItemKcal := (addedSugarG * 4.0) / kcal // fraction (0..1)
				if addedPctOfItemKcal >= 0.10 {
					warnings = append(warnings, Warning{
						Code:      "added_sugars_high",
						Severity:  High,
						Message:   fmt.Sprintf("High added sugars for this item (%.0f%% of its calories).", addedPctOfItemKcal*100),
						Metric:    "added_sugar_%_of_item_kcal",
						Value:     round2(addedPctOfItemKcal * 100),
						Limit:     10,
						Reference: dgaRef("Ch.1, p.41–42 (limits)"),
					})
				}
			} else if totalSugarG > 0 {
				totalPctOfItemKcal := (totalSugarG * 4.0) / kcal
				if totalPctOfItemKcal >= 0.10 {
					warnings = append(warnings, Warning{
						Code:      "total_sugars_proxy_high",
						Severity:  Caution,
						Message:   fmt.Sprintf("Likely high in added sugars (total sugars are %.0f%% of item calories; added sugar not reported).", totalPctOfItemKcal*100),
						Metric:    "total_sugar_%_of_item_kcal",
						Value:     round2(totalPctOfItemKcal * 100),
						Limit:     10,
						Reference: dgaRef("Ch.1, p.41–42 (limits)"),
					})
				}
			}
		}

		// Optional nudge for sugar-sweetened beverages
		if ctx.IsBeverage && (addedSugarG >= 10 || totalSugarG >= 15) && kcal >= 50 {
			warnings = append(warnings, Warning{
				Code:      "ssb_nudge",
				Severity:  Info,
				Message:   "Sugar-sweetened beverages are a major source of added sugars—consider lower-sugar options.",
				Reference: dgaRef("Ch.1, p.42 (sources/strategies)"),
			})
		}
	}

	// -----------------------------
	// 2) Saturated fat
	// -----------------------------
	// DGA: <10% of calories per day (age 2+).
	if kcal > 0 && satFatG > 0 {
		satPctOfItemKcal := (satFatG * 9.0) / kcal
		if satPctOfItemKcal >= 0.10 {
			warnings = append(warnings, Warning{
				Code:      "sat_fat_high",
				Severity:  High,
				Message:   fmt.Sprintf("High saturated fat for this item (%.0f%% of its calories).", satPctOfItemKcal*100),
				Metric:    "saturated_fat_%_of_item_kcal",
				Value:     round2(satPctOfItemKcal * 100),
				Limit:     10,
				Reference: dgaRef("Ch.1, p.44–46 (limit & swaps)"),
			})
		}
	}

	// -----------------------------
	// 3) Sodium
	// -----------------------------
	// DGA: sodium ≤2,300 mg/day for adults; lower CDRRs for children.
	// Label convention: ≥20% DV per serving is “high”.
	if sodiumMg > 0 && sodLimit > 0 {
		percent := sodiumMg / sodLimit // fraction of daily limit in ONE serving
		if percent >= 0.40 {
			warnings = append(warnings, Warning{
				Code:           "sodium_very_high",
				Severity:       High,
				Message:        fmt.Sprintf("Very high sodium for one serving (%.0f%% of the daily limit).", percent*100),
				Metric:         "sodium_%_of_daily_limit_per_serving",
				Value:          round2(percent * 100),
				Limit:          100,
				PercentOfLimit: round2(percent * 100),
				Reference:      dgaRef("Ch.1, p.47 (CDRR)"),
			})
		} else if percent >= 0.20 {
			warnings = append(warnings, Warning{
				Code:           "sodium_high",
				Severity:       Caution,
				Message:        fmt.Sprintf("High sodium for one serving (%.0f%% of the daily limit).", percent*100),
				Metric:         "sodium_%_of_daily_limit_per_serving",
				Value:          round2(percent * 100),
				Limit:          100,
				PercentOfLimit: round2(percent * 100),
				Reference:      dgaRef("Ch.1, p.47 (CDRR) + Label 20%% high"),
			})
		}
	}

	// -----------------------------
	// 4) Trans fat (keep as low as possible)
	// -----------------------------
	if transFatG > 0.0 {
		severity := Caution
		if transFatG >= 0.5 {
			severity = High
		}
		warnings = append(warnings, Warning{
			Code:      "trans_fat_present",
			Severity:  severity,
			Message:   fmt.Sprintf("Contains trans fat (%.2fg); keep intake as low as possible."),
			Metric:    "trans_fat_g",
			Value:     round2(transFatG),
			Reference: dgaRef("Ch.1, p.45 (trans fat note)"),
		})
	}

	// -----------------------------
	// 5) Optional: keep your allergen/preservative LLM check
	// -----------------------------
	if ctx.EnableLLM && os.Getenv("HUGGINGFACE_TOKEN") != "" {
		llmWarns, err := llmSafetyCheck(foodName)
		if err == nil {
			for _, msg := range llmWarns {
				warnings = append(warnings, Warning{
					Code:     "llm_flag",
					Severity: Info,
					Message:  msg,
				})
			}
		}
	}

	_ = caffeineMg // currently unused (intentionally)

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

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func dgaRef(where string) string {
	return "Dietary Guidelines for Americans, 2020–2025 — " + where
}

func round2(f float64) float64 {
	return math.Round(f*100) / 100
}

// llmSafetyCheck asks a free HuggingFace LLM to spot allergens/preservatives.
// Returns []string of any items found, or empty if none.
func llmSafetyCheck(foodName string) ([]string, error) {
	token := os.Getenv("HUGGINGFACE_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("missing HUGGINGFACE_TOKEN")
	}

	model := "google/flan-t5-small"
	prompt := fmt.Sprintf(
		"Food: %s\nList any common allergens (e.g., peanuts, shellfish) or harmful preservatives (e.g., BHA, sodium benzoate) it may contain, separated by commas. If none, reply \"none\".",
		foodName,
	)

	payload := map[string]string{"inputs": prompt}
	b, _ := json.Marshal(payload)

	req, _ := http.NewRequest(
		"POST",
		fmt.Sprintf("https://api-inference.huggingface.co/models/%s", model),
		bytes.NewReader(b),
	)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var hfOut []struct{ GeneratedText string `json:"generated_text"` }
	if err := json.NewDecoder(resp.Body).Decode(&hfOut); err != nil {
		return nil, err
	}
	if len(hfOut) == 0 {
		return nil, nil
	}

	answer := strings.TrimSpace(hfOut[0].GeneratedText)
	if strings.EqualFold(answer, "none") {
		return nil, nil
	}

	parts := strings.Split(answer, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, fmt.Sprintf("Contains: %s", p))
		}
	}
	return out, nil
}
