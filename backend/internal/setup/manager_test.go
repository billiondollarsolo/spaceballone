package setup

import (
	"testing"
)

func TestGetRecommendations_AllMissing(t *testing.T) {
	mgr := &Manager{}
	caps := &Capabilities{}
	recs := mgr.GetRecommendations(caps)

	if len(recs) != 7 {
		t.Errorf("expected 7 recommendations, got %d", len(recs))
	}

	// Check that tmux and docker are required
	required := 0
	for _, r := range recs {
		if r.Required {
			required++
		}
	}
	if required != 2 {
		t.Errorf("expected 2 required recommendations (tmux, docker), got %d", required)
	}
}

func TestGetRecommendations_AllPresent(t *testing.T) {
	mgr := &Manager{}
	caps := &Capabilities{
		Tmux:       true,
		Docker:     true,
		Node:       true,
		GoLang:     true,
		ClaudeCode: true,
		OpenCode:   true,
		Codex:      true,
	}
	recs := mgr.GetRecommendations(caps)
	if len(recs) != 0 {
		t.Errorf("expected 0 recommendations when all present, got %d", len(recs))
	}
}

func TestInstallCommand_Tmux(t *testing.T) {
	cmd, err := installCommand("apt", "tmux")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd == "" {
		t.Fatal("expected non-empty command")
	}
}

func TestInstallCommand_Unsupported(t *testing.T) {
	_, err := installCommand("apt", "unknown-package")
	if err == nil {
		t.Fatal("expected error for unsupported package")
	}
}

func TestInstallCommand_Docker(t *testing.T) {
	cmd, err := installCommand("", "docker")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd != "curl -fsSL https://get.docker.com | sh" {
		t.Errorf("unexpected docker install command: %s", cmd)
	}
}
