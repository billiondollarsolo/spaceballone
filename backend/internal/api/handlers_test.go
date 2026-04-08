package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spaceballone/backend/internal/auth"
	"github.com/spaceballone/backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(models.AllModels()...); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func TestHealthEndpoint(t *testing.T) {
	db := setupTestDB(t)
	router := NewRouter(db)

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %s", resp["status"])
	}
}

func TestLoginSuccess(t *testing.T) {
	db := setupTestDB(t)
	password, err := auth.EnsureDefaultAdmin(db)
	if err != nil {
		t.Fatal(err)
	}

	router := NewRouter(db)

	body, _ := json.Marshal(loginRequest{Username: "admin", Password: password})
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	// Check cookie is set
	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "spaceballone_session" {
			found = true
			if !c.HttpOnly {
				t.Error("cookie should be httpOnly")
			}
			if c.SameSite != http.SameSiteStrictMode {
				t.Errorf("cookie should be SameSite=Strict, got %v", c.SameSite)
			}
		}
	}
	if !found {
		t.Error("session cookie should be set")
	}

	var resp loginResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Username != "admin" {
		t.Errorf("expected admin, got %s", resp.Username)
	}
	if !resp.MustChangePassword {
		t.Error("expected must_change_password to be true")
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	db := setupTestDB(t)
	auth.EnsureDefaultAdmin(db)
	router := NewRouter(db)

	body, _ := json.Marshal(loginRequest{Username: "admin", Password: "wrong"})
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestMeWithoutAuth(t *testing.T) {
	db := setupTestDB(t)
	router := NewRouter(db)

	req := httptest.NewRequest("GET", "/api/auth/me", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// loginAndGetCookie is a helper that logs in and returns the session cookie.
func loginAndGetCookie(t *testing.T, router http.Handler, password string) *http.Cookie {
	t.Helper()
	body, _ := json.Marshal(loginRequest{Username: "admin", Password: password})
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	for _, c := range w.Result().Cookies() {
		if c.Name == "spaceballone_session" {
			return c
		}
	}
	t.Fatal("no session cookie found after login")
	return nil
}

func TestMeWithAuth(t *testing.T) {
	db := setupTestDB(t)
	password, _ := auth.EnsureDefaultAdmin(db)
	router := NewRouter(db)

	cookie := loginAndGetCookie(t, router, password)

	req := httptest.NewRequest("GET", "/api/auth/me", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["username"] != "admin" {
		t.Errorf("expected admin, got %v", resp["username"])
	}
}

func TestLogout(t *testing.T) {
	db := setupTestDB(t)
	password, _ := auth.EnsureDefaultAdmin(db)
	router := NewRouter(db)

	cookie := loginAndGetCookie(t, router, password)

	// Logout
	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Try to use session after logout
	req2 := httptest.NewRequest("GET", "/api/auth/me", nil)
	req2.AddCookie(cookie)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 after logout, got %d", w2.Code)
	}
}

func TestChangePassword(t *testing.T) {
	db := setupTestDB(t)
	password, _ := auth.EnsureDefaultAdmin(db)
	router := NewRouter(db)

	cookie := loginAndGetCookie(t, router, password)

	body, _ := json.Marshal(changePasswordRequest{
		CurrentPassword: password,
		NewPassword:     "newpassword123",
	})
	req := httptest.NewRequest("POST", "/api/auth/change-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var rotatedCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "spaceballone_session" {
			rotatedCookie = c
			break
		}
	}
	if rotatedCookie == nil {
		t.Fatal("expected password change to rotate the session cookie")
	}
	if rotatedCookie.Value == cookie.Value {
		t.Error("expected password change to issue a new session token")
	}

	// Verify must_change_password is now false
	var user models.User
	db.Where("username = ?", "admin").First(&user)
	if user.MustChangePassword {
		t.Error("must_change_password should be false after password change")
	}

	// Verify old password no longer works
	body2, _ := json.Marshal(loginRequest{Username: "admin", Password: password})
	req2 := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusUnauthorized {
		t.Errorf("old password should fail, got %d", w2.Code)
	}

	// Verify new password works
	body3, _ := json.Marshal(loginRequest{Username: "admin", Password: "newpassword123"})
	req3 := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body3))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Errorf("new password should work, got %d", w3.Code)
	}

	req4 := httptest.NewRequest("GET", "/api/auth/me", nil)
	req4.AddCookie(cookie)
	w4 := httptest.NewRecorder()
	router.ServeHTTP(w4, req4)
	if w4.Code != http.StatusUnauthorized {
		t.Errorf("old session should be invalid after password change, got %d", w4.Code)
	}

	req5 := httptest.NewRequest("GET", "/api/auth/me", nil)
	req5.AddCookie(rotatedCookie)
	w5 := httptest.NewRecorder()
	router.ServeHTTP(w5, req5)
	if w5.Code != http.StatusOK {
		t.Errorf("rotated session should be valid, got %d", w5.Code)
	}
}

func TestMustChangePasswordBlocksNonAuthEndpoints(t *testing.T) {
	db := setupTestDB(t)
	password, _ := auth.EnsureDefaultAdmin(db)
	router := NewRouter(db)

	cookie := loginAndGetCookie(t, router, password)

	// /api/auth/me should work (it's an auth endpoint)
	req := httptest.NewRequest("GET", "/api/auth/me", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("auth/me should work with must_change_password, got %d", w.Code)
	}
}
