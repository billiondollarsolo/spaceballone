package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"net"
	"testing"

	gossh "golang.org/x/crypto/ssh"
)

func testPublicKey(t *testing.T) gossh.PublicKey {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	publicKey, err := gossh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("failed to create SSH public key: %v", err)
	}
	return publicKey
}

func TestHostKeyCallbackTrustOnFirstUse(t *testing.T) {
	key := testPublicKey(t)
	var seen string

	callback := hostKeyCallback("", &seen)
	if err := callback("example.com", &net.IPAddr{}, key); err != nil {
		t.Fatalf("expected TOFU callback to accept key, got %v", err)
	}
	if seen == "" {
		t.Fatal("expected callback to capture host key fingerprint")
	}
}

func TestHostKeyCallbackRejectsMismatch(t *testing.T) {
	key := testPublicKey(t)
	callback := hostKeyCallback("SHA256:not-the-right-fingerprint", nil)
	if err := callback("example.com", &net.IPAddr{}, key); err == nil {
		t.Fatal("expected mismatched fingerprint to be rejected")
	}
}

func TestHostKeyCallbackAcceptsExpectedFingerprint(t *testing.T) {
	key := testPublicKey(t)
	expected := gossh.FingerprintSHA256(key)
	callback := hostKeyCallback(expected, nil)
	if err := callback("example.com", &net.IPAddr{}, key); err != nil {
		t.Fatalf("expected matching fingerprint to be accepted, got %v", err)
	}
}
