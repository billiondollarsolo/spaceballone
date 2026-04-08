// Package crypto provides AES-256-GCM encryption for stored credentials.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

// ErrMasterKeyNotSet is returned when SPACEBALLONE_MASTER_KEY is not configured.
var ErrMasterKeyNotSet = errors.New("SPACEBALLONE_MASTER_KEY environment variable is not set")

// ValidateMasterKey checks that the master key env var is present.
func ValidateMasterKey() error {
	if os.Getenv("SPACEBALLONE_MASTER_KEY") == "" {
		return ErrMasterKeyNotSet
	}
	return nil
}

var (
	cachedKey    []byte
	cachedKeyErr error
	keyOnce      sync.Once
)

// ResetMasterKeyCache resets the cached master key so that the next call to
// GetMasterKey re-reads the environment variable. This is intended for tests only.
func ResetMasterKeyCache() {
	keyOnce = sync.Once{}
	cachedKey = nil
	cachedKeyErr = nil
}

// GetMasterKey returns the derived encryption key from the environment variable.
// The key is computed once and cached for the lifetime of the process.
func GetMasterKey() ([]byte, error) {
	keyOnce.Do(func() {
		raw := os.Getenv("SPACEBALLONE_MASTER_KEY")
		if raw == "" {
			cachedKeyErr = ErrMasterKeyNotSet
			return
		}
		hash := sha256.Sum256([]byte(raw))
		cachedKey = hash[:]
	})
	return cachedKey, cachedKeyErr
}

// Encrypt encrypts plaintext using AES-256-GCM with the provided key.
// The returned ciphertext includes the nonce prepended.
func Encrypt(plaintext []byte, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("crypto: key must be 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("crypto: failed to generate nonce: %w", err)
	}

	// Seal appends the encrypted data to the nonce
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM with the provided key.
// Expects the nonce to be prepended to the ciphertext.
func Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("crypto: key must be 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("crypto: ciphertext too short")
	}

	nonce, ciphertextBody := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBody, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: decryption failed: %w", err)
	}

	return plaintext, nil
}
