package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/spaceballone/backend/internal/models"
	"golang.org/x/crypto/argon2"
	"gorm.io/gorm"
)

const (
	argonTime    = 1
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeyLen  = 32
	saltLen      = 16
)

// SessionExpiry returns the configured session lifetime.
func SessionExpiry() time.Duration {
	val := os.Getenv("SESSION_EXPIRY")
	if val == "" {
		val = "24h"
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return 24 * time.Hour
	}
	return d
}

// HashPassword hashes a password using Argon2id with a random salt.
// Returns "hex(salt):hex(hash)".
func HashPassword(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return hex.EncodeToString(salt) + ":" + hex.EncodeToString(hash), nil
}

// VerifyPassword checks a password against an Argon2id hash string.
func VerifyPassword(password, encoded string) bool {
	// Parse "hex(salt):hex(hash)"
	var saltHex, hashHex string
	for i := 0; i < len(encoded); i++ {
		if encoded[i] == ':' {
			saltHex = encoded[:i]
			hashHex = encoded[i+1:]
			break
		}
	}
	if saltHex == "" || hashHex == "" {
		return false
	}

	salt, err := hex.DecodeString(saltHex)
	if err != nil {
		return false
	}
	expectedHash, err := hex.DecodeString(hashHex)
	if err != nil {
		return false
	}

	computed := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	if len(computed) != len(expectedHash) {
		return false
	}
	// Constant-time compare
	var result byte
	for i := range computed {
		result |= computed[i] ^ expectedHash[i]
	}
	return result == 0
}

// GenerateSessionToken creates a cryptographically random session token.
func GenerateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GenerateRandomPassword creates a random password for the default admin user.
func GenerateRandomPassword() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// EnsureDefaultAdmin creates the default admin user if no users exist.
// Returns the generated password (empty string if admin already exists).
func EnsureDefaultAdmin(db *gorm.DB) (string, error) {
	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		return "", fmt.Errorf("failed to count users: %w", err)
	}
	if count > 0 {
		return "", nil
	}

	password, err := GenerateRandomPassword()
	if err != nil {
		return "", fmt.Errorf("failed to generate password: %w", err)
	}

	hash, err := HashPassword(password)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	admin := models.User{
		Username:           "admin",
		PasswordHash:       hash,
		MustChangePassword: true,
	}
	if err := db.Create(&admin).Error; err != nil {
		return "", fmt.Errorf("failed to create admin user: %w", err)
	}

	return password, nil
}

// CreateSession creates a new app session for a user.
func CreateSession(db *gorm.DB, userID string) (*models.AppSession, error) {
	token, err := GenerateSessionToken()
	if err != nil {
		return nil, err
	}

	session := models.AppSession{
		UserID:       userID,
		SessionToken: token,
		ExpiresAt:    time.Now().Add(SessionExpiry()),
	}
	if err := db.Create(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// ValidateSession checks if a session token is valid and not expired.
func ValidateSession(db *gorm.DB, token string) (*models.AppSession, error) {
	var session models.AppSession
	if err := db.Where("session_token = ? AND expires_at > ?", token, time.Now()).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// InvalidateSession deletes a session by token.
func InvalidateSession(db *gorm.DB, token string) error {
	return db.Where("session_token = ?", token).Delete(&models.AppSession{}).Error
}
