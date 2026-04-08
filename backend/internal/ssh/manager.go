// Package ssh provides SSH connection management for remote machines.
package ssh

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spaceballone/backend/internal/models"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

// Capabilities represents the discovered capabilities of a remote machine.
type Capabilities struct {
	Docker        bool   `json:"docker"`
	DockerVersion string `json:"docker_version,omitempty"`
	Tmux          bool   `json:"tmux"`
	TmuxVersion   string `json:"tmux_version,omitempty"`
	Node          bool   `json:"node"`
	GoLang        bool   `json:"go_lang"`
}

// connEntry holds the SSH client and retry state for a machine connection.
type connEntry struct {
	client    *ssh.Client
	machine   *models.Machine
	creds     string
	stopRetry chan struct{}
}

// StatusChangeFunc is called when a machine's status changes.
type StatusChangeFunc func(machineID, status string)

// Manager manages persistent SSH connections to remote machines.
type Manager struct {
	mu             sync.RWMutex
	connections    map[string]*connEntry
	db             *gorm.DB
	onStatusChange StatusChangeFunc
	stopMonitor    chan struct{}
	OnReconnect    func(machineID string)
	OnDisconnect   func(machineID string)
}

// NewManager creates a new SSH connection manager.
func NewManager(db *gorm.DB, onStatusChange StatusChangeFunc) *Manager {
	m := &Manager{
		connections:    make(map[string]*connEntry),
		db:             db,
		onStatusChange: onStatusChange,
		stopMonitor:    make(chan struct{}),
	}
	go m.healthMonitor()
	return m
}

func hostKeyCallback(expectedFingerprint string, seenFingerprint *string) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		fingerprint := ssh.FingerprintSHA256(key)
		if seenFingerprint != nil {
			*seenFingerprint = fingerprint
		}
		if expectedFingerprint == "" {
			return nil
		}
		if subtle.ConstantTimeCompare([]byte(fingerprint), []byte(expectedFingerprint)) == 1 {
			return nil
		}
		return fmt.Errorf("ssh: host key mismatch for %s: expected %s, got %s", hostname, expectedFingerprint, fingerprint)
	}
}

// buildSSHConfig parses the credential string and builds an ssh.ClientConfig.
func (m *Manager) buildSSHConfig(machine *models.Machine, decryptedCreds string, seenFingerprint *string) (*ssh.ClientConfig, error) {
	config := &ssh.ClientConfig{
		User:            "root",
		HostKeyCallback: hostKeyCallback(machine.HostKeyFingerprint, seenFingerprint),
		Timeout:         10 * time.Second,
	}

	creds := decryptedCreds
	user := "root"
	if parts := strings.SplitN(decryptedCreds, "\n", 2); len(parts) == 2 {
		user = parts[0]
		creds = parts[1]
	}
	config.User = user

	switch machine.AuthType {
	case models.AuthTypePassword:
		config.Auth = []ssh.AuthMethod{
			ssh.Password(creds),
		}
	case models.AuthTypeKey:
		signer, err := ssh.ParsePrivateKey([]byte(creds))
		if err != nil {
			return nil, fmt.Errorf("ssh: failed to parse private key: %w", err)
		}
		config.Auth = []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		}
	default:
		return nil, fmt.Errorf("ssh: unsupported auth type: %s", machine.AuthType)
	}

	return config, nil
}

// Connect establishes an SSH connection to a machine.
func (m *Manager) Connect(machine *models.Machine, decryptedCredentials string) error {
	var seenFingerprint string
	config, err := m.buildSSHConfig(machine, decryptedCredentials, &seenFingerprint)
	if err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", machine.Host, machine.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		m.updateStatus(machine.ID, models.MachineStatusError)
		return fmt.Errorf("ssh: failed to connect: %w", err)
	}

	m.mu.Lock()
	// Close existing connection if any
	if existing, ok := m.connections[machine.ID]; ok {
		if existing.stopRetry != nil {
			close(existing.stopRetry)
		}
		if existing.client != nil {
			existing.client.Close()
		}
	}
	m.connections[machine.ID] = &connEntry{
		client:  client,
		machine: machine,
		creds:   decryptedCredentials,
	}
	m.mu.Unlock()

	if machine.HostKeyFingerprint == "" && seenFingerprint != "" {
		if err := m.db.Model(&models.Machine{}).
			Where("id = ? AND (host_key_fingerprint = '' OR host_key_fingerprint IS NULL)", machine.ID).
			Update("host_key_fingerprint", seenFingerprint).Error; err != nil {
			log.Printf("ssh: failed to persist host key fingerprint for machine %s: %v", machine.ID, err)
		} else {
			machine.HostKeyFingerprint = seenFingerprint
		}
	}

	m.updateStatus(machine.ID, models.MachineStatusConnected)
	return nil
}

// Disconnect closes an SSH connection.
func (m *Manager) Disconnect(machineID string) error {
	m.mu.Lock()
	entry, ok := m.connections[machineID]
	if !ok {
		m.mu.Unlock()
		m.updateStatus(machineID, models.MachineStatusDisconnected)
		return nil
	}
	if entry.stopRetry != nil {
		close(entry.stopRetry)
	}
	delete(m.connections, machineID)
	m.mu.Unlock()

	if entry.client != nil {
		entry.client.Close()
	}
	m.updateStatus(machineID, models.MachineStatusDisconnected)
	if m.OnDisconnect != nil {
		m.OnDisconnect(machineID)
	}
	return nil
}

// GetConnection returns the active SSH client for a machine.
func (m *Manager) GetConnection(machineID string) (*ssh.Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, ok := m.connections[machineID]
	if !ok || entry.client == nil {
		return nil, fmt.Errorf("ssh: machine %s is not connected", machineID)
	}
	return entry.client, nil
}

// IsConnected checks if a machine has an active SSH connection.
func (m *Manager) IsConnected(machineID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, ok := m.connections[machineID]
	return ok && entry.client != nil
}

// DiscoverCapabilities runs commands on the remote machine and returns capabilities.
func (m *Manager) DiscoverCapabilities(machineID string) (*Capabilities, error) {
	client, err := m.GetConnection(machineID)
	if err != nil {
		return nil, err
	}

	caps := &Capabilities{}

	// Docker
	if out, err := RunCommand(client, "docker --version"); err == nil {
		caps.Docker = true
		caps.DockerVersion = strings.TrimSpace(out)
	}

	// Tmux
	if out, err := RunCommand(client, "tmux -V"); err == nil {
		caps.Tmux = true
		caps.TmuxVersion = strings.TrimSpace(out)
	}

	// Node
	if _, err := RunCommand(client, "which node"); err == nil {
		caps.Node = true
	}

	// Go
	if _, err := RunCommand(client, "which go"); err == nil {
		caps.GoLang = true
	}

	// Store capabilities in DB
	capsJSON, _ := json.Marshal(caps)
	m.db.Model(&models.Machine{}).Where("id = ?", machineID).Update("capabilities", string(capsJSON))

	return caps, nil
}

// Stop stops the health monitor and closes all connections.
func (m *Manager) Stop() {
	close(m.stopMonitor)
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, entry := range m.connections {
		if entry.stopRetry != nil {
			close(entry.stopRetry)
		}
		if entry.client != nil {
			entry.client.Close()
		}
		delete(m.connections, id)
	}
}

// RunCommand executes a command on the remote machine and returns combined output.
func RunCommand(client *ssh.Client, cmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	out, err := session.CombinedOutput(cmd)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (m *Manager) updateStatus(machineID, status string) {
	now := time.Now()
	updates := map[string]interface{}{
		"status": status,
	}
	if status == models.MachineStatusConnected {
		updates["last_heartbeat"] = now
	}
	m.db.Model(&models.Machine{}).Where("id = ?", machineID).Updates(updates)

	if m.onStatusChange != nil {
		m.onStatusChange(machineID, status)
	}
}

func (m *Manager) healthMonitor() {
	interval := 30 * time.Second
	if v := os.Getenv("HEARTBEAT_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			interval = d
		} else {
			log.Printf("ssh: invalid HEARTBEAT_INTERVAL %q, using default 30s", v)
		}
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopMonitor:
			return
		case <-ticker.C:
			m.pingAll()
		}
	}
}

func (m *Manager) pingAll() {
	m.mu.RLock()
	entries := make(map[string]*connEntry, len(m.connections))
	for id, entry := range m.connections {
		entries[id] = entry
	}
	m.mu.RUnlock()

	for id, entry := range entries {
		if entry.client == nil {
			continue
		}
		if _, err := RunCommand(entry.client, "echo ok"); err != nil {
			log.Printf("SSH health check failed for machine %s: %v", id, err)
			m.handleDisconnect(id, entry)
		} else {
			now := time.Now()
			m.db.Model(&models.Machine{}).Where("id = ?", id).Updates(map[string]interface{}{
				"last_heartbeat": now,
				"status":         models.MachineStatusConnected,
			})
		}
	}
}

func (m *Manager) handleDisconnect(machineID string, entry *connEntry) {
	m.updateStatus(machineID, models.MachineStatusReconnecting)

	stopRetry := make(chan struct{})
	m.mu.Lock()
	if e, ok := m.connections[machineID]; ok {
		e.stopRetry = stopRetry
	}
	m.mu.Unlock()

	go m.retryConnection(machineID, entry, stopRetry)
}

func (m *Manager) retryConnection(machineID string, entry *connEntry, stop chan struct{}) {
	backoff := time.Second
	maxBackoff := 60 * time.Second
	attempt := 0

	for {
		attempt++
		select {
		case <-stop:
			return
		case <-time.After(backoff):
		}

		var seenFingerprint string
		config, err := m.buildSSHConfig(entry.machine, entry.creds, &seenFingerprint)
		if err != nil {
			log.Printf("SSH retry: config build failed for machine %s: %v", machineID, err)
			m.updateStatus(machineID, models.MachineStatusError)
			if m.OnDisconnect != nil {
				m.OnDisconnect(machineID)
			}
			return
		}

		addr := fmt.Sprintf("%s:%d", entry.machine.Host, entry.machine.Port)
		client, err := ssh.Dial("tcp", addr, config)
		if err != nil {
			log.Printf("SSH retry attempt %d failed for machine %s: %v", attempt, machineID, err)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		m.mu.Lock()
		if e, ok := m.connections[machineID]; ok {
			if e.client != nil {
				e.client.Close()
			}
			e.client = client
			e.stopRetry = nil
		}
		m.mu.Unlock()

		if entry.machine.HostKeyFingerprint == "" && seenFingerprint != "" {
			if err := m.db.Model(&models.Machine{}).
				Where("id = ? AND (host_key_fingerprint = '' OR host_key_fingerprint IS NULL)", entry.machine.ID).
				Update("host_key_fingerprint", seenFingerprint).Error; err != nil {
				log.Printf("ssh: failed to persist host key fingerprint for machine %s during retry: %v", entry.machine.ID, err)
			} else {
				entry.machine.HostKeyFingerprint = seenFingerprint
			}
		}

		m.updateStatus(machineID, models.MachineStatusConnected)
		log.Printf("SSH reconnected to machine %s after %d retries", machineID, attempt)
		if m.OnReconnect != nil {
			m.OnReconnect(machineID)
		}
		return
	}
}
