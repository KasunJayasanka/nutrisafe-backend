package services

import (
	"bytes"
	"encoding/json"
	"fmt"
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
        client: &http.Client{Timeout: 5 * time.Second},
        token:  os.Getenv("HUGGINGFACE_TOKEN"),
        model:  "google/flan-t5-small", // or another free model
    }
}

// Summarize intake and call HF to get suggestions
func (r *RecService) GetRecs(userID uint) ([]string, error) {
    // 1) fetch today's meal items
    var items []models.MealItem
    today := time.Now().Format("2006-01-02")
    config.DB.
        Table("meal_items mi").
        Joins("JOIN meals m ON m.id=mi.meal_id").
        Where("m.user_id = ? AND DATE(m.ate_at)=?", userID, today).
        Select("mi.food_label,mi.quantity,mi.calories,mi.protein").
        Scan(&items)

    // 2) build prompt
    var sb bytes.Buffer
    sb.WriteString("Today's meals:\n")
    for _, it := range items {
        sb.WriteString(fmt.Sprintf(
            "- %s: %.0fg, %.0f kcal, %.0fg protein\n",
            it.FoodLabel, it.Quantity, it.Calories, it.Protein,
        ))
    }
    sb.WriteString("\nSuggest healthy adjustments or additions:")

    // 3) call HF Inference API
    body := map[string]string{"inputs": sb.String()}
    b, _ := json.Marshal(body)
    req, _ := http.NewRequest(
        "POST",
        fmt.Sprintf("https://api-inference.huggingface.co/models/%s", r.model),
        bytes.NewReader(b),
    )
    req.Header.Set("Authorization", "Bearer "+r.token)
    req.Header.Set("Content-Type", "application/json")

    resp, err := r.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var hfOut []struct{ GeneratedText string }
    if err := json.NewDecoder(resp.Body).Decode(&hfOut); err != nil {
        return nil, err
    }

    // split lines
    var recs []string
    for _, line := range strings.Split(hfOut[0].GeneratedText, "\n") {
        line = strings.TrimSpace(line)
        if line != "" {
            recs = append(recs, line)
        }
    }
    return recs, nil
}
