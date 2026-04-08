// Package ws provides WebSocket handlers for status updates.
package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spaceballone/backend/internal/auth"
	authmw "github.com/spaceballone/backend/internal/middleware"
	"gorm.io/gorm"
)

// checkOrigin validates the Origin header against the configured FRONTEND_URL.
func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}
	return origin == frontendURL
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     checkOrigin,
}

// ValidateWSSession validates the session cookie on a WebSocket HTTP request.
// Returns an error and sends a 401 response if the session is invalid.
func ValidateWSSession(db *gorm.DB, w http.ResponseWriter, r *http.Request) bool {
	cookie, err := r.Cookie(authmw.SessionCookieName)
	if err != nil || cookie.Value == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return false
	}
	if _, err := auth.ValidateSession(db, cookie.Value); err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return false
	}
	return true
}

// StatusMessage represents a machine status change message.
type StatusMessage struct {
	Type      string `json:"type"`
	MachineID string `json:"machine_id"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// Hub maintains the set of active WebSocket clients and broadcasts messages.
type Hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]struct{}
	DB      *gorm.DB
}

// NewHub creates a new WebSocket hub.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[*websocket.Conn]struct{}),
	}
}

// HandleWebSocket upgrades the HTTP connection to WebSocket and registers the client.
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	if h.DB != nil && !ValidateWSSession(h.DB, w, r) {
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws: upgrade failed: %v", err)
		return
	}

	h.mu.Lock()
	h.clients[conn] = struct{}{}
	h.mu.Unlock()

	// Read pump: just consume messages to detect close.
	go func() {
		defer func() {
			h.mu.Lock()
			delete(h.clients, conn)
			h.mu.Unlock()
			conn.Close()
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

// BroadcastJSON sends an arbitrary JSON message to all connected clients.
func (h *Hub) BroadcastJSON(msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("ws: failed to marshal message: %v", err)
		return
	}

	h.mu.RLock()
	clients := make([]*websocket.Conn, 0, len(h.clients))
	for conn := range h.clients {
		clients = append(clients, conn)
	}
	h.mu.RUnlock()

	for _, conn := range clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("ws: failed to write to client: %v", err)
			h.mu.Lock()
			delete(h.clients, conn)
			h.mu.Unlock()
			conn.Close()
		}
	}
}

// BroadcastStatus sends a machine status change to all connected clients.
func (h *Hub) BroadcastStatus(machineID, status string) {
	h.BroadcastJSON(StatusMessage{
		Type:      "machine_status",
		MachineID: machineID,
		Status:    status,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}
