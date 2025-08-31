package controllers

import (
	"net/http"
	"time"

	"backend/services"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type RealtimeController struct {
	RT *services.RealtimeHub
}

// constructor
func NewRealtimeController(rt *services.RealtimeHub) *RealtimeController {
	return &RealtimeController{RT: rt}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // tighten behind ALB/CloudFront if needed
}

func (rc *RealtimeController) AlertsWS(c *gin.Context) {
	uid := c.GetUint("userID")

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	cl := &services.WSClient{UserID: uid, Conn: conn}
	rc.RT.Register(cl)

	// optional: ping to keep connections alive through some proxies
	go func() {
		t := time.NewTicker(25 * time.Second)
		defer t.Stop()
		for range t.C {
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				rc.RT.Unregister(cl)
				return
			}
		}
	}()

	// read loop ends on client close/error → unregister
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			rc.RT.Unregister(cl)
			return
		}
	}
}
