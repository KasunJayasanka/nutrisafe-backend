package controllers

import (
    "net/http"

    "backend/services"
    "github.com/gin-gonic/gin"
)

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
    var req struct{ ImageBase64 string `json:"image_base64" binding:"required"` }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "invalid body"})
        return
    }
    eda := services.NewEdamamService()
    rek, err := services.NewRekognitionService()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    foodSvc := services.NewFoodService(eda, rek)
    out, err := foodSvc.Recognize(req.ImageBase64)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, out)
}
