// Package setup provides machine capability discovery and package installation.
package setup

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/spaceballone/backend/internal/models"
	sshmanager "github.com/spaceballone/backend/internal/ssh"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

// Capabilities represents the discovered capabilities of a remote machine.
type Capabilities struct {
	Tmux              bool   `json:"tmux"`
	TmuxVersion       string `json:"tmux_version,omitempty"`
	Docker            bool   `json:"docker"`
	DockerVersion     string `json:"docker_version,omitempty"`
	CodeServer        bool   `json:"code_server"`
	CodeServerVersion string `json:"code_server_version,omitempty"`
	Node              bool   `json:"node"`
	NodeVersion       string `json:"node_version,omitempty"`
	GoLang            bool   `json:"go_lang"`
	GoVersion         string `json:"go_version,omitempty"`
	ClaudeCode        bool   `json:"claude_code"`
	OpenCode          bool   `json:"opencode"`
	Codex             bool   `json:"codex"`
}

// Recommendation describes a recommended package to install.
type Recommendation struct {
	Package     string `json:"package"`
	Reason      string `json:"reason"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

// StatusResponse is the response for the status endpoint.
type StatusResponse struct {
	Capabilities    *Capabilities    `json:"capabilities"`
	Recommendations []Recommendation `json:"recommendations"`
}

// Manager handles setup operations on remote machines.
type Manager struct {
	DB  *gorm.DB
	SSH *sshmanager.Manager
}

// NewManager creates a new setup manager.
func NewManager(db *gorm.DB, sshMgr *sshmanager.Manager) *Manager {
	return &Manager{DB: db, SSH: sshMgr}
}

// DiscoverCapabilities runs detection commands via SSH and returns structured results.
func (m *Manager) DiscoverCapabilities(client *ssh.Client) (*Capabilities, error) {
	caps := &Capabilities{}

	// tmux
	if out, err := sshmanager.RunCommand(client, "tmux -V"); err == nil {
		caps.Tmux = true
		caps.TmuxVersion = strings.TrimSpace(out)
	}

	// docker
	if out, err := sshmanager.RunCommand(client, "docker --version"); err == nil {
		caps.Docker = true
		caps.DockerVersion = strings.TrimSpace(out)
	}

	// code-server
	if out, err := sshmanager.RunCommand(client, "which code-server && code-server --version"); err == nil {
		caps.CodeServer = true
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) > 1 {
			caps.CodeServerVersion = strings.TrimSpace(lines[1])
		}
	}

	// node
	if out, err := sshmanager.RunCommand(client, "which node && node --version"); err == nil {
		caps.Node = true
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) > 1 {
			caps.NodeVersion = strings.TrimSpace(lines[1])
		} else if len(lines) == 1 {
			caps.NodeVersion = strings.TrimSpace(lines[0])
		}
	}

	// go
	if out, err := sshmanager.RunCommand(client, "which go && go version"); err == nil {
		caps.GoLang = true
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) > 1 {
			caps.GoVersion = strings.TrimSpace(lines[1])
		} else if len(lines) == 1 {
			caps.GoVersion = strings.TrimSpace(lines[0])
		}
	}

	// claude code
	if _, err := sshmanager.RunCommand(client, "which claude"); err == nil {
		caps.ClaudeCode = true
	}

	// opencode
	if _, err := sshmanager.RunCommand(client, "which opencode"); err == nil {
		caps.OpenCode = true
	}

	// codex
	if _, err := sshmanager.RunCommand(client, "which codex"); err == nil {
		caps.Codex = true
	}

	return caps, nil
}

// SaveCapabilities stores capabilities in the database for a machine.
func (m *Manager) SaveCapabilities(machineID string, caps *Capabilities) error {
	capsJSON, err := json.Marshal(caps)
	if err != nil {
		return err
	}
	return m.DB.Model(&models.Machine{}).Where("id = ?", machineID).Update("capabilities", string(capsJSON)).Error
}

// GetRecommendations returns recommendations based on current capabilities.
func (m *Manager) GetRecommendations(caps *Capabilities) []Recommendation {
	var recs []Recommendation

	if !caps.Tmux {
		recs = append(recs, Recommendation{
			Package:     "tmux",
			Reason:      "Required for terminal session management",
			Required:    true,
			Description: "Terminal multiplexer for persistent sessions",
		})
	}
	if !caps.Docker {
		recs = append(recs, Recommendation{
			Package:     "docker",
			Reason:      "Required for container-based services",
			Required:    true,
			Description: "Container runtime for isolated environments",
		})
	}
	if !caps.CodeServer {
		recs = append(recs, Recommendation{
			Package:     "code-server",
			Reason:      "Recommended for Code tab",
			Required:    false,
			Description: "VS Code in the browser",
		})
	}
	if !caps.Node {
		recs = append(recs, Recommendation{
			Package:     "node",
			Reason:      "Recommended for JavaScript/TypeScript development",
			Required:    false,
			Description: "Node.js JavaScript runtime",
		})
	}
	if !caps.GoLang {
		recs = append(recs, Recommendation{
			Package:     "go",
			Reason:      "Recommended for Go development",
			Required:    false,
			Description: "Go programming language",
		})
	}
	if !caps.ClaudeCode {
		recs = append(recs, Recommendation{
			Package:     "claude_code",
			Reason:      "AI coding assistant by Anthropic",
			Required:    false,
			Description: "Claude Code - AI coding agent",
		})
	}
	if !caps.OpenCode {
		recs = append(recs, Recommendation{
			Package:     "opencode",
			Reason:      "AI coding assistant",
			Required:    false,
			Description: "OpenCode - terminal-based AI coding",
		})
	}
	if !caps.Codex {
		recs = append(recs, Recommendation{
			Package:     "codex",
			Reason:      "AI coding assistant by OpenAI",
			Required:    false,
			Description: "Codex CLI - OpenAI coding agent",
		})
	}

	return recs
}

// detectPackageManager detects the available package manager on the remote machine.
func detectPackageManager(client *ssh.Client) string {
	if _, err := sshmanager.RunCommand(client, "which apt-get"); err == nil {
		return "apt"
	}
	if _, err := sshmanager.RunCommand(client, "which dnf"); err == nil {
		return "dnf"
	}
	if _, err := sshmanager.RunCommand(client, "which yum"); err == nil {
		return "yum"
	}
	if _, err := sshmanager.RunCommand(client, "which brew"); err == nil {
		return "brew"
	}
	if _, err := sshmanager.RunCommand(client, "which apk"); err == nil {
		return "apk"
	}
	return ""
}

// installCommand returns the install command for the given package and package manager.
func installCommand(pkgMgr, packageName string) (string, error) {
	switch packageName {
	case "tmux":
		switch pkgMgr {
		case "apt":
			return "sudo apt-get update && sudo apt-get install -y tmux", nil
		case "dnf":
			return "sudo dnf install -y tmux", nil
		case "yum":
			return "sudo yum install -y tmux", nil
		case "brew":
			return "brew install tmux", nil
		case "apk":
			return "sudo apk add tmux", nil
		}
	case "docker":
		return "curl -fsSL https://get.docker.com | sh", nil
	case "code-server":
		return "curl -fsSL https://code-server.dev/install.sh | sh", nil
	case "node":
		switch pkgMgr {
		case "apt":
			return "curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash - && sudo apt-get install -y nodejs", nil
		case "dnf":
			return "curl -fsSL https://rpm.nodesource.com/setup_20.x | sudo bash - && sudo dnf install -y nodejs", nil
		case "yum":
			return "curl -fsSL https://rpm.nodesource.com/setup_20.x | sudo bash - && sudo yum install -y nodejs", nil
		case "brew":
			return "brew install node@20", nil
		case "apk":
			return "sudo apk add nodejs npm", nil
		}
	case "go":
		return "curl -fsSL https://go.dev/dl/go1.22.5.linux-amd64.tar.gz | sudo tar -C /usr/local -xzf - && echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.profile", nil
	case "claude_code":
		return "npm install -g @anthropic-ai/claude-code", nil
	case "opencode":
		return "go install github.com/opencode-ai/opencode@latest", nil
	case "codex":
		return "npm install -g @openai/codex", nil
	}

	return "", fmt.Errorf("unsupported package: %s", packageName)
}

// InstallPackage installs a package on the remote machine, streaming progress to progressCh.
func (m *Manager) InstallPackage(client *ssh.Client, packageName string, progressCh chan<- string) error {
	pkgMgr := detectPackageManager(client)
	if pkgMgr == "" && (packageName == "tmux" || packageName == "node") {
		progressCh <- "Error: no supported package manager found"
		close(progressCh)
		return fmt.Errorf("no supported package manager found")
	}

	cmd, err := installCommand(pkgMgr, packageName)
	if err != nil {
		progressCh <- fmt.Sprintf("Error: %s", err.Error())
		close(progressCh)
		return err
	}

	progressCh <- fmt.Sprintf("Detected package manager: %s", pkgMgr)
	progressCh <- fmt.Sprintf("Running: %s", cmd)

	session, err := client.NewSession()
	if err != nil {
		progressCh <- fmt.Sprintf("Error creating SSH session: %s", err.Error())
		close(progressCh)
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Get stdout and stderr pipes for streaming
	stdout, err := session.StdoutPipe()
	if err != nil {
		progressCh <- fmt.Sprintf("Error getting stdout pipe: %s", err.Error())
		close(progressCh)
		return err
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		progressCh <- fmt.Sprintf("Error getting stderr pipe: %s", err.Error())
		close(progressCh)
		return err
	}

	if err := session.Start(cmd); err != nil {
		progressCh <- fmt.Sprintf("Error starting command: %s", err.Error())
		close(progressCh)
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Use a WaitGroup to ensure reader goroutines finish before closing the channel
	var wg sync.WaitGroup
	wg.Add(2)

	// Stream output line by line
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, readErr := stdout.Read(buf)
			if n > 0 {
				lines := strings.Split(string(buf[:n]), "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line != "" {
						progressCh <- line
					}
				}
			}
			if readErr != nil {
				break
			}
		}
	}()

	// Also stream stderr
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, readErr := stderr.Read(buf)
			if n > 0 {
				lines := strings.Split(string(buf[:n]), "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line != "" {
						progressCh <- line
					}
				}
			}
			if readErr != nil {
				break
			}
		}
	}()

	waitErr := session.Wait()

	// Wait for reader goroutines to finish before closing the channel
	wg.Wait()

	if waitErr != nil {
		progressCh <- fmt.Sprintf("Installation failed: %s", waitErr.Error())
		close(progressCh)
		return fmt.Errorf("installation failed: %w", waitErr)
	}

	progressCh <- "Installation complete"
	close(progressCh)
	return nil
}
