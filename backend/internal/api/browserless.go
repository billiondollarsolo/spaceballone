package api

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/spaceballone/backend/internal/browser"
	"github.com/spaceballone/backend/internal/models"
	sshmanager "github.com/spaceballone/backend/internal/ssh"
	"gorm.io/gorm"
)

// BrowserlessHandler holds dependencies for Browserless endpoints.
type BrowserlessHandler struct {
	DB      *gorm.DB
	SSH     *sshmanager.Manager
	Browser *browser.Manager
}

// StartBrowserless handles POST /api/machines/{id}/browserless/start.
func (h *BrowserlessHandler) StartBrowserless(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "id")

	_, client, ok := requireConnectedMachine(h.DB, h.SSH, machineID, w)
	if !ok {
		return
	}

	if err := h.Browser.StartBrowserless(client, machineID); err != nil {
		log.Printf("Failed to start browserless on machine %s: %v", machineID, err)
		writeError(w, http.StatusInternalServerError, "Failed to start browserless")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// StopBrowserless handles POST /api/machines/{id}/browserless/stop.
func (h *BrowserlessHandler) StopBrowserless(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "id")

	_, client, ok := requireConnectedMachine(h.DB, h.SSH, machineID, w)
	if !ok {
		return
	}

	if err := h.Browser.StopBrowserless(client); err != nil {
		log.Printf("Failed to stop browserless on machine %s: %v", machineID, err)
		writeError(w, http.StatusInternalServerError, "Failed to stop browserless")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// BrowserlessStatus handles GET /api/machines/{id}/browserless/status.
func (h *BrowserlessHandler) BrowserlessStatus(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "id")

	var machine models.Machine
	if err := h.DB.First(&machine, "id = ?", machineID).Error; err != nil {
		writeError(w, http.StatusNotFound, "machine not found")
		return
	}

	running := false
	if h.SSH.IsConnected(machineID) {
		client, err := h.SSH.GetConnection(machineID)
		if err == nil {
			running = h.Browser.IsBrowserlessRunning(client)
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"running": running,
	})
}
