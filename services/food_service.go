package services

import (
    "backend/models"
    "fmt"
)

type FoodService struct {
    eda *EdamamService
    rek *RekognitionService
}

type FoodInfo struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Category string `json:"category"`
}

type NutritionPreview struct {
	Food       FoodInfo           `json:"food"`
	MeasureURI string             `json:"measure_uri"`
	Quantity   float64            `json:"quantity"`
	Nutrients  map[string]float64 `json:"nutrients"`
	Summary    struct {
		Calories float64 `json:"calories"`
		Protein  float64 `json:"protein"`
		Carbs    float64 `json:"carbs"`
		Fat      float64 `json:"fat"`
		Sodium   float64 `json:"sodium"`
		Sugar    float64 `json:"sugar"`
	} `json:"summary"`
}

func NewFoodService(eda *EdamamService, rek *RekognitionService) *FoodService {
    return &FoodService{eda: eda, rek: rek}
}

// Search manually
func (s *FoodService) Search(query string) ([]models.FoodItem, error) {
    return s.eda.SearchFoods(query)
}

// Recognize via image → returns top hint list
func (s *FoodService) Recognize(base64Img string) ([]models.FoodItem, error) {
    labels, err := s.rek.RecognizeLabels(base64Img)
    if err != nil {
        return nil, err
    }
    // pick first label
    if len(labels) == 0 {
        return nil, fmt.Errorf("no labels detected")
    }
    return s.eda.SearchFoods(labels[0])
}

// Analyze & upsert nutrition+meal item fields
func (s *FoodService) Analyze(foodID, measureURI string, qty float64) (map[string]float64, error) {
    return s.eda.AnalyzeFood(foodID, measureURI, qty)
}

func (s *FoodService) AnalyzePreview(foodID, measureURI string, qty float64) (*NutritionPreview, error) {
    if foodID == "" || measureURI == "" || qty <= 0 {
        return nil, fmt.Errorf("food_id, measure_uri and positive quantity are required")
    }

    // Single call: nutrients + basic food info
    nut, fi, err := s.eda.AnalyzeFoodWithInfo(foodID, measureURI, qty)
    if err != nil {
        return nil, err
    }

    info := FoodInfo{ID: foodID}
    if fi != nil {
        if fi.Label != "" {
            info.Label = fi.Label
        }
        if fi.Category != "" {
            info.Category = fi.Category
        }
    }

    res := &NutritionPreview{
        Food:       info,
        MeasureURI: measureURI,
        Quantity:   qty,
        Nutrients:  nut,
    }

    // Safe even if some keys are missing
    res.Summary.Calories = nut["ENERC_KCAL"]
    res.Summary.Protein  = nut["PROCNT"]
    res.Summary.Carbs    = nut["CHOCDF"]
    res.Summary.Fat      = nut["FAT"]
    res.Summary.Sodium   = nut["NA"]
    res.Summary.Sugar    = nut["SUGAR"]

    return res, nil
}
