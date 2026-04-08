// Package codeserver manages code-server lifecycle on remote machines.
package codeserver

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	sshmanager "github.com/spaceballone/backend/internal/ssh"
	"golang.org/x/crypto/ssh"
)

// tunnelInfo holds the state for an active SSH port-forward tunnel.
type tunnelInfo struct {
	LocalPort  int
	RemotePort int
	Closer     func()
}

// Manager manages code-server instances and SSH tunnels per machine.
type Manager struct {
	mu      sync.Mutex
	tunnels map[string]*tunnelInfo // machineID -> tunnelInfo
}

// NewManager creates a new code-server manager.
func NewManager() *Manager {
	return &Manager{
		tunnels: make(map[string]*tunnelInfo),
	}
}

// StartCodeServer starts code-server on the remote machine if not already running,
// sets up an SSH tunnel, and returns the local tunnel port.
func (m *Manager) StartCodeServer(client *ssh.Client, machineID string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If we already have a tunnel, return it
	if t, ok := m.tunnels[machineID]; ok {
		return t.LocalPort, nil
	}

	// Check if code-server is running
	if !isCodeServerRunning(client) {
		// Start code-server
		_, err := sshmanager.RunCommand(client, `nohup code-server --bind-addr 127.0.0.1:8443 --auth none > /dev/null 2>&1 &`)
		if err != nil {
			return 0, fmt.Errorf("codeserver: failed to start: %w", err)
		}

		// Wait for it to be ready
		ready := false
		for i := 0; i < 10; i++ {
			time.Sleep(1 * time.Second)
			out, err := sshmanager.RunCommand(client, `curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:8443`)
			if err == nil && strings.TrimSpace(out) == "200" {
				ready = true
				break
			}
		}
		if !ready {
			return 0, fmt.Errorf("codeserver: timed out waiting for code-server to start")
		}
	}

	// Set up SSH tunnel
	localPort, closer, err := setupPortTunnel(client, 8443)
	if err != nil {
		return 0, fmt.Errorf("codeserver: failed to set up tunnel: %w", err)
	}

	m.tunnels[machineID] = &tunnelInfo{
		LocalPort:  localPort,
		RemotePort: 8443,
		Closer:     closer,
	}

	return localPort, nil
}

// StopCodeServer kills code-server on the remote machine and closes the tunnel.
func (m *Manager) StopCodeServer(client *ssh.Client, machineID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Close tunnel if exists
	if t, ok := m.tunnels[machineID]; ok {
		if t.Closer != nil {
			t.Closer()
		}
		delete(m.tunnels, machineID)
	}

	// Kill code-server
	_, err := sshmanager.RunCommand(client, `pkill -f "code-server" || true`)
	if err != nil {
		return fmt.Errorf("codeserver: failed to stop: %w", err)
	}

	return nil
}

// IsCodeServerRunning checks whether code-server is running on the remote machine.
func (m *Manager) IsCodeServerRunning(client *ssh.Client) bool {
	return isCodeServerRunning(client)
}

// GetTunnelURL returns the local tunnel URL for a machine, or empty string if none.
func (m *Manager) GetTunnelURL(machineID string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.tunnels[machineID]; ok {
		return fmt.Sprintf("http://localhost:%d", t.LocalPort)
	}
	return ""
}

// CleanupMachine removes any tunnel for a machine (called on disconnect).
func (m *Manager) CleanupMachine(machineID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.tunnels[machineID]; ok {
		if t.Closer != nil {
			t.Closer()
		}
		delete(m.tunnels, machineID)
	}
}

func isCodeServerRunning(client *ssh.Client) bool {
	out, err := sshmanager.RunCommand(client, `pgrep -f "code-server"`)
	return err == nil && strings.TrimSpace(out) != ""
}

// setupPortTunnel creates an SSH local port forward from a random local port to
// the given remote port on the SSH server.
func setupPortTunnel(client *ssh.Client, remotePort int) (int, func(), error) {
	// Bind a random local port
	localListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, nil, fmt.Errorf("failed to listen on local port: %w", err)
	}
	localPort := localListener.Addr().(*net.TCPAddr).Port

	done := make(chan struct{})
	closer := func() {
		select {
		case <-done:
		default:
			close(done)
		}
		localListener.Close()
	}

	go func() {
		for {
			localConn, err := localListener.Accept()
			if err != nil {
				select {
				case <-done:
					return
				default:
					log.Printf("codeserver: tunnel accept error: %v", err)
					return
				}
			}

			remoteAddr := fmt.Sprintf("127.0.0.1:%d", remotePort)
			remoteConn, err := client.Dial("tcp", remoteAddr)
			if err != nil {
				log.Printf("codeserver: tunnel dial remote error: %v", err)
				localConn.Close()
				continue
			}

			go func() {
				defer localConn.Close()
				defer remoteConn.Close()
				errCh := make(chan error, 2)
				go func() {
					_, err := copyData(remoteConn, localConn)
					errCh <- err
				}()
				go func() {
					_, err := copyData(localConn, remoteConn)
					errCh <- err
				}()
				<-errCh
			}()
		}
	}()

	return localPort, closer, nil
}

// copyData copies data between two connections.
func copyData(dst net.Conn, src net.Conn) (int64, error) {
	buf := make([]byte, 32*1024)
	var total int64
	for {
		n, err := src.Read(buf)
		if n > 0 {
			wn, werr := dst.Write(buf[:n])
			total += int64(wn)
			if werr != nil {
				return total, werr
			}
		}
		if err != nil {
			return total, err
		}
	}
}
