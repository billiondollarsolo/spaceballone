package api

import (
	"encoding/json"
	"net/http"

	"github.com/spaceballone/backend/internal/models"
	sshmanager "github.com/spaceballone/backend/internal/ssh"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeError writes a JSON error response with the given status code and message.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// requireConnectedMachine looks up a machine by ID, verifies it is connected via SSH,
// and returns the machine, SSH client, and true. On failure it writes an error response
// and returns nil, nil, false.
func requireConnectedMachine(db *gorm.DB, sshMgr *sshmanager.Manager, machineID string, w http.ResponseWriter) (*models.Machine, *ssh.Client, bool) {
	var machine models.Machine
	if err := db.First(&machine, "id = ?", machineID).Error; err != nil {
		writeError(w, http.StatusNotFound, "machine not found")
		return nil, nil, false
	}
	if !sshMgr.IsConnected(machineID) {
		writeError(w, http.StatusBadGateway, "machine not connected")
		return nil, nil, false
	}
	client, err := sshMgr.GetConnection(machineID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to get SSH connection")
		return nil, nil, false
	}
	return &machine, client, true
}
