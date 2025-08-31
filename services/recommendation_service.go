package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"backend/config"
	"backend/models"
)

type RecService struct {
	client *http.Client
	token  string
	model  string
}

func NewRecService() *RecService {
	return &RecService{
		client: &http.Client{Timeout: 15 * time.Second}, // give a bit more time
		token:  os.Getenv("HUGGINGFACE_TOKEN"),
		model:  "google/flan-t5-small",
	}
}

// Summarize intake and call HF to get suggestions
func (r *RecService) GetRecs(userID uint) ([]string, error) {
	if r.token == "" {
		return nil, fmt.Errorf("HUGGINGFACE_TOKEN not set")
	}

	// 1) fetch today's meal items
	var items []models.MealItem
	today := time.Now().Format("2006-01-02")
	if err := config.DB.
		Table("meal_items mi").
		Joins("JOIN meals m ON m.id=mi.meal_id").
		Where("m.user_id = ? AND DATE(m.ate_at)=?", userID, today).
		Select("mi.food_label,mi.quantity,mi.calories,mi.protein").
		Scan(&items).Error; err != nil {
		return nil, fmt.Errorf("db error fetching meals: %w", err)
	}

	// 2) build prompt
	var sb bytes.Buffer
	sb.WriteString("Today's meals:\n")
	if len(items) == 0 {
		sb.WriteString("- (no meals logged yet)\n")
	} else {
		for _, it := range items {
			sb.WriteString(fmt.Sprintf(
				"- %s: %.0fg, %.0f kcal, %.0fg protein\n",
				it.FoodLabel, it.Quantity, it.Calories, it.Protein,
			))
		}
	}
	sb.WriteString("\nSuggest 3–5 healthy, practical adjustments or additions focusing on balance, fiber, and reduced added sugars/sodium. Return plain bullet points.")

	// 3) call HF Inference API
	body := map[string]any{
		"inputs": sb.String(),
		// optional generation params for more consistent bullets
		"parameters": map[string]any{
			"max_new_tokens": 128,
			"temperature":    0.2,
		},
	}
	b, _ := json.Marshal(body)

	req, _ := http.NewRequest(
		"POST",
		fmt.Sprintf("https://api-inference.huggingface.co/models/%s", r.model),
		bytes.NewReader(b),
	)
	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("Content-Type", "application/json")
	// Ensure HF loads cold models instead of returning a “loading” error
	req.Header.Set("x-wait-for-model", "true")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hf request error: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read hf response error: %w", err)
	}

	// Non-200 => surface exact HF error body (often JSON with {"error": "..."} or plain text)
	if resp.StatusCode != http.StatusOK {
		// Try to parse {"error": "..."} nicely; fallback to raw
		var hfErr struct{ Error string `json:"error"` }
		if json.Unmarshal(respBytes, &hfErr) == nil && hfErr.Error != "" {
			return nil, fmt.Errorf("hf api error (%d): %s", resp.StatusCode, hfErr.Error)
		}
		return nil, fmt.Errorf("hf api error (%d): %s", resp.StatusCode, string(respBytes))
	}

	// Try to parse the expected text2text format: [{"generated_text":"..."}]
	var hfOut []struct {
		GeneratedText string `json:"generated_text"`
	}
	if err := json.Unmarshal(respBytes, &hfOut); err != nil {
		// If parsing fails, show a helpful message including a snippet of the body
		bodyPreview := string(respBytes)
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200] + "..."
		}
		return nil, fmt.Errorf("decode hf response error: %v | body: %s", err, bodyPreview)
	}
	if len(hfOut) == 0 || strings.TrimSpace(hfOut[0].GeneratedText) == "" {
		return nil, fmt.Errorf("empty recommendations from hf")
	}

	// split lines into bullets
	var recs []string
	for _, line := range strings.Split(hfOut[0].GeneratedText, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// strip common bullets/prefixes
		line = strings.TrimLeft(line, "-•* \t")
		if line != "" {
			recs = append(recs, line)
		}
	}
	return recs, nil
}
