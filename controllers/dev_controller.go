// controllers/dev_controller.go
package controllers

import (
	"net/http"

	"backend/services"
	"github.com/gin-gonic/gin"
)

type DevController struct {
	Push *services.PushService
}

func NewDevController(p *services.PushService) *DevController {
	return &DevController{Push: p}
}

type pushReq struct {
	Title string            `json:"title"`
	Body  string            `json:"body"`
	Data  map[string]string `json:"data"`
}

func (d *DevController) PushTest(c *gin.Context) {
	// get user id from context (set by your auth middleware)
	v, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	uid, _ := v.(uint)

	var req pushReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// sane defaults for quick tests
	if req.Title == "" {
		req.Title = "Test alert 🔔"
	}
	if req.Body == "" {
		req.Body = "This is only a test."
	}
	if req.Data == nil {
		req.Data = map[string]string{"type": "warning"}
	}

	// PushService currently doesn't return an error
	d.Push.PushToUser(uid, req.Title, req.Body, req.Data)

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
