package auth

import (
	"testing"

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

func TestHashAndVerifyPassword(t *testing.T) {
	password := "testpassword123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == "" {
		t.Fatal("hash should not be empty")
	}
	if !VerifyPassword(password, hash) {
		t.Error("VerifyPassword should return true for correct password")
	}
	if VerifyPassword("wrongpassword", hash) {
		t.Error("VerifyPassword should return false for wrong password")
	}
}

func TestGenerateSessionToken(t *testing.T) {
	token1, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("GenerateSessionToken failed: %v", err)
	}
	token2, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("GenerateSessionToken failed: %v", err)
	}
	if token1 == token2 {
		t.Error("tokens should be unique")
	}
	if len(token1) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("expected token length 64, got %d", len(token1))
	}
}

func TestEnsureDefaultAdmin(t *testing.T) {
	db := setupTestDB(t)

	// First call should create admin
	password, err := EnsureDefaultAdmin(db)
	if err != nil {
		t.Fatalf("EnsureDefaultAdmin failed: %v", err)
	}
	if password == "" {
		t.Error("expected password on first call")
	}

	// Verify admin user exists
	var user models.User
	if err := db.Where("username = ?", "admin").First(&user).Error; err != nil {
		t.Fatalf("admin user not found: %v", err)
	}
	if !user.MustChangePassword {
		t.Error("admin should have must_change_password=true")
	}
	if !VerifyPassword(password, user.PasswordHash) {
		t.Error("password should verify against stored hash")
	}

	// Second call should not create another admin
	password2, err := EnsureDefaultAdmin(db)
	if err != nil {
		t.Fatalf("second EnsureDefaultAdmin failed: %v", err)
	}
	if password2 != "" {
		t.Error("expected empty password on second call")
	}
}

func TestCreateAndValidateSession(t *testing.T) {
	db := setupTestDB(t)

	password, _ := EnsureDefaultAdmin(db)
	_ = password

	var user models.User
	db.Where("username = ?", "admin").First(&user)

	session, err := CreateSession(db, user.ID)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	if session.SessionToken == "" {
		t.Error("session token should not be empty")
	}

	// Validate session
	validated, err := ValidateSession(db, session.SessionToken)
	if err != nil {
		t.Fatalf("ValidateSession failed: %v", err)
	}
	if validated.UserID != user.ID {
		t.Error("validated session should have correct user ID")
	}

	// Invalid token
	_, err = ValidateSession(db, "invalid-token")
	if err == nil {
		t.Error("ValidateSession should fail with invalid token")
	}
}

func TestInvalidateSession(t *testing.T) {
	db := setupTestDB(t)
	EnsureDefaultAdmin(db)

	var user models.User
	db.Where("username = ?", "admin").First(&user)

	session, _ := CreateSession(db, user.ID)

	if err := InvalidateSession(db, session.SessionToken); err != nil {
		t.Fatalf("InvalidateSession failed: %v", err)
	}

	_, err := ValidateSession(db, session.SessionToken)
	if err == nil {
		t.Error("session should be invalid after invalidation")
	}
}

func TestInvalidateUserSessions(t *testing.T) {
	db := setupTestDB(t)
	EnsureDefaultAdmin(db)

	var user models.User
	db.Where("username = ?", "admin").First(&user)

	session1, _ := CreateSession(db, user.ID)
	session2, _ := CreateSession(db, user.ID)

	if err := InvalidateUserSessions(db, user.ID); err != nil {
		t.Fatalf("InvalidateUserSessions failed: %v", err)
	}

	if _, err := ValidateSession(db, session1.SessionToken); err == nil {
		t.Error("session1 should be invalid after user invalidation")
	}
	if _, err := ValidateSession(db, session2.SessionToken); err == nil {
		t.Error("session2 should be invalid after user invalidation")
	}
}
