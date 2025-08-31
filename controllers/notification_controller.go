package controllers

import (
    "backend/config"
    "backend/models"
    "github.com/gin-gonic/gin"
    "net/http"
)

type toggleReq struct {
    Enabled bool `json:"enabled"`
}

// POST /user/notifications/toggle
func ToggleNotifications(c *gin.Context) {
    uid := c.GetUint("userID")

    var req toggleReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
        return
    }

    // update all devices for this user (or you could filter by platform if needed)
    if err := config.DB.Model(&models.UserDevice{}).
        Where("user_id = ?", uid).
        Update("enabled", req.Enabled).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "notifications updated",
        "enabled": req.Enabled,
    })
}
