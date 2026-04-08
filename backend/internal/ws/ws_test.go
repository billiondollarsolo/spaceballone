package ws

import (
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

func authenticatedRequest(t *testing.T, db *gorm.DB, mustChangePassword bool) *http.Request {
	t.Helper()

	_, password, err := auth.EnsureDefaultAdmin(db)
	if err != nil {
		t.Fatalf("EnsureDefaultAdmin failed: %v", err)
	}

	var user models.User
	if err := db.Where("email = ?", "admin@spaceballone.local").First(&user).Error; err != nil {
		t.Fatalf("admin user not found: %v", err)
	}

	if !mustChangePassword {
		hash, err := auth.HashPassword(password)
		if err != nil {
			t.Fatalf("failed to hash password: %v", err)
		}
		if err := db.Model(&user).Updates(map[string]interface{}{
			"password_hash":        hash,
			"must_change_password": false,
		}).Error; err != nil {
			t.Fatalf("failed to update user: %v", err)
		}
	}

	session, err := auth.CreateSession(db, user.ID)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/ws/status", nil)
	req.AddCookie(&http.Cookie{Name: "spaceballone_session", Value: session.SessionToken})
	return req
}

func TestValidateWSSessionRejectsMustChangePassword(t *testing.T) {
	db := setupTestDB(t)
	req := authenticatedRequest(t, db, true)
	w := httptest.NewRecorder()

	if _, ok := ValidateWSSession(db, w, req); ok {
		t.Fatal("expected websocket auth to reject must_change_password users")
	}
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestValidateWSSessionAcceptsNormalUser(t *testing.T) {
	db := setupTestDB(t)
	req := authenticatedRequest(t, db, false)
	w := httptest.NewRecorder()

	user, ok := ValidateWSSession(db, w, req)
	if !ok {
		t.Fatal("expected websocket auth to accept a normal user")
	}
	if user == nil || user.Email != "admin@spaceballone.local" {
		t.Fatalf("expected authenticated admin user, got %#v", user)
	}
}
