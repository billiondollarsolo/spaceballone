package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/spaceballone/backend/internal/auth"
	sshmanager "github.com/spaceballone/backend/internal/ssh"
	"github.com/spaceballone/backend/internal/ws"
)

func setupMachineTestRouter(t *testing.T) (http.Handler, *http.Cookie) {
	t.Helper()
	os.Setenv("SPACEBALLONE_MASTER_KEY", "test-master-key-for-machines")
	t.Cleanup(func() { os.Unsetenv("SPACEBALLONE_MASTER_KEY") })

	db := setupTestDB(t)
	if _, _, err := auth.EnsureDefaultAdmin(db); err != nil {
		t.Fatal(err)
	}

	// Change password to clear must_change_password
	hash, _ := auth.HashPassword("newpass123")
	db.Exec("UPDATE users SET password_hash = ?, must_change_password = false WHERE email = 'admin@spaceballone.local'", hash)

	wsHub := ws.NewHub()
	sshMgr := sshmanager.NewManager(db, func(machineID, status string) {
		wsHub.BroadcastStatus(machineID, status)
	})
	t.Cleanup(func() { sshMgr.Stop() })

	router := NewRouterWithDeps(db, sshMgr, wsHub, nil, nil)
	cookie := loginAndGetCookie(t, router, "newpass123")

	return router, cookie
}

func TestCreateMachine(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)

	body, _ := json.Marshal(createMachineRequest{
		Name:        "Test Server",
		Host:        "192.168.1.100",
		Port:        22,
		AuthType:    "password",
		Credentials: "root\nsecretpass",
	})
	req := httptest.NewRequest("POST", "/api/machines", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", w.Code, w.Body.String())
	}

	var resp machineResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Name != "Test Server" {
		t.Errorf("expected name 'Test Server', got %s", resp.Name)
	}
	if resp.Host != "192.168.1.100" {
		t.Errorf("expected host '192.168.1.100', got %s", resp.Host)
	}
	if resp.Port != 22 {
		t.Errorf("expected port 22, got %d", resp.Port)
	}
	if resp.Status != "disconnected" {
		t.Errorf("expected status 'disconnected', got %s", resp.Status)
	}
	if resp.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestCreateMachineValidation(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)

	// Missing required fields
	body, _ := json.Marshal(map[string]string{"name": "test"})
	req := httptest.NewRequest("POST", "/api/machines", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	// Invalid auth_type
	body, _ = json.Marshal(createMachineRequest{
		Name:     "test",
		Host:     "host",
		AuthType: "invalid",
	})
	req = httptest.NewRequest("POST", "/api/machines", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestListMachines(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)

	// Create two machines
	for _, name := range []string{"Server 1", "Server 2"} {
		body, _ := json.Marshal(createMachineRequest{
			Name:        name,
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
	}

	// List
	req := httptest.NewRequest("GET", "/api/machines", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var machines []machineResponse
	json.NewDecoder(w.Body).Decode(&machines)
	if len(machines) != 2 {
		t.Errorf("expected 2 machines, got %d", len(machines))
	}
}

func TestGetMachine(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)

	// Create a machine
	body, _ := json.Marshal(createMachineRequest{
		Name:        "My Server",
		Host:        "10.0.0.1",
		Port:        2222,
		AuthType:    "key",
		Credentials: "root\nfake-ssh-key",
	})
	req := httptest.NewRequest("POST", "/api/machines", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var created machineResponse
	json.NewDecoder(w.Body).Decode(&created)

	// Get by ID
	req = httptest.NewRequest("GET", "/api/machines/"+created.ID, nil)
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var got machineResponse
	json.NewDecoder(w.Body).Decode(&got)
	if got.Name != "My Server" {
		t.Errorf("expected 'My Server', got %s", got.Name)
	}
	if got.Port != 2222 {
		t.Errorf("expected port 2222, got %d", got.Port)
	}
}

func TestGetMachineNotFound(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)

	req := httptest.NewRequest("GET", "/api/machines/nonexistent-id", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestUpdateMachine(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)

	// Create
	body, _ := json.Marshal(createMachineRequest{
		Name:        "Original",
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

	var created machineResponse
	json.NewDecoder(w.Body).Decode(&created)

	// Update
	body, _ = json.Marshal(createMachineRequest{
		Name: "Updated",
		Host: "10.0.0.2",
		Port: 2222,
	})
	req = httptest.NewRequest("PUT", "/api/machines/"+created.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var updated machineResponse
	json.NewDecoder(w.Body).Decode(&updated)
	if updated.Name != "Updated" {
		t.Errorf("expected 'Updated', got %s", updated.Name)
	}
	if updated.Host != "10.0.0.2" {
		t.Errorf("expected '10.0.0.2', got %s", updated.Host)
	}
	if updated.Port != 2222 {
		t.Errorf("expected 2222, got %d", updated.Port)
	}
}

func TestDeleteMachine(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)

	// Create
	body, _ := json.Marshal(createMachineRequest{
		Name:        "ToDelete",
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

	var created machineResponse
	json.NewDecoder(w.Body).Decode(&created)

	// Delete
	req = httptest.NewRequest("DELETE", "/api/machines/"+created.ID, nil)
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	// Verify it's gone
	req = httptest.NewRequest("GET", "/api/machines/"+created.ID, nil)
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", w.Code)
	}
}

func TestMachineEndpointsRequireAuth(t *testing.T) {
	os.Setenv("SPACEBALLONE_MASTER_KEY", "test-key")
	defer os.Unsetenv("SPACEBALLONE_MASTER_KEY")

	db := setupTestDB(t)
	wsHub := ws.NewHub()
	sshMgr := sshmanager.NewManager(db, func(machineID, status string) {})
	defer sshMgr.Stop()
	router := NewRouterWithDeps(db, sshMgr, wsHub, nil, nil)

	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/api/machines"},
		{"POST", "/api/machines"},
		{"GET", "/api/machines/some-id"},
		{"PUT", "/api/machines/some-id"},
		{"DELETE", "/api/machines/some-id"},
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

func TestCredentialsNotInResponse(t *testing.T) {
	router, cookie := setupMachineTestRouter(t)

	body, _ := json.Marshal(createMachineRequest{
		Name:        "Secret Server",
		Host:        "10.0.0.1",
		Port:        22,
		AuthType:    "password",
		Credentials: "root\nsuper-secret-password-12345",
	})
	req := httptest.NewRequest("POST", "/api/machines", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	responseBody := w.Body.String()
	if bytes.Contains([]byte(responseBody), []byte("super-secret-password-12345")) {
		t.Error("credentials should not appear in response")
	}
	if bytes.Contains([]byte(responseBody), []byte("encrypted_credentials")) {
		t.Error("encrypted_credentials should not appear in response")
	}

	// Also check list endpoint
	var created machineResponse
	json.Unmarshal([]byte(responseBody), &created)

	req = httptest.NewRequest("GET", "/api/machines", nil)
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	listBody := w.Body.String()
	if bytes.Contains([]byte(listBody), []byte("super-secret-password-12345")) {
		t.Error("credentials should not appear in list response")
	}
}
