package ws

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/spaceballone/backend/internal/browser"
	"github.com/spaceballone/backend/internal/models"
	sshmanager "github.com/spaceballone/backend/internal/ssh"
	"gorm.io/gorm"
)

// BrowserHandler handles WebSocket browser screencast connections.
type BrowserHandler struct {
	DB      *gorm.DB
	SSH     *sshmanager.Manager
	Browser *browser.Manager
}

// browserInputEvent represents an input event from the frontend.
type browserInputEvent struct {
	Type      string   `json:"type"`
	X         float64  `json:"x,omitempty"`
	Y         float64  `json:"y,omitempty"`
	Button    string   `json:"button,omitempty"`
	Key       string   `json:"key,omitempty"`
	Code      string   `json:"code,omitempty"`
	Text      string   `json:"text,omitempty"`
	Modifiers []string `json:"modifiers,omitempty"`
	DeltaX    float64  `json:"deltaX,omitempty"`
	DeltaY    float64  `json:"deltaY,omitempty"`
	URL       string   `json:"url,omitempty"`
}

// screencastFrameParams represents the CDP Page.screencastFrame event params.
type screencastFrameParams struct {
	Data      string `json:"data"`
	SessionID int    `json:"sessionId"`
}

// HandleBrowserWS handles ws://…/api/ws/browser/{sessionId}.
func (h *BrowserHandler) HandleBrowserWS(w http.ResponseWriter, r *http.Request) {
	if !ValidateWSSession(h.DB, w, r) {
		return
	}

	sessionID := chi.URLParam(r, "sessionId")

	// Look up session -> project -> machine
	var session models.Session
	if err := h.DB.First(&session, "id = ?", sessionID).Error; err != nil {
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}

	var project models.Project
	if err := h.DB.First(&project, "id = ?", session.ProjectID).Error; err != nil {
		http.Error(w, `{"error":"project not found"}`, http.StatusNotFound)
		return
	}

	if h.SSH == nil || !h.SSH.IsConnected(project.MachineID) {
		http.Error(w, `{"error":"machine not connected"}`, http.StatusBadGateway)
		return
	}

	sshClient, err := h.SSH.GetConnection(project.MachineID)
	if err != nil {
		http.Error(w, `{"error":"failed to get SSH connection"}`, http.StatusBadGateway)
		return
	}

	// Set up SSH tunnel to remote CDP port
	localPort, tunnelCloser, err := browser.SetupCDPTunnel(sshClient, 9222)
	if err != nil {
		http.Error(w, `{"error":"failed to set up CDP tunnel"}`, http.StatusInternalServerError)
		return
	}
	defer tunnelCloser()

	// Connect to Browserless CDP via tunnel
	cdpURL := fmt.Sprintf("ws://127.0.0.1:%d", localPort)
	cdp, err := browser.NewCDPClient(cdpURL)
	if err != nil {
		http.Error(w, `{"error":"failed to connect to CDP"}`, http.StatusBadGateway)
		return
	}
	defer cdp.Close()

	// Upgrade frontend connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws/browser: upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Start screencast
	if err := cdp.PageStartScreencast("jpeg", 60, 1280, 720); err != nil {
		log.Printf("ws/browser: failed to start screencast: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"failed to start screencast"}`))
		return
	}

	// Forward CDP events (screencast frames) to the frontend
	go func() {
		for evt := range cdp.Events() {
			if evt.Method == "Page.screencastFrame" {
				var params screencastFrameParams
				if err := json.Unmarshal(evt.Params, &params); err != nil {
					continue
				}

				// Ack the frame
				cdp.PageScreencastFrameAck(params.SessionID)

				// Decode base64 image and send as binary
				imgData, err := base64.StdEncoding.DecodeString(params.Data)
				if err != nil {
					continue
				}

				if err := conn.WriteMessage(websocket.BinaryMessage, imgData); err != nil {
					return
				}
			}
		}
	}()

	// Read input events from frontend and forward to CDP
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("ws/browser: read error: %v", err)
			}
			break
		}

		var evt browserInputEvent
		if err := json.Unmarshal(data, &evt); err != nil {
			continue
		}

		switch evt.Type {
		case "mousemove":
			cdp.InputDispatchMouseEvent("mouseMoved", evt.X, evt.Y, "", 0)
		case "click":
			btn := evt.Button
			if btn == "" {
				btn = "left"
			}
			cdp.InputDispatchMouseEvent("mousePressed", evt.X, evt.Y, btn, 1)
			cdp.InputDispatchMouseEvent("mouseReleased", evt.X, evt.Y, btn, 1)
		case "keydown":
			cdp.InputDispatchKeyEvent("keyDown", evt.Key, evt.Code, evt.Text)
		case "keyup":
			cdp.InputDispatchKeyEvent("keyUp", evt.Key, evt.Code, "")
		case "scroll":
			cdp.InputDispatchMouseEvent("mouseWheel", evt.X, evt.Y, "", 0)
		case "navigate":
			if evt.URL != "" {
				cdp.PageNavigate(evt.URL)
			}
		}
	}
}
