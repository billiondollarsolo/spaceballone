package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// createTestProject is a helper that creates a machine + project and returns both IDs.
func createTestProject(t *testing.T, router http.Handler, cookie *http.Cookie) (machineID, projectID string) {
	t.Helper()
	machineID = createTestMachine(t, router, cookie)

	body, _ := json.Marshal(createProjectRequest{
		Name:          "Test Project",
		DirectoryPath: "/home/test",
	})
	req := httptest.NewRequest("POST", "/api/machines/"+machineID+"/projects", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("failed to create project: %d %s", w.Code, w.Body.String())
	}
	var resp projectResponse
	json.NewDecoder(w.Body).Decode(&resp)
	return machineID, resp.ID
}

func TestCreateSession(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)
	_, projectID := createTestProject(t, router, cookie)

	req := httptest.NewRequest("POST", "/api/projects/"+projectID+"/sessions", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", w.Code, w.Body.String())
	}

	var resp sessionResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Name != "Session 1" {
		t.Errorf("expected auto-name 'Session 1', got %s", resp.Name)
	}
	if resp.Status != "active" {
		t.Errorf("expected status 'active', got %s", resp.Status)
	}
	if resp.ProjectID != projectID {
		t.Errorf("expected project_id %s, got %s", projectID, resp.ProjectID)
	}
	// Should have a default terminal tab
	if len(resp.TerminalTabs) != 1 {
		t.Fatalf("expected 1 default terminal tab, got %d", len(resp.TerminalTabs))
	}
	if resp.TerminalTabs[0].Name != "Terminal 1" {
		t.Errorf("expected tab name 'Terminal 1', got %s", resp.TerminalTabs[0].Name)
	}
	if resp.TerminalTabs[0].TmuxWindowIndex != 0 {
		t.Errorf("expected tmux window index 0, got %d", resp.TerminalTabs[0].TmuxWindowIndex)
	}
}

func TestCreateSessionAutoNaming(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)
	_, projectID := createTestProject(t, router, cookie)

	// Create first session
	req := httptest.NewRequest("POST", "/api/projects/"+projectID+"/sessions", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var first sessionResponse
	json.NewDecoder(w.Body).Decode(&first)
	if first.Name != "Session 1" {
		t.Errorf("expected 'Session 1', got %s", first.Name)
	}

	// Create second session
	req = httptest.NewRequest("POST", "/api/projects/"+projectID+"/sessions", nil)
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var second sessionResponse
	json.NewDecoder(w.Body).Decode(&second)
	if second.Name != "Session 2" {
		t.Errorf("expected 'Session 2', got %s", second.Name)
	}
}

func TestCreateSessionCustomName(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)
	_, projectID := createTestProject(t, router, cookie)

	body, _ := json.Marshal(createSessionRequest{Name: "My Custom Session"})
	req := httptest.NewRequest("POST", "/api/projects/"+projectID+"/sessions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var resp sessionResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Name != "My Custom Session" {
		t.Errorf("expected 'My Custom Session', got %s", resp.Name)
	}
}

func TestCreateSessionProjectNotFound(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)

	req := httptest.NewRequest("POST", "/api/projects/nonexistent/sessions", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestListSessions(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)
	_, projectID := createTestProject(t, router, cookie)

	// Create two sessions
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/api/projects/"+projectID+"/sessions", nil)
		req.AddCookie(cookie)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("failed to create session: %d", w.Code)
		}
	}

	req := httptest.NewRequest("GET", "/api/projects/"+projectID+"/sessions", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var sessions []sessionResponse
	json.NewDecoder(w.Body).Decode(&sessions)
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions))
	}
}

func TestGetSession(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)
	_, projectID := createTestProject(t, router, cookie)

	// Create session
	req := httptest.NewRequest("POST", "/api/projects/"+projectID+"/sessions", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var created sessionResponse
	json.NewDecoder(w.Body).Decode(&created)

	// Get session
	req = httptest.NewRequest("GET", "/api/sessions/"+created.ID, nil)
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var got sessionResponse
	json.NewDecoder(w.Body).Decode(&got)
	if got.Name != "Session 1" {
		t.Errorf("expected 'Session 1', got %s", got.Name)
	}
	if len(got.TerminalTabs) != 1 {
		t.Errorf("expected 1 terminal tab, got %d", len(got.TerminalTabs))
	}
}

func TestGetSessionNotFound(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)

	req := httptest.NewRequest("GET", "/api/sessions/nonexistent-id", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestUpdateSession(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)
	_, projectID := createTestProject(t, router, cookie)

	// Create session
	req := httptest.NewRequest("POST", "/api/projects/"+projectID+"/sessions", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var created sessionResponse
	json.NewDecoder(w.Body).Decode(&created)

	// Update
	body, _ := json.Marshal(createSessionRequest{Name: "Renamed Session"})
	req = httptest.NewRequest("PUT", "/api/sessions/"+created.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var updated sessionResponse
	json.NewDecoder(w.Body).Decode(&updated)
	if updated.Name != "Renamed Session" {
		t.Errorf("expected 'Renamed Session', got %s", updated.Name)
	}
}

func TestDeleteSession(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)
	_, projectID := createTestProject(t, router, cookie)

	// Create session
	req := httptest.NewRequest("POST", "/api/projects/"+projectID+"/sessions", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var created sessionResponse
	json.NewDecoder(w.Body).Decode(&created)

	// Delete
	req = httptest.NewRequest("DELETE", "/api/sessions/"+created.ID, nil)
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	// Verify gone
	req = httptest.NewRequest("GET", "/api/sessions/"+created.ID, nil)
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", w.Code)
	}
}

func TestDeleteSessionCascadesTerminalTabs(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)
	_, projectID := createTestProject(t, router, cookie)

	// Create session (auto-creates default terminal tab)
	req := httptest.NewRequest("POST", "/api/projects/"+projectID+"/sessions", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var sess sessionResponse
	json.NewDecoder(w.Body).Decode(&sess)

	tabID := sess.TerminalTabs[0].ID

	// Delete session
	req = httptest.NewRequest("DELETE", "/api/sessions/"+sess.ID, nil)
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Terminal tab should also be gone
	req = httptest.NewRequest("DELETE", "/api/terminals/"+tabID, nil)
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for deleted terminal tab, got %d", w.Code)
	}
}

func TestCreateTerminal(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)
	_, projectID := createTestProject(t, router, cookie)

	// Create session
	req := httptest.NewRequest("POST", "/api/projects/"+projectID+"/sessions", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var sess sessionResponse
	json.NewDecoder(w.Body).Decode(&sess)

	// Create additional terminal tab
	req = httptest.NewRequest("POST", "/api/sessions/"+sess.ID+"/terminals", nil)
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", w.Code, w.Body.String())
	}

	var tab terminalTabResponse
	json.NewDecoder(w.Body).Decode(&tab)
	if tab.Name != "Terminal 2" {
		t.Errorf("expected 'Terminal 2', got %s", tab.Name)
	}
	if tab.SessionID != sess.ID {
		t.Errorf("expected session_id %s, got %s", sess.ID, tab.SessionID)
	}
}

func TestDeleteTerminal(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)
	_, projectID := createTestProject(t, router, cookie)

	// Create session with default tab
	req := httptest.NewRequest("POST", "/api/projects/"+projectID+"/sessions", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var sess sessionResponse
	json.NewDecoder(w.Body).Decode(&sess)

	// Create a second terminal tab
	req = httptest.NewRequest("POST", "/api/sessions/"+sess.ID+"/terminals", nil)
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var tab terminalTabResponse
	json.NewDecoder(w.Body).Decode(&tab)

	// Delete the second tab
	req = httptest.NewRequest("DELETE", "/api/terminals/"+tab.ID, nil)
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	// Verify gone
	req = httptest.NewRequest("DELETE", "/api/terminals/"+tab.ID, nil)
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", w.Code)
	}
}

func TestDeleteTerminalNotFound(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)

	req := httptest.NewRequest("DELETE", "/api/terminals/nonexistent", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestSessionEndpointsRequireAuth(t *testing.T) {
	router, _ := setupMachineTestRouter(t)

	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/api/projects/some-id/sessions"},
		{"POST", "/api/projects/some-id/sessions"},
		{"GET", "/api/sessions/some-id"},
		{"PUT", "/api/sessions/some-id"},
		{"DELETE", "/api/sessions/some-id"},
		{"POST", "/api/sessions/some-id/terminals"},
		{"DELETE", "/api/terminals/some-id"},
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

func TestSearchEndpoint(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)
	machineID := createTestMachine(t, router, cookie)

	// Create a project
	body, _ := json.Marshal(createProjectRequest{
		Name:          "Search Project",
		DirectoryPath: "/tmp/search",
	})
	req := httptest.NewRequest("POST", "/api/machines/"+machineID+"/projects", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Search for "Search"
	req = httptest.NewRequest("GET", "/api/search?q=Search", nil)
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var results []SearchResult
	json.NewDecoder(w.Body).Decode(&results)
	if len(results) < 1 {
		t.Errorf("expected at least 1 result, got %d", len(results))
	}

	foundProject := false
	for _, r := range results {
		if r.Type == "project" && r.Name == "Search Project" {
			foundProject = true
		}
	}
	if !foundProject {
		t.Error("expected to find 'Search Project' in results")
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)

	req := httptest.NewRequest("GET", "/api/search?q=", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var results []SearchResult
	json.NewDecoder(w.Body).Decode(&results)
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty query, got %d", len(results))
	}
}

func TestSearchNoResults(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)

	req := httptest.NewRequest("GET", "/api/search?q=zzz_nonexistent_zzz", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var results []SearchResult
	json.NewDecoder(w.Body).Decode(&results)
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}
