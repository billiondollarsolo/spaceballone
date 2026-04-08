package ports

import (
	"fmt"
	"strconv"
	"strings"

	sshmanager "github.com/spaceballone/backend/internal/ssh"
	"golang.org/x/crypto/ssh"
)

type DiscoveredPort struct {
	Port       int    `json:"port"`
	PID        int    `json:"pid"`
	Command    string `json:"command"`
	ProjectDir string `json:"project_dir,omitempty"`
	IsHTTP     bool   `json:"is_http"`
	URL        string `json:"url,omitempty"`
}

type Manager struct{}

func NewManager() *Manager {
	return &Manager{}
}

var ignoredPorts = map[int]bool{
	22:   true,
	9222: true,
}

func (m *Manager) ScanPorts(client *ssh.Client, projectDir string) ([]DiscoveredPort, error) {
	output, err := sshmanager.RunCommand(client, "ss -tlnp 2>/dev/null || netstat -tlnp 2>/dev/null")
	if err != nil {
		return nil, fmt.Errorf("failed to scan ports: %w", err)
	}

	rawPorts := parseSSOutput(output)
	var filtered []DiscoveredPort

	for _, p := range rawPorts {
		if ignoredPorts[p.Port] {
			continue
		}

		if p.PID > 0 {
			cwd, _ := sshmanager.RunCommand(client, fmt.Sprintf("readlink /proc/%d/cwd 2>/dev/null || echo", p.PID))
			cwd = strings.TrimSpace(cwd)
			p.ProjectDir = cwd

			if projectDir != "" && cwd != "" && !strings.HasPrefix(cwd, projectDir) {
				continue
			}

			cmdLine, _ := sshmanager.RunCommand(client, fmt.Sprintf("cat /proc/%d/cmdline 2>/dev/null | tr '\\0' ' '", p.PID))
			cmdLine = strings.TrimSpace(cmdLine)
			if cmdLine != "" {
				p.Command = cmdLine
			}
		}

		httpCode, _ := sshmanager.RunCommand(client, fmt.Sprintf(
			"curl -s -o /dev/null -w '%%{http_code}' -m 2 http://127.0.0.1:%d/ 2>/dev/null || echo 000", p.Port,
		))
		httpCode = strings.TrimSpace(httpCode)
		if httpCode != "" && httpCode != "000" {
			p.IsHTTP = true
			p.URL = fmt.Sprintf("http://127.0.0.1:%d", p.Port)
		}

		filtered = append(filtered, p)
	}

	return filtered, nil
}

func parseSSOutput(output string) []DiscoveredPort {
	var ports []DiscoveredPort
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "State") || strings.HasPrefix(line, "Proto") {
			continue
		}

		port, pid, cmd := parseSSLine(line)
		if port <= 0 {
			continue
		}

		ports = append(ports, DiscoveredPort{
			Port:    port,
			PID:     pid,
			Command: cmd,
		})
	}

	return ports
}

func parseSSLine(line string) (port int, pid int, cmd string) {
	usersIdx := strings.Index(line, "users:((")
	if usersIdx == -1 {
		return 0, 0, ""
	}

	rest := line[usersIdx:]
	parenStart := strings.Index(rest, "(\"")
	if parenStart == -1 {
		return 0, 0, ""
	}
	afterQuote := rest[parenStart+2:]
	commaIdx := strings.Index(afterQuote, "\"")
	if commaIdx > 0 {
		cmd = afterQuote[:commaIdx]
	}

	pidIdx := strings.Index(afterQuote, "pid=")
	if pidIdx != -1 {
		pidStr := afterQuote[pidIdx+4:]
		end := strings.IndexAny(pidStr, ",)")
		if end > 0 {
			pid, _ = strconv.Atoi(pidStr[:end])
		}
	}

	fields := strings.Fields(line)
	for _, f := range fields {
		if strings.Contains(f, ":") && !strings.Contains(f, "(") {
			parts := strings.Split(f, ":")
			if len(parts) >= 2 {
				p, err := strconv.Atoi(parts[len(parts)-1])
				if err == nil {
					port = p
					break
				}
			}
		}
	}

	return port, pid, cmd
}
