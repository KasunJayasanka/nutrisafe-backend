package controllers

import (
	"backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

type DevUploadRequest struct {
	ImageBase64 string `json:"image_base64" binding:"required"`
}

func DevUploadImage(c *gin.Context) {
	var req DevUploadRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Static path for dev
	url, err := utils.UploadBase64ImageToS3(req.ImageBase64, "general/dev-upload")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Upload failed", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}
