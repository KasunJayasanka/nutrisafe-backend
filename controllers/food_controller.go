package controllers

import (
    "net/http"

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
