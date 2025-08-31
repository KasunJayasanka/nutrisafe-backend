package services

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

type WSClient struct {
	UserID uint
	Conn   *websocket.Conn
}

type RealtimeHub struct {
	mu      sync.RWMutex
	clients map[uint]map[*WSClient]struct{}
}

func NewRealtimeHub() *RealtimeHub {
	return &RealtimeHub{clients: make(map[uint]map[*WSClient]struct{})}
}

func (h *RealtimeHub) Register(c *WSClient) {
	h.mu.Lock()
	if h.clients[c.UserID] == nil {
		h.clients[c.UserID] = make(map[*WSClient]struct{})
	}
	h.clients[c.UserID][c] = struct{}{}
	h.mu.Unlock()
}

func (h *RealtimeHub) Unregister(c *WSClient) {
	h.mu.Lock()
	if set := h.clients[c.UserID]; set != nil {
		delete(set, c)
		if len(set) == 0 { delete(h.clients, c.UserID) }
	}
	h.mu.Unlock()
	_ = c.Conn.Close()
}

func (h *RealtimeHub) BroadcastAlert(userID uint, payload any) {
	msg, _ := json.Marshal(payload)
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[userID] {
		_ = c.Conn.WriteMessage(websocket.TextMessage, msg)
	}
}
