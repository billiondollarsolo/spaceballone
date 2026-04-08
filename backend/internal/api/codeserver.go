package api

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

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

	_ = port // tunnel port used internally; expose via reverse proxy path
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"url": "/api/code-server-proxy/" + machineID,
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
		tunnelURL := h.CodeServer.GetTunnelURL(machineID)
		if tunnelURL != "" {
			proxyURL := "/api/code-server-proxy/" + machineID
			url = &proxyURL
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

	tunnelURL := h.CodeServer.GetTunnelURL(machineID)
	if tunnelURL == "" {
		writeError(w, http.StatusBadRequest, "code-server is not running or no tunnel available")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"url": "/api/code-server-proxy/" + machineID + "/?folder=" + folder,
	})
}

// ProxyCodeServer reverse-proxies requests to the code-server SSH tunnel.
func (h *CodeServerHandler) ProxyCodeServer(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineId")

	tunnelURL := h.CodeServer.GetTunnelURL(machineID)
	if tunnelURL == "" {
		writeError(w, http.StatusBadGateway, "code-server is not running")
		return
	}

	target, err := url.Parse(tunnelURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invalid tunnel URL")
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// Strip the proxy prefix from the path
	prefix := "/api/code-server-proxy/" + machineID
	r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
	if r.URL.Path == "" {
		r.URL.Path = "/"
	}
	r.Host = target.Host

	proxy.ServeHTTP(w, r)
}
