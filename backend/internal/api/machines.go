package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/spaceballone/backend/internal/crypto"
	"github.com/spaceballone/backend/internal/models"
	sshmanager "github.com/spaceballone/backend/internal/ssh"
	"gorm.io/gorm"
)

// MachineHandler holds dependencies for machine endpoints.
type MachineHandler struct {
	DB  *gorm.DB
	SSH *sshmanager.Manager
}

type createMachineRequest struct {
	Name        string `json:"name"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	AuthType    string `json:"auth_type"`
	Credentials string `json:"credentials"`
}

type machineResponse struct {
	ID                 string      `json:"id"`
	Name               string      `json:"name"`
	Host               string      `json:"host"`
	Port               int         `json:"port"`
	AuthType           string      `json:"auth_type"`
	HostKeyFingerprint string      `json:"host_key_fingerprint,omitempty"`
	Status             string      `json:"status"`
	Capabilities       interface{} `json:"capabilities"`
	LastHeartbeat      interface{} `json:"last_heartbeat,omitempty"`
	CreatedAt          string      `json:"created_at"`
	UpdatedAt          string      `json:"updated_at"`
}

func toMachineResponse(m *models.Machine) machineResponse {
	resp := machineResponse{
		ID:                 m.ID,
		Name:               m.Name,
		Host:               m.Host,
		Port:               m.Port,
		AuthType:           m.AuthType,
		HostKeyFingerprint: m.HostKeyFingerprint,
		Status:             m.Status,
		CreatedAt:          m.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:          m.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if m.LastHeartbeat != nil {
		resp.LastHeartbeat = m.LastHeartbeat.Format("2006-01-02T15:04:05Z")
	}

	if m.Capabilities != "" {
		var caps interface{}
		if err := json.Unmarshal([]byte(m.Capabilities), &caps); err == nil {
			resp.Capabilities = caps
		}
	}

	return resp
}

// CreateMachine handles POST /api/machines.
func (h *MachineHandler) CreateMachine(w http.ResponseWriter, r *http.Request) {
	var req createMachineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.Host == "" || req.AuthType == "" {
		writeError(w, http.StatusBadRequest, "name, host, and auth_type are required")
		return
	}

	if req.AuthType != models.AuthTypePassword && req.AuthType != models.AuthTypeKey {
		writeError(w, http.StatusBadRequest, "auth_type must be 'password' or 'key'")
		return
	}

	if req.Port == 0 {
		req.Port = 22
	}

	key, err := crypto.GetMasterKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "master key not configured")
		return
	}

	encrypted, err := crypto.Encrypt([]byte(req.Credentials), key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encrypt credentials")
		return
	}

	machine := models.Machine{
		Name:                 req.Name,
		Host:                 req.Host,
		Port:                 req.Port,
		AuthType:             req.AuthType,
		EncryptedCredentials: encrypted,
		Status:               models.MachineStatusDisconnected,
	}

	if err := h.DB.Create(&machine).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create machine")
		return
	}

	writeJSON(w, http.StatusCreated, toMachineResponse(&machine))
}

// ListMachines handles GET /api/machines.
func (h *MachineHandler) ListMachines(w http.ResponseWriter, r *http.Request) {
	var machines []models.Machine
	if err := h.DB.Find(&machines).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list machines")
		return
	}

	resp := make([]machineResponse, len(machines))
	for i := range machines {
		resp[i] = toMachineResponse(&machines[i])
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetMachine handles GET /api/machines/:id.
func (h *MachineHandler) GetMachine(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var machine models.Machine
	if err := h.DB.First(&machine, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusNotFound, "machine not found")
		return
	}

	writeJSON(w, http.StatusOK, toMachineResponse(&machine))
}

// UpdateMachine handles PUT /api/machines/:id.
func (h *MachineHandler) UpdateMachine(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var machine models.Machine
	if err := h.DB.First(&machine, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusNotFound, "machine not found")
		return
	}

	var req createMachineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name != "" {
		machine.Name = req.Name
	}
	if req.Host != "" {
		if req.Host != machine.Host {
			machine.HostKeyFingerprint = ""
		}
		machine.Host = req.Host
	}
	if req.Port != 0 {
		if req.Port != machine.Port {
			machine.HostKeyFingerprint = ""
		}
		machine.Port = req.Port
	}
	if req.AuthType != "" {
		if req.AuthType != models.AuthTypePassword && req.AuthType != models.AuthTypeKey {
			writeError(w, http.StatusBadRequest, "auth_type must be 'password' or 'key'")
			return
		}
		machine.AuthType = req.AuthType
	}

	if req.Credentials != "" {
		key, err := crypto.GetMasterKey()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "master key not configured")
			return
		}
		encrypted, err := crypto.Encrypt([]byte(req.Credentials), key)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to encrypt credentials")
			return
		}
		machine.EncryptedCredentials = encrypted
	}

	if err := h.DB.Save(&machine).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update machine")
		return
	}

	writeJSON(w, http.StatusOK, toMachineResponse(&machine))
}

// DeleteMachine handles DELETE /api/machines/:id.
func (h *MachineHandler) DeleteMachine(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var machine models.Machine
	if err := h.DB.First(&machine, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusNotFound, "machine not found")
		return
	}

	// Disconnect if connected
	if h.SSH != nil && h.SSH.IsConnected(id) {
		_ = h.SSH.Disconnect(id)
	}

	// Cascade delete: sessions via projects
	var projects []models.Project
	h.DB.Where("machine_id = ?", id).Find(&projects)
	for _, p := range projects {
		// Delete terminal tabs via sessions
		var sessions []models.Session
		h.DB.Where("project_id = ?", p.ID).Find(&sessions)
		for _, s := range sessions {
			h.DB.Where("session_id = ?", s.ID).Delete(&models.TerminalTab{})
		}
		h.DB.Where("project_id = ?", p.ID).Delete(&models.Session{})
	}
	h.DB.Where("machine_id = ?", id).Delete(&models.Project{})
	h.DB.Delete(&machine)

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ConnectMachine handles POST /api/machines/:id/connect.
func (h *MachineHandler) ConnectMachine(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var machine models.Machine
	if err := h.DB.First(&machine, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusNotFound, "machine not found")
		return
	}

	key, err := crypto.GetMasterKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "master key not configured")
		return
	}

	creds, err := crypto.Decrypt(machine.EncryptedCredentials, key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to decrypt credentials")
		return
	}

	if err := h.SSH.Connect(&machine, string(creds)); err != nil {
		log.Printf("Failed to connect to machine %s: %v", id, err)
		writeError(w, http.StatusBadGateway, "Failed to connect to machine")
		return
	}

	// Run capability discovery
	caps, err := h.SSH.DiscoverCapabilities(machine.ID)
	if err != nil {
		// Connection succeeded but capability discovery failed; not fatal.
		log.Printf("Capability discovery failed for machine %s: %v", machine.ID, err)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":       models.MachineStatusConnected,
			"capabilities": nil,
			"warning":      "capability discovery failed",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":       models.MachineStatusConnected,
		"capabilities": caps,
	})
}

// DisconnectMachine handles POST /api/machines/:id/disconnect.
func (h *MachineHandler) DisconnectMachine(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.SSH.Disconnect(id); err != nil {
		log.Printf("Failed to disconnect machine %s: %v", id, err)
		writeError(w, http.StatusBadRequest, "Failed to disconnect machine")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": models.MachineStatusDisconnected})
}

// GetCapabilities handles GET /api/machines/:id/capabilities.
func (h *MachineHandler) GetCapabilities(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var machine models.Machine
	if err := h.DB.First(&machine, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusNotFound, "machine not found")
		return
	}

	if machine.Capabilities != "" {
		var caps interface{}
		if err := json.Unmarshal([]byte(machine.Capabilities), &caps); err == nil {
			writeJSON(w, http.StatusOK, caps)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{})
}
