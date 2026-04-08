// Package browser manages Browserless (headless Chrome) containers on remote machines.
package browser

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	sshmanager "github.com/spaceballone/backend/internal/ssh"
	"golang.org/x/crypto/ssh"
)

// Manager manages Browserless Docker containers on remote machines.
type Manager struct{}

// NewManager creates a new Browserless manager.
func NewManager() *Manager {
	return &Manager{}
}

func browserlessStartCommand() string {
	return `docker run -d --name spaceballone-browser -p 127.0.0.1:9222:3000 --restart unless-stopped browserless/chrome:latest`
}

// StartBrowserless starts the Browserless Docker container on the remote machine.
func (m *Manager) StartBrowserless(client *ssh.Client, machineID string) error {
	// Check if already running
	if m.IsBrowserlessRunning(client) {
		return nil
	}

	// Start the container
	cmd := browserlessStartCommand()
	if _, err := sshmanager.RunCommand(client, cmd); err != nil {
		return fmt.Errorf("browserless: failed to start container: %w", err)
	}

	// Wait for the CDP endpoint to be ready
	ready := false
	for i := 0; i < 15; i++ {
		time.Sleep(2 * time.Second)
		out, err := sshmanager.RunCommand(client, `curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:9222/json/version`)
		if err == nil && strings.TrimSpace(out) == "200" {
			ready = true
			break
		}
	}
	if !ready {
		return fmt.Errorf("browserless: timed out waiting for container to start")
	}

	return nil
}

// StopBrowserless stops and removes the Browserless Docker container.
func (m *Manager) StopBrowserless(client *ssh.Client) error {
	_, err := sshmanager.RunCommand(client, `docker stop spaceballone-browser && docker rm spaceballone-browser`)
	if err != nil {
		return fmt.Errorf("browserless: failed to stop container: %w", err)
	}
	return nil
}

// CleanupMachine cleans up any browser resources for a machine on disconnect.
// Since there's no local tunnel state, this is a no-op placeholder for interface
// compatibility. The container keeps running on the remote machine.
func (m *Manager) CleanupMachine(machineID string) {
	// No local state to clean up for browser manager.
	// The remote Docker container will be cleaned up on next connect or explicitly via StopBrowserless.
}

// IsBrowserlessRunning checks if the Browserless container is running.
func (m *Manager) IsBrowserlessRunning(client *ssh.Client) bool {
	out, err := sshmanager.RunCommand(client, `docker ps --filter name=spaceballone-browser --format '{{.Status}}'`)
	return err == nil && strings.TrimSpace(out) != ""
}

// SetupCDPTunnel creates an SSH tunnel to the remote CDP port (9222) and returns
// a local listener port and cleanup function.
func SetupCDPTunnel(client *ssh.Client, remotePort int) (int, func(), error) {
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
					log.Printf("browser: tunnel accept error: %v", err)
					return
				}
			}

			remoteAddr := fmt.Sprintf("127.0.0.1:%d", remotePort)
			remoteConn, err := client.Dial("tcp", remoteAddr)
			if err != nil {
				log.Printf("browser: tunnel dial remote error: %v", err)
				localConn.Close()
				continue
			}

			go proxyConns(localConn, remoteConn)
		}
	}()

	return localPort, closer, nil
}

func proxyConns(a, b net.Conn) {
	defer a.Close()
	defer b.Close()
	errCh := make(chan error, 2)
	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := a.Read(buf)
			if n > 0 {
				if _, werr := b.Write(buf[:n]); werr != nil {
					errCh <- werr
					return
				}
			}
			if err != nil {
				errCh <- err
				return
			}
		}
	}()
	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := b.Read(buf)
			if n > 0 {
				if _, werr := a.Write(buf[:n]); werr != nil {
					errCh <- werr
					return
				}
			}
			if err != nil {
				errCh <- err
				return
			}
		}
	}()
	<-errCh
}
