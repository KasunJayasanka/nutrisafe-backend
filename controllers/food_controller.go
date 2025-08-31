package controllers

import (
	"net/http"
	"strconv"

	"backend/services"

	"github.com/gin-gonic/gin"
)

type recognizeRequest struct {
    ImageBase64 string `json:"image_base64" binding:"required"`
}


// GET /food/search?q=apple
func SearchFoods(c *gin.Context) {
	eda := services.NewEdamamService()
	rek, err := services.NewRekognitionService()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	foodSvc := services.NewFoodService(eda, rek)
	out, err := foodSvc.Search(c.Query("q"))
    if err != nil {
        c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, out)
}

// POST /food/recognize  { "image_base64": "data:…"}
func RecognizeFood(c *gin.Context) {
    // 1️⃣ Bind the JSON
    var body recognizeRequest
    if err := c.ShouldBindJSON(&body); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // 2️⃣ Initialize your services
    eda := services.NewEdamamService()
    rek, err := services.NewRekognitionService()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    foodSvc := services.NewFoodService(eda, rek)

    // 3️⃣ Recognize
    items, err := foodSvc.Recognize(body.ImageBase64)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // 4️⃣ Return the list of FoodItems
    c.JSON(http.StatusOK, items)
}

type analyzeFoodRequest struct {
	FoodID     string  `json:"food_id" binding:"required"`
	MeasureURI string  `json:"measure_uri" binding:"required"`
	Quantity   float64 `json:"quantity" binding:"required"`
}

// POST /food/analyze
func AnalyzeFood(c *gin.Context) {
	var body analyzeFoodRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	eda := services.NewEdamamService()
	rek, err := services.NewRekognitionService()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	foodSvc := services.NewFoodService(eda, rek)

	out, err := foodSvc.AnalyzePreview(body.FoodID, body.MeasureURI, body.Quantity)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

// GET /food/:id/nutrition?measure_uri=...&quantity=...
func GetFoodNutrition(c *gin.Context) {
	id := c.Param("id")
	measureURI := c.Query("measure_uri")
	qStr := c.Query("quantity")

	if id == "" || measureURI == "" || qStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id, measure_uri and quantity are required"})
		return
	}
	qty, err := strconv.ParseFloat(qStr, 64)
	if err != nil || qty <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "quantity must be a positive number"})
		return
	}

	eda := services.NewEdamamService()
	rek, err := services.NewRekognitionService()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	foodSvc := services.NewFoodService(eda, rek)

	out, err := foodSvc.AnalyzePreview(id, measureURI, qty)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}