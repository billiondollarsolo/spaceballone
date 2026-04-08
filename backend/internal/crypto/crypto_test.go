package crypto

import (
	"bytes"
	"crypto/sha256"
	"os"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := sha256.Sum256([]byte("test-master-key"))

	plaintext := []byte("super-secret-password")
	ciphertext, err := Encrypt(plaintext, key[:])
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if bytes.Equal(plaintext, ciphertext) {
		t.Fatal("ciphertext should not equal plaintext")
	}

	decrypted, err := Decrypt(ciphertext, key[:])
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("decrypted text does not match: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptProducesDifferentCiphertexts(t *testing.T) {
	key := sha256.Sum256([]byte("test-master-key"))
	plaintext := []byte("same-input")

	ct1, err := Encrypt(plaintext, key[:])
	if err != nil {
		t.Fatal(err)
	}
	ct2, err := Encrypt(plaintext, key[:])
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(ct1, ct2) {
		t.Fatal("encrypting the same plaintext twice should produce different ciphertexts (random nonce)")
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	key1 := sha256.Sum256([]byte("key-one"))
	key2 := sha256.Sum256([]byte("key-two"))

	ciphertext, err := Encrypt([]byte("secret"), key1[:])
	if err != nil {
		t.Fatal(err)
	}

	_, err = Decrypt(ciphertext, key2[:])
	if err == nil {
		t.Fatal("expected decryption with wrong key to fail")
	}
}

func TestDecryptTooShort(t *testing.T) {
	key := sha256.Sum256([]byte("key"))
	_, err := Decrypt([]byte("short"), key[:])
	if err == nil {
		t.Fatal("expected error for short ciphertext")
	}
}

func TestEncryptInvalidKeyLength(t *testing.T) {
	_, err := Encrypt([]byte("test"), []byte("short-key"))
	if err == nil {
		t.Fatal("expected error for invalid key length")
	}
}

func TestGetMasterKey(t *testing.T) {
	ResetMasterKeyCache()
	os.Setenv("SPACEBALLONE_MASTER_KEY", "test-key-123")
	defer os.Unsetenv("SPACEBALLONE_MASTER_KEY")

	key, err := GetMasterKey()
	if err != nil {
		t.Fatalf("GetMasterKey failed: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(key))
	}
}

func TestGetMasterKeyNotSet(t *testing.T) {
	ResetMasterKeyCache()
	os.Unsetenv("SPACEBALLONE_MASTER_KEY")
	_, err := GetMasterKey()
	if err != ErrMasterKeyNotSet {
		t.Fatalf("expected ErrMasterKeyNotSet, got %v", err)
	}
}

func TestEncryptDecryptEmptyPlaintext(t *testing.T) {
	key := sha256.Sum256([]byte("test-master-key"))
	plaintext := []byte("")

	ciphertext, err := Encrypt(plaintext, key[:])
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(ciphertext, key[:])
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("decrypted text does not match: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptDecryptLargePayload(t *testing.T) {
	key := sha256.Sum256([]byte("test-master-key"))
	// Simulate a large SSH private key
	plaintext := bytes.Repeat([]byte("A"), 4096)

	ciphertext, err := Encrypt(plaintext, key[:])
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(ciphertext, key[:])
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatal("decrypted text does not match for large payload")
	}
}
