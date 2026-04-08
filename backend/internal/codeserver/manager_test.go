package codeserver

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	mgr := NewManager()
	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
	if mgr.tunnels == nil {
		t.Fatal("expected tunnels map to be initialized")
	}
}

func TestGetTunnelURL_NoTunnel(t *testing.T) {
	mgr := NewManager()
	url := mgr.GetTunnelURL("nonexistent")
	if url != "" {
		t.Errorf("expected empty URL, got %q", url)
	}
}

func TestCleanupMachine_NoTunnel(t *testing.T) {
	mgr := NewManager()
	// Should not panic
	mgr.CleanupMachine("nonexistent")
}

func TestCleanupMachine_WithTunnel(t *testing.T) {
	mgr := NewManager()
	closed := false
	mgr.tunnels["machine-1"] = &tunnelInfo{
		LocalPort:  12345,
		RemotePort: 8443,
		Closer:     func() { closed = true },
	}

	mgr.CleanupMachine("machine-1")

	if !closed {
		t.Error("expected closer to be called")
	}
	if _, ok := mgr.tunnels["machine-1"]; ok {
		t.Error("expected tunnel to be removed")
	}
}
