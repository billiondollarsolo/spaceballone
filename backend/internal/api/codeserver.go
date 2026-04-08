package api

import (
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/spaceballone/backend/internal/codeserver"
	"github.com/spaceballone/backend/internal/models"
	sshmanager "github.com/spaceballone/backend/internal/ssh"
	"gorm.io/gorm"
)

// CodeServerHandler holds dependencies for code-server endpoints.
type CodeServerHandler struct {
	DB         *gorm.DB
	SSH        *sshmanager.Manager
	CodeServer *codeserver.Manager
}

// StartCodeServer handles POST /api/machines/{id}/code-server/start.
func (h *CodeServerHandler) StartCodeServer(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "id")

	_, client, ok := requireConnectedMachine(h.DB, h.SSH, machineID, w)
	if !ok {
		return
	}

	port, err := h.CodeServer.StartCodeServer(client, machineID)
	if err != nil {
		log.Printf("Failed to start code-server on machine %s: %v", machineID, err)
		writeError(w, http.StatusInternalServerError, "Failed to start code-server")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"url": "http://localhost:" + strconv.Itoa(port),
	})
}

// StopCodeServer handles POST /api/machines/{id}/code-server/stop.
func (h *CodeServerHandler) StopCodeServer(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "id")

	_, client, ok := requireConnectedMachine(h.DB, h.SSH, machineID, w)
	if !ok {
		return
	}

	if err := h.CodeServer.StopCodeServer(client, machineID); err != nil {
		log.Printf("Failed to stop code-server on machine %s: %v", machineID, err)
		writeError(w, http.StatusInternalServerError, "Failed to stop code-server")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// CodeServerStatus handles GET /api/machines/{id}/code-server/status.
func (h *CodeServerHandler) CodeServerStatus(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "id")

	var machine models.Machine
	if err := h.DB.First(&machine, "id = ?", machineID).Error; err != nil {
		writeError(w, http.StatusNotFound, "machine not found")
		return
	}

	running := false
	var url *string

	if h.SSH.IsConnected(machineID) {
		client, err := h.SSH.GetConnection(machineID)
		if err == nil {
			running = h.CodeServer.IsCodeServerRunning(client)
		}
	}

	if running {
		u := h.CodeServer.GetTunnelURL(machineID)
		if u != "" {
			url = &u
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"running": running,
		"url":     url,
	})
}

// OpenFolder handles POST /api/machines/{id}/code-server/open?folder=/path.
func (h *CodeServerHandler) OpenFolder(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "id")
	folder := r.URL.Query().Get("folder")

	if folder == "" {
		writeError(w, http.StatusBadRequest, "folder query parameter is required")
		return
	}

	url := h.CodeServer.GetTunnelURL(machineID)
	if url == "" {
		writeError(w, http.StatusBadRequest, "code-server is not running or no tunnel available")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"url": url + "/?folder=" + folder,
	})
}
