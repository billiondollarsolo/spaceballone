package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/spaceballone/backend/internal/models"
	sshmanager "github.com/spaceballone/backend/internal/ssh"
	"github.com/spaceballone/backend/internal/terminal"
	"gorm.io/gorm"
)

// ProjectHandler holds dependencies for project endpoints.
type ProjectHandler struct {
	DB       *gorm.DB
	SSH      *sshmanager.Manager
	Terminal *terminal.Manager
}

type createProjectRequest struct {
	Name          string `json:"name"`
	DirectoryPath string `json:"directory_path"`
}

type projectResponse struct {
	ID            string `json:"id"`
	MachineID     string `json:"machine_id"`
	Name          string `json:"name"`
	DirectoryPath string `json:"directory_path"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

func toProjectResponse(p *models.Project) projectResponse {
	return projectResponse{
		ID:            p.ID,
		MachineID:     p.MachineID,
		Name:          p.Name,
		DirectoryPath: p.DirectoryPath,
		CreatedAt:     p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     p.UpdatedAt.Format(time.RFC3339),
	}
}

// ListProjects handles GET /api/machines/{id}/projects.
func (h *ProjectHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "id")

	var machine models.Machine
	if err := h.DB.First(&machine, "id = ?", machineID).Error; err != nil {
		writeError(w, http.StatusNotFound, "machine not found")
		return
	}

	var projects []models.Project
	if err := h.DB.Where("machine_id = ?", machineID).Find(&projects).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list projects")
		return
	}

	resp := make([]projectResponse, len(projects))
	for i := range projects {
		resp[i] = toProjectResponse(&projects[i])
	}

	writeJSON(w, http.StatusOK, resp)
}

// CreateProject handles POST /api/machines/{id}/projects.
func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "id")

	var machine models.Machine
	if err := h.DB.First(&machine, "id = ?", machineID).Error; err != nil {
		writeError(w, http.StatusNotFound, "machine not found")
		return
	}

	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.DirectoryPath == "" {
		writeError(w, http.StatusBadRequest, "name and directory_path are required")
		return
	}

	project := models.Project{
		MachineID:     machineID,
		Name:          req.Name,
		DirectoryPath: req.DirectoryPath,
	}

	if err := h.DB.Create(&project).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create project")
		return
	}

	writeJSON(w, http.StatusCreated, toProjectResponse(&project))
}

// GetProject handles GET /api/projects/{id}.
func (h *ProjectHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var project models.Project
	if err := h.DB.First(&project, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	writeJSON(w, http.StatusOK, toProjectResponse(&project))
}

// UpdateProject handles PUT /api/projects/{id}.
func (h *ProjectHandler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var project models.Project
	if err := h.DB.First(&project, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name != "" {
		project.Name = req.Name
	}
	if req.DirectoryPath != "" {
		project.DirectoryPath = req.DirectoryPath
	}

	if err := h.DB.Save(&project).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update project")
		return
	}

	writeJSON(w, http.StatusOK, toProjectResponse(&project))
}

// DeleteProject handles DELETE /api/projects/{id}.
func (h *ProjectHandler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var project models.Project
	if err := h.DB.First(&project, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	// Cascade: delete terminal tabs, sessions
	var sessions []models.Session
	h.DB.Where("project_id = ?", id).Find(&sessions)
	for _, s := range sessions {
		// Kill tmux session if machine is connected
		if h.SSH != nil && h.SSH.IsConnected(project.MachineID) {
			client, err := h.SSH.GetConnection(project.MachineID)
			if err == nil {
				h.Terminal.KillTmuxSession(client, terminal.SessionName(s.ID))
			}
		}
		h.DB.Where("session_id = ?", s.ID).Delete(&models.TerminalTab{})
	}
	h.DB.Where("project_id = ?", id).Delete(&models.Session{})
	h.DB.Delete(&project)

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// FileEntry represents a file or directory in the remote file browser.
type FileEntry struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "file" or "dir"
	Size        string `json:"size"`
	Modified    string `json:"modified"`
	Permissions string `json:"permissions"`
}

// BrowseDirectory handles GET /api/machines/{id}/browse?path=/.
func (h *ProjectHandler) BrowseDirectory(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "id")

	_, client, ok := requireConnectedMachine(h.DB, h.SSH, machineID, w)
	if !ok {
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/"
	}

	cmd := fmt.Sprintf("ls -la %s", shellQuote(path))
	out, err := sshmanager.RunCommand(client, cmd)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to list directory: "+err.Error())
		return
	}

	entries := parseLsOutput(out)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"path":    path,
		"entries": entries,
	})
}

// parseLsOutput parses the output of `ls -la` into structured entries.
func parseLsOutput(output string) []FileEntry {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var entries []FileEntry

	for _, line := range lines {
		// Skip the "total" line
		if strings.HasPrefix(line, "total ") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		name := strings.Join(fields[8:], " ")
		if name == "." {
			continue
		}

		entryType := "file"
		perms := fields[0]
		if len(perms) > 0 && perms[0] == 'd' {
			entryType = "dir"
		} else if len(perms) > 0 && perms[0] == 'l' {
			entryType = "file" // symlinks shown as files
		}

		modified := strings.Join(fields[5:8], " ")

		entries = append(entries, FileEntry{
			Name:        name,
			Type:        entryType,
			Size:        fields[4],
			Modified:    modified,
			Permissions: perms,
		})
	}

	return entries
}

// shellQuote wraps a string for safe shell usage.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

// escapeJSON escapes a string for inclusion in a JSON string literal.
func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
