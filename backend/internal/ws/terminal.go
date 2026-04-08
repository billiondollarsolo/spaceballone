package ws

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/spaceballone/backend/internal/models"
	sshmanager "github.com/spaceballone/backend/internal/ssh"
	"github.com/spaceballone/backend/internal/terminal"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

// TerminalHandler handles WebSocket terminal connections.
type TerminalHandler struct {
	DB       *gorm.DB
	SSH      *sshmanager.Manager
	Terminal *terminal.Manager
}

type resizeMessage struct {
	Type string `json:"type"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

// HandleTerminalWS handles ws://…/api/ws/terminal/{terminalId}.
func (h *TerminalHandler) HandleTerminalWS(w http.ResponseWriter, r *http.Request) {
	if _, ok := ValidateWSSession(h.DB, w, r); !ok {
		return
	}

	terminalID := chi.URLParam(r, "terminalId")

	// Look up terminal tab
	var tab models.TerminalTab
	if err := h.DB.First(&tab, "id = ?", terminalID).Error; err != nil {
		http.Error(w, `{"error":"terminal tab not found"}`, http.StatusNotFound)
		return
	}

	// Look up session
	var session models.Session
	if err := h.DB.First(&session, "id = ?", tab.SessionID).Error; err != nil {
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}

	// Look up project
	var project models.Project
	if err := h.DB.First(&project, "id = ?", session.ProjectID).Error; err != nil {
		http.Error(w, `{"error":"project not found"}`, http.StatusNotFound)
		return
	}

	// Get SSH connection
	if h.SSH == nil || !h.SSH.IsConnected(project.MachineID) {
		http.Error(w, `{"error":"machine not connected"}`, http.StatusBadGateway)
		return
	}

	client, err := h.SSH.GetConnection(project.MachineID)
	if err != nil {
		http.Error(w, `{"error":"failed to get SSH connection"}`, http.StatusBadGateway)
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws/terminal: upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	tmuxName := terminal.SessionName(session.ID)

	sshSession, startAttach, err := terminal.PrepareTmuxAttach(client, tmuxName, tab.TmuxWindowIndex, 80, 24, project.DirectoryPath)
	if err != nil {
		log.Printf("ws/terminal: failed to prepare tmux attach: %v", err)
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"failed to attach to terminal"}`))
		return
	}

	stdin, err := sshSession.StdinPipe()
	if err != nil {
		log.Printf("ws/terminal: failed to get stdin pipe: %v", err)
		sshSession.Close()
		return
	}

	stdout, err := sshSession.StdoutPipe()
	if err != nil {
		log.Printf("ws/terminal: failed to get stdout pipe: %v", err)
		sshSession.Close()
		return
	}

	if err := startAttach(); err != nil {
		log.Printf("ws/terminal: failed to start tmux attach: %v", err)
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"failed to attach to terminal"}`))
		return
	}

	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			stdin.Close()
			sshSession.Close()
		})
	}
	defer cleanup()

	// SSH stdout -> WebSocket
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				if writeErr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); writeErr != nil {
					break
				}
			}
			if err != nil {
				if err != io.EOF {
					log.Printf("ws/terminal: stdout read error: %v", err)
				}
				break
			}
		}
		conn.Close()
	}()

	// WebSocket -> SSH stdin (with resize handling)
	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("ws/terminal: read error: %v", err)
			}
			break
		}

		if msgType == websocket.TextMessage {
			// Check for resize messages
			var resize resizeMessage
			if json.Unmarshal(data, &resize) == nil && resize.Type == "resize" {
				handleResize(sshSession, resize.Cols, resize.Rows)
				continue
			}
		}

		// Forward data to SSH stdin
		if _, err := stdin.Write(data); err != nil {
			log.Printf("ws/terminal: stdin write error: %v", err)
			break
		}
	}
}

// handleResize sends a window change request to the SSH session.
func handleResize(session *ssh.Session, cols, rows int) {
	if cols <= 0 || rows <= 0 {
		return
	}
	// Send window-change request
	if err := session.WindowChange(rows, cols); err != nil {
		log.Printf("ws/terminal: resize failed: %v", err)
	}
}
