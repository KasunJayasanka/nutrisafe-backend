package controllers

import (

    "backend/services"
    "backend/models"
    "backend/config"
    "github.com/gin-gonic/gin"
)

func GetRecommendations(c *gin.Context) {
    email := c.GetString("email")
    var u models.User
    config.DB.First(&u, "email=?", email)

    recSvc := services.NewRecService()
    recs, err := recSvc.GetRecs(u.ID)

    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, gin.H{"recommendations": recs})
}
