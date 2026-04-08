package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/spaceballone/backend/internal/models"
	sshmanager "github.com/spaceballone/backend/internal/ssh"
	"github.com/spaceballone/backend/internal/terminal"
	"github.com/spaceballone/backend/internal/ws"
	"gorm.io/gorm"
)

// SessionHandler holds dependencies for session endpoints.
type SessionHandler struct {
	DB       *gorm.DB
	SSH      *sshmanager.Manager
	Terminal *terminal.Manager
	Hub      *ws.Hub
}

type createSessionRequest struct {
	Name string `json:"name,omitempty"`
}

type sessionResponse struct {
	ID           string                `json:"id"`
	ProjectID    string                `json:"project_id"`
	Name         string                `json:"name"`
	Status       string                `json:"status"`
	LastActive   *string               `json:"last_active,omitempty"`
	CreatedAt    string                `json:"created_at"`
	UpdatedAt    string                `json:"updated_at"`
	TerminalTabs []terminalTabResponse `json:"terminal_tabs,omitempty"`
}

type terminalTabResponse struct {
	ID              string `json:"id"`
	SessionID       string `json:"session_id"`
	TmuxWindowIndex int    `json:"tmux_window_index"`
	Name            string `json:"name"`
	CreatedAt       string `json:"created_at"`
}

func toSessionResponse(s *models.Session) sessionResponse {
	resp := sessionResponse{
		ID:        s.ID,
		ProjectID: s.ProjectID,
		Name:      s.Name,
		Status:    s.Status,
		CreatedAt: s.CreatedAt.Format(time.RFC3339),
		UpdatedAt: s.UpdatedAt.Format(time.RFC3339),
	}
	if s.LastActive != nil {
		t := s.LastActive.Format(time.RFC3339)
		resp.LastActive = &t
	}
	for i := range s.TerminalTabs {
		resp.TerminalTabs = append(resp.TerminalTabs, toTerminalTabResponse(&s.TerminalTabs[i]))
	}
	return resp
}

func toTerminalTabResponse(t *models.TerminalTab) terminalTabResponse {
	return terminalTabResponse{
		ID:              t.ID,
		SessionID:       t.SessionID,
		TmuxWindowIndex: t.TmuxWindowIndex,
		Name:            t.Name,
		CreatedAt:       t.CreatedAt.Format(time.RFC3339),
	}
}

// ListSessions handles GET /api/projects/{id}/sessions.
func (h *SessionHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")

	var project models.Project
	if err := h.DB.First(&project, "id = ?", projectID).Error; err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	var sessions []models.Session
	if err := h.DB.Preload("TerminalTabs").Where("project_id = ?", projectID).Find(&sessions).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list sessions")
		return
	}

	resp := make([]sessionResponse, len(sessions))
	for i := range sessions {
		resp[i] = toSessionResponse(&sessions[i])
	}

	writeJSON(w, http.StatusOK, resp)
}

// CreateSession handles POST /api/projects/{id}/sessions.
func (h *SessionHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")

	var project models.Project
	if err := h.DB.First(&project, "id = ?", projectID).Error; err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	var req createSessionRequest
	// Body is optional
	json.NewDecoder(r.Body).Decode(&req)

	// Auto-generate session name
	name := req.Name
	if name == "" {
		var count int64
		h.DB.Model(&models.Session{}).Where("project_id = ?", projectID).Count(&count)
		name = fmt.Sprintf("Session %d", count+1)
	}

	now := time.Now()
	session := models.Session{
		ProjectID:  projectID,
		Name:       name,
		Status:     models.SessionStatusActive,
		LastActive: &now,
	}

	if err := h.DB.Create(&session).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	tmuxName := terminal.SessionName(session.ID)

	// Create tmux session on the remote machine if connected
	if h.SSH != nil && h.SSH.IsConnected(project.MachineID) {
		client, err := h.SSH.GetConnection(project.MachineID)
		if err == nil {
			if err := h.Terminal.CreateTmuxSession(client, tmuxName, project.DirectoryPath); err != nil {
				// Not fatal; tmux will be created on demand
				_ = err
			}
		}
	}

	// Create default terminal tab (window 0)
	tab := models.TerminalTab{
		SessionID:       session.ID,
		TmuxWindowIndex: 0,
		Name:            "Terminal 1",
	}
	if err := h.DB.Create(&tab).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create default terminal tab")
		return
	}

	// Reload with tabs
	h.DB.Preload("TerminalTabs").First(&session, "id = ?", session.ID)

	writeJSON(w, http.StatusCreated, toSessionResponse(&session))
}

// GetSession handles GET /api/sessions/{id}.
func (h *SessionHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var session models.Session
	if err := h.DB.Preload("TerminalTabs").First(&session, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	writeJSON(w, http.StatusOK, toSessionResponse(&session))
}

// UpdateSession handles PUT /api/sessions/{id}.
func (h *SessionHandler) UpdateSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var session models.Session
	if err := h.DB.First(&session, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	var req createSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name != "" {
		session.Name = req.Name
	}

	if err := h.DB.Save(&session).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update session")
		return
	}

	h.DB.Preload("TerminalTabs").First(&session, "id = ?", id)

	writeJSON(w, http.StatusOK, toSessionResponse(&session))
}

// DeleteSession handles DELETE /api/sessions/{id}.
func (h *SessionHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var session models.Session
	if err := h.DB.First(&session, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	// Kill tmux session on remote
	var project models.Project
	if err := h.DB.First(&project, "id = ?", session.ProjectID).Error; err == nil {
		if h.SSH != nil && h.SSH.IsConnected(project.MachineID) {
			client, err := h.SSH.GetConnection(project.MachineID)
			if err == nil {
				_ = h.Terminal.KillTmuxSession(client, terminal.SessionName(session.ID))
			}
		}
	}

	// Cascade: delete terminal tabs
	h.DB.Where("session_id = ?", id).Delete(&models.TerminalTab{})
	h.DB.Delete(&session)

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// CreateTerminal handles POST /api/sessions/{id}/terminals.
func (h *SessionHandler) CreateTerminal(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")

	var session models.Session
	if err := h.DB.First(&session, "id = ?", sessionID).Error; err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	var project models.Project
	if err := h.DB.First(&project, "id = ?", session.ProjectID).Error; err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	// Count existing tabs for naming
	var tabCount int64
	h.DB.Model(&models.TerminalTab{}).Where("session_id = ?", sessionID).Count(&tabCount)
	tabName := fmt.Sprintf("Terminal %d", tabCount+1)

	windowIndex := int(tabCount) // default if tmux not available
	tmuxName := terminal.SessionName(session.ID)

	// Create tmux window on remote if connected
	if h.SSH != nil && h.SSH.IsConnected(project.MachineID) {
		client, err := h.SSH.GetConnection(project.MachineID)
		if err == nil {
			idx, err := h.Terminal.CreateTmuxWindow(client, tmuxName, tabName)
			if err == nil {
				windowIndex = idx
			}
		}
	}

	tab := models.TerminalTab{
		SessionID:       sessionID,
		TmuxWindowIndex: windowIndex,
		Name:            tabName,
	}

	if err := h.DB.Create(&tab).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create terminal tab")
		return
	}

	writeJSON(w, http.StatusCreated, toTerminalTabResponse(&tab))
}

// DeleteTerminal handles DELETE /api/terminals/{id}.
func (h *SessionHandler) DeleteTerminal(w http.ResponseWriter, r *http.Request) {
	tabID := chi.URLParam(r, "id")

	var tab models.TerminalTab
	if err := h.DB.First(&tab, "id = ?", tabID).Error; err != nil {
		writeError(w, http.StatusNotFound, "terminal tab not found")
		return
	}

	var session models.Session
	if err := h.DB.First(&session, "id = ?", tab.SessionID).Error; err == nil {
		var project models.Project
		if err := h.DB.First(&project, "id = ?", session.ProjectID).Error; err == nil {
			if h.SSH != nil && h.SSH.IsConnected(project.MachineID) {
				client, err := h.SSH.GetConnection(project.MachineID)
				if err == nil {
					_ = h.Terminal.KillTmuxWindow(client, terminal.SessionName(session.ID), tab.TmuxWindowIndex)
				}
			}
		}
	}

	h.DB.Delete(&tab)

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// RecoverSessions checks for active DB sessions whose tmux sessions are gone and recreates them.
// Called when SSH reconnects.
func (h *SessionHandler) RecoverSessions(machineID string) {
	if h.SSH == nil || !h.SSH.IsConnected(machineID) {
		return
	}

	client, err := h.SSH.GetConnection(machineID)
	if err != nil {
		return
	}

	// Get all projects for this machine
	var projects []models.Project
	h.DB.Where("machine_id = ?", machineID).Find(&projects)

	for _, project := range projects {
		var sessions []models.Session
		h.DB.Where("project_id = ? AND status = ?", project.ID, models.SessionStatusActive).Find(&sessions)

		for _, session := range sessions {
			tmuxName := terminal.SessionName(session.ID)
			if !h.Terminal.SessionExists(client, tmuxName) {
				// Recreate the tmux session
				if err := h.Terminal.CreateTmuxSession(client, tmuxName, project.DirectoryPath); err != nil {
					continue
				}

				// Broadcast recovery notification
				if h.Hub != nil {
					h.Hub.BroadcastJSON(map[string]string{
						"type":       "session_recovered",
						"session_id": session.ID,
						"machine_id": machineID,
					})
				}
			}
		}
	}
}

// SearchHandler holds dependencies for the search endpoint.
type SearchHandler struct {
	DB *gorm.DB
}

// SearchResult represents a single search result.
type SearchResult struct {
	Type     string      `json:"type"`
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Metadata interface{} `json:"metadata,omitempty"`
}

// Search handles GET /api/search?q=...
func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSON(w, http.StatusOK, []SearchResult{})
		return
	}

	like := "%" + query + "%"
	var results []SearchResult

	// Search machines
	var machines []models.Machine
	h.DB.Where("name LIKE ?", like).Find(&machines)
	for _, m := range machines {
		results = append(results, SearchResult{
			Type: "machine",
			ID:   m.ID,
			Name: m.Name,
			Metadata: map[string]interface{}{
				"host":   m.Host,
				"status": m.Status,
			},
		})
	}

	// Search projects
	var projects []models.Project
	h.DB.Where("name LIKE ?", like).Find(&projects)
	for _, p := range projects {
		results = append(results, SearchResult{
			Type: "project",
			ID:   p.ID,
			Name: p.Name,
			Metadata: map[string]interface{}{
				"machine_id":     p.MachineID,
				"directory_path": p.DirectoryPath,
			},
		})
	}

	// Search sessions
	var sessions []models.Session
	h.DB.Where("name LIKE ?", like).Find(&sessions)
	for _, s := range sessions {
		results = append(results, SearchResult{
			Type: "session",
			ID:   s.ID,
			Name: s.Name,
			Metadata: map[string]interface{}{
				"project_id": s.ProjectID,
				"status":     s.Status,
			},
		})
	}

	if results == nil {
		results = []SearchResult{}
	}

	writeJSON(w, http.StatusOK, results)
}
