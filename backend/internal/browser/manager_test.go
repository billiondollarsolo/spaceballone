package browser

import (
	"strings"
	"testing"
)

func TestNewManager(t *testing.T) {
	mgr := NewManager()
	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
}

func TestBrowserlessStartCommandBindsLoopbackOnly(t *testing.T) {
	cmd := browserlessStartCommand()
	if cmd == "" {
		t.Fatal("expected non-empty command")
	}
	if want := "-p 127.0.0.1:9222:3000"; !strings.Contains(cmd, want) {
		t.Fatalf("expected command to contain %q, got %q", want, cmd)
	}
	if strings.Contains(cmd, "5900") {
		t.Fatalf("expected command to omit 5900 exposure, got %q", cmd)
	}
}
