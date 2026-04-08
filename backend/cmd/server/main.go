package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/spaceballone/backend/internal/api"
	"github.com/spaceballone/backend/internal/auth"
	"github.com/spaceballone/backend/internal/browser"
	"github.com/spaceballone/backend/internal/codeserver"
	"github.com/spaceballone/backend/internal/crypto"
	"github.com/spaceballone/backend/internal/db"
	sshmanager "github.com/spaceballone/backend/internal/ssh"
	"github.com/spaceballone/backend/internal/terminal"
	"github.com/spaceballone/backend/internal/ws"
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
	password, err := auth.EnsureDefaultAdmin(database)
	if err != nil {
		log.Fatalf("Failed to ensure default admin: %v", err)
	}
	if password != "" {
		fmt.Println("==========================================================")
		fmt.Println("  Default admin user created")
		fmt.Printf("  Username: admin\n")
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

	// Initialize code-server and browser managers
	csMgr := codeserver.NewManager()
	brMgr := browser.NewManager()

	// Terminal manager (shared with router)
	termMgr := terminal.NewManager()

	// Wire SSH reconnect callback to recover sessions
	sessionHandler := &api.SessionHandler{DB: database, SSH: sshMgr, Terminal: termMgr, Hub: wsHub}
	sshMgr.OnReconnect = func(machineID string) {
		sessionHandler.RecoverSessions(machineID)
	}

	// Wire SSH disconnect callback to clean up tunnels
	sshMgr.OnDisconnect = func(machineID string) {
		csMgr.CleanupMachine(machineID)
		brMgr.CleanupMachine(machineID)
	}

	// Create router with all dependencies
	router := api.NewRouterFromDeps(api.RouterDeps{
		DB:       database,
		SSH:      sshMgr,
		WS:       wsHub,
		CS:       csMgr,
		Browser:  brMgr,
		Terminal: termMgr,
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
