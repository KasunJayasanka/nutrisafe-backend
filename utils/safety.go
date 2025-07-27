package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// AssessFoodSafety applies both rule‐based thresholds *and* a free LLM check.
// Returns a slice of warnings; empty means “Safe”.
func AssessFoodSafety(foodName string, nutrients map[string]float64) []string {
	warnings := []string{}

	// ─── Rule‐based checks ─────────────────────────────────
	kcal   := nutrients["ENERC_KCAL"] // calories
	sugar  := nutrients["SUGAR"]      // grams
	satFat := nutrients["FAT"]        // grams (approx saturated fat)
	sodium := nutrients["NA"]         // mg

	if kcal > 0 && sugar*4.0 > 0.10*kcal {
		warnings = append(warnings,
			"High sugar (>10% of calories).")
	}
	if kcal > 0 && satFat*9.0 > 0.10*kcal {
		warnings = append(warnings,
			"High saturated fat (>10% of calories).")
	}
	if sodium > 2300 {
		warnings = append(warnings,
			"High sodium (>2300mg).")
	}

	// ─── LLM‐based allergen/preservative check ────────────────
	llmWarns, err := llmSafetyCheck(foodName)
	if err == nil && len(llmWarns) > 0 {
		warnings = append(warnings, llmWarns...)
	}

	return warnings
}

// llmSafetyCheck asks a free HuggingFace LLM to spot allergens/preservatives.
// Returns []string of any items found, or empty if none.
func llmSafetyCheck(foodName string) ([]string, error) {
	token := os.Getenv("HUGGINGFACE_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("missing HUGGINGFACE_TOKEN")
	}

	// Use a lightweight public model
	model := "google/flan-t5-small"
	prompt := fmt.Sprintf(
		"Food: %s\nList any common allergens (e.g., peanuts, shellfish) or harmful preservatives (e.g., BHA, sodium benzoate) it may contain, separated by commas. If none, reply \"none\".",
		foodName,
	)

	// HF inference API expects {"inputs": "..."}
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

	// HF returns [{"generated_text":"..."}]
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

	// split on commas and trim
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
