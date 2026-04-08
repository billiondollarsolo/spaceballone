package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spaceballone/backend/internal/api"
	"github.com/spaceballone/backend/internal/auth"
	"github.com/spaceballone/backend/internal/browser"
	"github.com/spaceballone/backend/internal/crypto"
	"github.com/spaceballone/backend/internal/db"
	"github.com/spaceballone/backend/internal/models"
	"github.com/spaceballone/backend/internal/ports"
	sshmanager "github.com/spaceballone/backend/internal/ssh"
	"github.com/spaceballone/backend/internal/terminal"
	"github.com/spaceballone/backend/internal/ws"
	"golang.org/x/crypto/ssh"
)

func main() {
	// Validate master key — required for credential encryption
	if err := crypto.ValidateMasterKey(); err != nil {
		log.Fatalf("FATAL: %v — server cannot start without SPACEBALLONE_MASTER_KEY", err)
	}

	// Initialize database
	database, err := db.Init()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Ensure default admin user
	email, password, err := auth.EnsureDefaultAdmin(database)
	if err != nil {
		log.Fatalf("Failed to ensure default admin: %v", err)
	}
	if password != "" {
		fmt.Println("==========================================================")
		fmt.Println("  Default admin user created")
		fmt.Printf("  Email:    %s\n", email)
		fmt.Printf("  Password: %s\n", password)
		fmt.Println("  Please change this password on first login.")
		fmt.Println("==========================================================")
	}

	// Initialize WebSocket hub
	wsHub := ws.NewHub()

	// Initialize SSH manager with status change callback
	sshMgr := sshmanager.NewManager(database, func(machineID, status string) {
		wsHub.BroadcastStatus(machineID, status)
	})
	defer sshMgr.Stop()

	// Initialize browser manager
	brMgr := browser.NewManager()

	// Terminal manager (shared with router)
	termMgr := terminal.NewManager()

	// Port scanner manager
	portsMgr := ports.NewManager()

	// Wire browser lifecycle to inject/remove agent env vars in tmux sessions
	brMgr.OnBrowserStart = func(client *ssh.Client) {
		sessions, err := termMgr.ListTmuxSessions(client)
		if err != nil {
			return
		}
		for _, name := range sessions {
			if strings.HasPrefix(name, "sbo-") {
				_ = terminal.SetTmuxEnv(client, name, "CHROME_CDP_URL", "http://127.0.0.1:9222")
				_ = terminal.SetTmuxEnv(client, name, "BROWSERLESS_URL", "http://127.0.0.1:9222")
			}
		}
	}
	brMgr.OnBrowserStop = func(client *ssh.Client) {
		sessions, err := termMgr.ListTmuxSessions(client)
		if err != nil {
			return
		}
		for _, name := range sessions {
			if strings.HasPrefix(name, "sbo-") {
				_ = terminal.UnsetTmuxEnv(client, name, "CHROME_CDP_URL")
				_ = terminal.UnsetTmuxEnv(client, name, "BROWSERLESS_URL")
			}
		}
	}

	// Wire SSH reconnect callback to recover sessions
	sessionHandler := &api.SessionHandler{DB: database, SSH: sshMgr, Terminal: termMgr, Hub: wsHub}
	sshMgr.OnReconnect = func(machineID string) {
		sessionHandler.RecoverSessions(machineID)
	}

	// Wire SSH disconnect callback to clean up tunnels
	sshMgr.OnDisconnect = func(machineID string) {
		brMgr.CleanupMachine(machineID)
	}

	// Reset all machines to disconnected on startup (in-memory connections are lost)
	database.Model(&models.Machine{}).Where("status = ?", "connected").Update("status", models.MachineStatusDisconnected)

	// Create router with all dependencies
	router := api.NewRouterFromDeps(api.RouterDeps{
		DB:       database,
		SSH:      sshMgr,
		WS:       wsHub,
		Browser:  brMgr,
		Terminal: termMgr,
		Ports:    portsMgr,
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	certPath := os.Getenv("TLS_CERT_PATH")
	keyPath := os.Getenv("TLS_KEY_PATH")

	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // disabled for WebSocket and SSE streaming
		IdleTimeout:  120 * time.Second,
	}

	if certPath != "" && keyPath != "" {
		log.Printf("SpaceBallOne server starting on %s (TLS)", addr)
		if err := srv.ListenAndServeTLS(certPath, keyPath); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	} else {
		log.Printf("SpaceBallOne server starting on %s (plaintext HTTP)", addr)
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}
}
