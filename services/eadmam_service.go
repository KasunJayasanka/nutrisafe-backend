package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
	"backend/models"
)

type EdamamService struct {
    foodAppID, foodAppKey   string
    nutriAppID, nutriAppKey string
    client                  *http.Client
}

// NewEdamamService initializes the EdamamService with credentials and HTTP client
func NewEdamamService() *EdamamService {
    return &EdamamService{
        foodAppID:   os.Getenv("EDAMAM_APP_ID"),
        foodAppKey:  os.Getenv("EDAMAM_APP_KEY"),
        nutriAppID:  os.Getenv("EDAMAM_NUTRI_APP_ID"),
        nutriAppKey: os.Getenv("EDAMAM_NUTRI_APP_KEY"),
        client:      &http.Client{Timeout: 10 * time.Second},
    }
}

// SearchFoods calls the Edamam Food Database API parser endpoint
type foodParserResponse struct {
    Hints []struct {
        Food struct {
            FoodID   string `json:"foodId"`
            Label    string `json:"label"`
            Category string `json:"category"`
        } `json:"food"`
    } `json:"hints"`
}

func (s *EdamamService) SearchFoods(query string) ([]models.FoodItem, error) {
    // Build request URL
    u := fmt.Sprintf(
        "https://api.edamam.com/api/food-database/v2/parser?ingr=%s&app_id=%s&app_key=%s",
        url.QueryEscape(query), s.foodAppID, s.foodAppKey,
    )

    resp, err := s.client.Get(u)
    if err != nil {
        return nil, fmt.Errorf("failed to call Edamam parser: %w", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read Edamam parser response: %w", err)
    }
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("edamam parser API error %d: %s", resp.StatusCode, string(body))
    }

    var pr foodParserResponse
    if err := json.Unmarshal(body, &pr); err != nil {
        return nil, fmt.Errorf("failed to parse Edamam parser JSON: %w", err)
    }

    results := make([]models.FoodItem, 0, len(pr.Hints))
    for _, h := range pr.Hints {
        results = append(results, models.FoodItem{
            EdamamFoodID: h.Food.FoodID,
            Label:        h.Food.Label,
            Category:     h.Food.Category,
        })
    }
    return results, nil
}

// AnalyzeFood calls the Edamam Nutrition Analysis API for a single ingredient
type nutritionResponse struct {
    TotalNutrients map[string]struct {
        Quantity float64 `json:"quantity"`
    } `json:"totalNutrients"`
}

func (s *EdamamService) AnalyzeFood(foodID, measureURI string, qty float64) (map[string]float64, error) {
    // Build request payload
    payload := map[string]interface{}{
        "ingredients": []map[string]interface{}{ {
            "quantity":   qty,
            "measureURI": measureURI,
            "foodId":     foodID,
        }},
    }
    b, err := json.Marshal(payload)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal nutrition payload: %w", err)
    }

    u := fmt.Sprintf(
        "https://api.edamam.com/api/food-database/v2/nutrients?app_id=%s&app_key=%s",
        s.nutriAppID, s.nutriAppKey,
    )
	
    req, err := http.NewRequest("POST", u, bytes.NewReader(b))
    if err != nil {
        return nil, fmt.Errorf("failed to create nutrition request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := s.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to call Edamam nutrition API: %w", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read nutrition response: %w", err)
    }
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("edamam nutrition API error %d: %s", resp.StatusCode, string(body))
    }

    var nr nutritionResponse
    if err := json.Unmarshal(body, &nr); err != nil {
        return nil, fmt.Errorf("failed to parse nutrition JSON: %w", err)
    }

    // Flatten to simple map
    nut := make(map[string]float64, len(nr.TotalNutrients))
    for k, v := range nr.TotalNutrients {
        nut[k] = v.Quantity
    }
    return nut, nil
}


// Put near your other structs:
type nutritionResponseFull struct {
    Ingredients []struct {
        Parsed []struct {
            Food         string `json:"food"`                   // label string
            FoodID       string `json:"foodId"`                 // edamam id
            FoodURI      string `json:"foodURI,omitempty"`      // optional
            FoodCategory string `json:"foodCategory,omitempty"` // optional
        } `json:"parsed"`
    } `json:"ingredients"`
    TotalNutrients map[string]struct {
        Quantity float64 `json:"quantity"`
    } `json:"totalNutrients"`
}

// AnalyzeFoodWithInfo calls the same nutrients endpoint but also extracts
// label/id/category (best-effort) from the response so you can show a preview.
func (s *EdamamService) AnalyzeFoodWithInfo(foodID, measureURI string, qty float64) (map[string]float64, *models.FoodItem, error) {
    // Build request payload
    payload := map[string]interface{}{
        "ingredients": []map[string]interface{}{
            {
                "quantity":   qty,
                "measureURI": measureURI,
                "foodId":     foodID,
            },
        },
    }
    b, err := json.Marshal(payload)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to marshal nutrition payload: %w", err)
    }

    u := fmt.Sprintf(
        "https://api.edamam.com/api/food-database/v2/nutrients?app_id=%s&app_key=%s",
        s.nutriAppID, s.nutriAppKey,
    )

    req, err := http.NewRequest("POST", u, bytes.NewReader(b))
    if err != nil {
        return nil, nil, fmt.Errorf("failed to create nutrition request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := s.client.Do(req)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to call Edamam nutrition API: %w", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to read nutrition response: %w", err)
    }
    if resp.StatusCode != http.StatusOK {
        return nil, nil, fmt.Errorf("edamam nutrition API error %d: %s", resp.StatusCode, string(body))
    }

    var nr nutritionResponseFull
    if err := json.Unmarshal(body, &nr); err != nil {
        return nil, nil, fmt.Errorf("failed to parse nutrition JSON: %w", err)
    }

    // Flatten nutrients
    nut := make(map[string]float64, len(nr.TotalNutrients))
    for k, v := range nr.TotalNutrients {
        nut[k] = v.Quantity
    }

    // Best-effort food info from the first parsed ingredient (if present)
    var info *models.FoodItem
    if len(nr.Ingredients) > 0 && len(nr.Ingredients[0].Parsed) > 0 {
        p := nr.Ingredients[0].Parsed[0]
        info = &models.FoodItem{
            EdamamFoodID: p.FoodID,
            Label:        p.Food,         // "food" is a label string here
            Category:     p.FoodCategory, // may be empty
        }
    }

    return nut, info, nil
}
