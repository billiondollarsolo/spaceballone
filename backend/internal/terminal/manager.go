// Package terminal provides tmux session management over SSH.
package terminal

import (
	"fmt"
	"strconv"
	"strings"

	sshmanager "github.com/spaceballone/backend/internal/ssh"
	"golang.org/x/crypto/ssh"
)

// Manager manages tmux sessions on remote machines via SSH.
type Manager struct{}

// NewManager creates a new terminal manager.
func NewManager() *Manager {
	return &Manager{}
}

// SessionName returns the tmux session name for a given session UUID.
// Convention: "sbo-{first 8 chars of UUID}".
func SessionName(sessionID string) string {
	if len(sessionID) >= 8 {
		return "sbo-" + sessionID[:8]
	}
	return "sbo-" + sessionID
}

// CreateTmuxSession creates a new tmux session on the remote machine.
func (m *Manager) CreateTmuxSession(client *ssh.Client, sessionName, workDir string) error {
	cmd := fmt.Sprintf("tmux new-session -d -s %s -c %s", shellEscape(sessionName), shellEscape(workDir))
	if _, err := sshmanager.RunCommand(client, cmd); err != nil {
		return fmt.Errorf("terminal: failed to create tmux session %q: %w", sessionName, err)
	}
	return nil
}

// AttachTmuxSession attaches to an existing tmux session and returns a read-write channel.
// The caller receives an ssh.Session whose Stdin/Stdout/Stderr are connected to tmux.
func AttachTmuxSession(client *ssh.Client, sessionName string, cols, rows int) (*ssh.Session, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("terminal: failed to open SSH session: %w", err)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}

	if err := session.RequestPty("xterm-256color", rows, cols, modes); err != nil {
		session.Close()
		return nil, fmt.Errorf("terminal: failed to request PTY: %w", err)
	}

	cmd := fmt.Sprintf("tmux attach-session -t %s", shellEscape(sessionName))
	if err := session.Start(cmd); err != nil {
		session.Close()
		return nil, fmt.Errorf("terminal: failed to attach to tmux session %q: %w", sessionName, err)
	}

	return session, nil
}

// AttachTmuxWindow attaches to a specific tmux window within a session.
// PrepareTmuxAttach sets up an SSH session for attaching to a tmux window.
// The caller must obtain StdinPipe/StdoutPipe before calling the returned start function.
func PrepareTmuxAttach(client *ssh.Client, sessionName string, windowIndex int, cols, rows int, workDir string) (sess *ssh.Session, start func() error, err error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, nil, fmt.Errorf("terminal: failed to open SSH session: %w", err)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}

	if err := session.RequestPty("xterm-256color", rows, cols, modes); err != nil {
		session.Close()
		return nil, nil, fmt.Errorf("terminal: failed to request PTY: %w", err)
	}

	target := fmt.Sprintf("%s:%d", sessionName, windowIndex)
	cmd := fmt.Sprintf("tmux new-session -A -s %s -c %s", shellEscape(sessionName), shellEscape(workDir))

	start = func() error {
		if err := session.Start(cmd); err != nil {
			session.Close()
			return fmt.Errorf("terminal: failed to attach to tmux window %q: %w", target, err)
		}
		return nil
	}

	return session, start, nil
}

// CreateTmuxWindow creates a new window in an existing tmux session.
// Returns the new window index.
func (m *Manager) CreateTmuxWindow(client *ssh.Client, sessionName, windowName string) (int, error) {
	cmd := fmt.Sprintf("tmux new-window -t %s -n %s -P -F '#{window_index}'",
		shellEscape(sessionName), shellEscape(windowName))
	out, err := sshmanager.RunCommand(client, cmd)
	if err != nil {
		return -1, fmt.Errorf("terminal: failed to create tmux window: %w", err)
	}
	idx, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return -1, fmt.Errorf("terminal: failed to parse window index from %q: %w", out, err)
	}
	return idx, nil
}

// KillTmuxWindow kills a specific window in a tmux session.
func (m *Manager) KillTmuxWindow(client *ssh.Client, sessionName string, windowIndex int) error {
	target := fmt.Sprintf("%s:%d", sessionName, windowIndex)
	cmd := fmt.Sprintf("tmux kill-window -t %s", shellEscape(target))
	if _, err := sshmanager.RunCommand(client, cmd); err != nil {
		return fmt.Errorf("terminal: failed to kill tmux window %q: %w", target, err)
	}
	return nil
}

// KillTmuxSession kills an entire tmux session.
func (m *Manager) KillTmuxSession(client *ssh.Client, sessionName string) error {
	cmd := fmt.Sprintf("tmux kill-session -t %s", shellEscape(sessionName))
	if _, err := sshmanager.RunCommand(client, cmd); err != nil {
		return fmt.Errorf("terminal: failed to kill tmux session %q: %w", sessionName, err)
	}
	return nil
}

// ListTmuxSessions lists existing tmux sessions on the remote machine.
func (m *Manager) ListTmuxSessions(client *ssh.Client) ([]string, error) {
	out, err := sshmanager.RunCommand(client, "tmux list-sessions -F '#{session_name}' 2>/dev/null || true")
	if err != nil {
		return nil, fmt.Errorf("terminal: failed to list tmux sessions: %w", err)
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

// SessionExists checks if a tmux session exists on the remote machine.
func (m *Manager) SessionExists(client *ssh.Client, sessionName string) bool {
	cmd := fmt.Sprintf("tmux has-session -t %s 2>/dev/null && echo yes || echo no", shellEscape(sessionName))
	out, err := sshmanager.RunCommand(client, cmd)
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) == "yes"
}

// shellEscape wraps a string in single quotes for safe shell usage.
func shellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
