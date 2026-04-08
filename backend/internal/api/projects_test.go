package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// createTestMachine is a helper that creates a machine and returns its ID.
func createTestMachine(t *testing.T, router http.Handler, cookie *http.Cookie) string {
	t.Helper()
	body, _ := json.Marshal(createMachineRequest{
		Name:        "Test Machine",
		Host:        "10.0.0.1",
		Port:        22,
		AuthType:    "password",
		Credentials: "root\npass",
	})
	req := httptest.NewRequest("POST", "/api/machines", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("failed to create machine: %d %s", w.Code, w.Body.String())
	}
	var resp machineResponse
	json.NewDecoder(w.Body).Decode(&resp)
	return resp.ID
}

func TestCreateProject(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)
	machineID := createTestMachine(t, router, cookie)

	body, _ := json.Marshal(createProjectRequest{
		Name:          "My Project",
		DirectoryPath: "/home/user/project",
	})
	req := httptest.NewRequest("POST", "/api/machines/"+machineID+"/projects", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", w.Code, w.Body.String())
	}

	var resp projectResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Name != "My Project" {
		t.Errorf("expected name 'My Project', got %s", resp.Name)
	}
	if resp.DirectoryPath != "/home/user/project" {
		t.Errorf("expected directory_path '/home/user/project', got %s", resp.DirectoryPath)
	}
	if resp.MachineID != machineID {
		t.Errorf("expected machine_id %s, got %s", machineID, resp.MachineID)
	}
	if resp.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestCreateProjectValidation(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)
	machineID := createTestMachine(t, router, cookie)

	// Missing required fields
	body, _ := json.Marshal(map[string]string{"name": "test"})
	req := httptest.NewRequest("POST", "/api/machines/"+machineID+"/projects", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateProjectMachineNotFound(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)

	body, _ := json.Marshal(createProjectRequest{
		Name:          "test",
		DirectoryPath: "/tmp",
	})
	req := httptest.NewRequest("POST", "/api/machines/nonexistent/projects", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestListProjects(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)
	machineID := createTestMachine(t, router, cookie)

	// Create two projects
	for _, name := range []string{"Project 1", "Project 2"} {
		body, _ := json.Marshal(createProjectRequest{
			Name:          name,
			DirectoryPath: "/home/" + name,
		})
		req := httptest.NewRequest("POST", "/api/machines/"+machineID+"/projects", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(cookie)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("failed to create project: %d %s", w.Code, w.Body.String())
		}
	}

	req := httptest.NewRequest("GET", "/api/machines/"+machineID+"/projects", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var projects []projectResponse
	json.NewDecoder(w.Body).Decode(&projects)
	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}
}

func TestGetProject(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)
	machineID := createTestMachine(t, router, cookie)

	body, _ := json.Marshal(createProjectRequest{
		Name:          "Get Test",
		DirectoryPath: "/opt/project",
	})
	req := httptest.NewRequest("POST", "/api/machines/"+machineID+"/projects", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var created projectResponse
	json.NewDecoder(w.Body).Decode(&created)

	req = httptest.NewRequest("GET", "/api/projects/"+created.ID, nil)
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var got projectResponse
	json.NewDecoder(w.Body).Decode(&got)
	if got.Name != "Get Test" {
		t.Errorf("expected 'Get Test', got %s", got.Name)
	}
}

func TestGetProjectNotFound(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)

	req := httptest.NewRequest("GET", "/api/projects/nonexistent-id", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestUpdateProject(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)
	machineID := createTestMachine(t, router, cookie)

	body, _ := json.Marshal(createProjectRequest{
		Name:          "Original",
		DirectoryPath: "/tmp/orig",
	})
	req := httptest.NewRequest("POST", "/api/machines/"+machineID+"/projects", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var created projectResponse
	json.NewDecoder(w.Body).Decode(&created)

	// Update
	body, _ = json.Marshal(createProjectRequest{
		Name:          "Updated",
		DirectoryPath: "/tmp/updated",
	})
	req = httptest.NewRequest("PUT", "/api/projects/"+created.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var updated projectResponse
	json.NewDecoder(w.Body).Decode(&updated)
	if updated.Name != "Updated" {
		t.Errorf("expected 'Updated', got %s", updated.Name)
	}
	if updated.DirectoryPath != "/tmp/updated" {
		t.Errorf("expected '/tmp/updated', got %s", updated.DirectoryPath)
	}
}

func TestDeleteProject(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)
	machineID := createTestMachine(t, router, cookie)

	body, _ := json.Marshal(createProjectRequest{
		Name:          "ToDelete",
		DirectoryPath: "/tmp/del",
	})
	req := httptest.NewRequest("POST", "/api/machines/"+machineID+"/projects", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var created projectResponse
	json.NewDecoder(w.Body).Decode(&created)

	// Delete
	req = httptest.NewRequest("DELETE", "/api/projects/"+created.ID, nil)
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	// Verify gone
	req = httptest.NewRequest("GET", "/api/projects/"+created.ID, nil)
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", w.Code)
	}
}

func TestDeleteProjectCascadesSessions(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)
	machineID := createTestMachine(t, router, cookie)

	// Create project
	body, _ := json.Marshal(createProjectRequest{
		Name:          "CascadeTest",
		DirectoryPath: "/tmp/cascade",
	})
	req := httptest.NewRequest("POST", "/api/machines/"+machineID+"/projects", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var proj projectResponse
	json.NewDecoder(w.Body).Decode(&proj)

	// Create a session
	req = httptest.NewRequest("POST", "/api/projects/"+proj.ID+"/sessions", nil)
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for session, got %d; body: %s", w.Code, w.Body.String())
	}
	var sess sessionResponse
	json.NewDecoder(w.Body).Decode(&sess)

	// Delete project
	req = httptest.NewRequest("DELETE", "/api/projects/"+proj.ID, nil)
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Session should be gone
	req = httptest.NewRequest("GET", "/api/sessions/"+sess.ID, nil)
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cascaded session, got %d", w.Code)
	}
}

func TestProjectEndpointsRequireAuth(t *testing.T) {
	router, _ := setupMachineTestRouter(t)

	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/api/machines/some-id/projects"},
		{"POST", "/api/machines/some-id/projects"},
		{"GET", "/api/projects/some-id"},
		{"PUT", "/api/projects/some-id"},
		{"DELETE", "/api/projects/some-id"},
		{"GET", "/api/machines/some-id/browse?path=/"},
	}

	for _, ep := range endpoints {
		req := httptest.NewRequest(ep.method, ep.path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: expected 401, got %d", ep.method, ep.path, w.Code)
		}
	}
}

func TestParseLsOutput(t *testing.T) {
	output := `total 32
drwxr-xr-x  5 root root 4096 Jan  1 12:00 .
drwxr-xr-x 20 root root 4096 Jan  1 12:00 ..
-rw-r--r--  1 root root  220 Jan  1 12:00 .bashrc
drwxr-xr-x  2 root root 4096 Jan  1 12:00 Documents
-rwxr-xr-x  1 root root 8192 Jan  1 12:00 script.sh
lrwxrwxrwx  1 root root   11 Jan  1 12:00 link -> /etc/hosts`

	entries := parseLsOutput(output)

	if len(entries) != 5 { // skip "." but include ".."
		t.Fatalf("expected 5 entries, got %d", len(entries))
	}

	// Check ".."
	if entries[0].Name != ".." || entries[0].Type != "dir" {
		t.Errorf("expected '..' dir, got %s %s", entries[0].Name, entries[0].Type)
	}

	// Check .bashrc
	if entries[1].Name != ".bashrc" || entries[1].Type != "file" {
		t.Errorf("expected '.bashrc' file, got %s %s", entries[1].Name, entries[1].Type)
	}

	// Check Documents
	if entries[2].Name != "Documents" || entries[2].Type != "dir" {
		t.Errorf("expected 'Documents' dir, got %s %s", entries[2].Name, entries[2].Type)
	}

	// Check script.sh
	if entries[3].Name != "script.sh" || entries[3].Type != "file" {
		t.Errorf("expected 'script.sh' file, got %s %s", entries[3].Name, entries[3].Type)
	}

	// Check symlink (shown as file)
	if entries[4].Type != "file" {
		t.Errorf("expected symlink as file, got %s", entries[4].Type)
	}
}
