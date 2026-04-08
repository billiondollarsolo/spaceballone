package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/spaceballone/backend/internal/ports"
	sshmanager "github.com/spaceballone/backend/internal/ssh"
	"gorm.io/gorm"
)

type PortHandler struct {
	DB    *gorm.DB
	SSH   *sshmanager.Manager
	Ports *ports.Manager
}

func (h *PortHandler) ListPorts(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "id")
	_, client, ok := requireConnectedMachine(h.DB, h.SSH, machineID, w)
	if !ok {
		return
	}

	projectDir := r.URL.Query().Get("project_dir")

	result, err := h.Ports.ScanPorts(client, projectDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to scan ports: "+err.Error())
		return
	}

	if result == nil {
		result = []ports.DiscoveredPort{}
	}

	writeJSON(w, http.StatusOK, result)
}
