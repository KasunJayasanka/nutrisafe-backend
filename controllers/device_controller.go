package controllers

import (
	"net/http"

	"backend/services"
	"github.com/gin-gonic/gin"
)

type DeviceController struct {
	Push *services.PushService
}

// constructor
func NewDeviceController(ps *services.PushService) *DeviceController {
	return &DeviceController{Push: ps}
}

func (dc *DeviceController) Register(c *gin.Context) {
	uid := c.GetUint("userID")

	var req services.RegisterDeviceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dev, err := dc.Push.RegisterDevice(uid, req.Platform, req.Token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"endpoint_arn": dev.EndpointARN})
}
