package services

import (
    "backend/models"
    "fmt"
)

type FoodService struct {
    eda *EdamamService
    rek *RekognitionService
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
