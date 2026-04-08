package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/spaceballone/backend/internal/models"
	"github.com/spaceballone/backend/internal/setup"
	sshmanager "github.com/spaceballone/backend/internal/ssh"
	"gorm.io/gorm"
)

// SetupHandler holds dependencies for setup endpoints.
type SetupHandler struct {
	DB    *gorm.DB
	SSH   *sshmanager.Manager
	Setup *setup.Manager
}

// Discover handles POST /api/machines/{id}/setup/discover.
func (h *SetupHandler) Discover(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	_, client, ok := requireConnectedMachine(h.DB, h.SSH, id, w)
	if !ok {
		return
	}

	caps, err := h.Setup.DiscoverCapabilities(client)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("discovery failed: %s", err.Error()))
		return
	}

	if err := h.Setup.SaveCapabilities(id, caps); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save capabilities")
		return
	}

	writeJSON(w, http.StatusOK, caps)
}

// installRequest is the request body for the install endpoint.
type installRequest struct {
	Package string `json:"package"`
}

// Install handles POST /api/machines/{id}/setup/install.
// Streams progress via Server-Sent Events.
func (h *SetupHandler) Install(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	_, client, ok := requireConnectedMachine(h.DB, h.SSH, id, w)
	if !ok {
		return
	}

	var req installRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Package == "" {
		writeError(w, http.StatusBadRequest, "package is required")
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	progressCh := make(chan string, 100)

	// Run installation in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- h.Setup.InstallPackage(client, req.Package, progressCh)
	}()

	// Stream progress
	for line := range progressCh {
		data := map[string]interface{}{
			"line": line,
			"done": false,
		}
		jsonData, _ := json.Marshal(data)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
	}

	// Send final message
	installErr := <-errCh
	success := installErr == nil

	finalData := map[string]interface{}{
		"line": "Installation complete",
		"done": true,
	}
	if !success {
		finalData["line"] = fmt.Sprintf("Installation failed: %s", installErr.Error())
		finalData["error"] = true
	}
	jsonData, _ := json.Marshal(finalData)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
	flusher.Flush()
}

// Status handles GET /api/machines/{id}/setup/status.
func (h *SetupHandler) Status(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var machine models.Machine
	if err := h.DB.First(&machine, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusNotFound, "machine not found")
		return
	}

	var caps *setup.Capabilities
	if machine.Capabilities != "" {
		caps = &setup.Capabilities{}
		if err := json.Unmarshal([]byte(machine.Capabilities), caps); err != nil {
			caps = nil
		}
	}

	if caps == nil {
		caps = &setup.Capabilities{}
	}

	recs := h.Setup.GetRecommendations(caps)

	writeJSON(w, http.StatusOK, setup.StatusResponse{
		Capabilities:    caps,
		Recommendations: recs,
	})
}
