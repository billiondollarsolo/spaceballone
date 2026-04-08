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
	if err := crypto.ValidateMasterKey(); err != nil {
		log.Fatalf("FATAL: %v — server cannot start without SPACEBALLONE_MASTER_KEY", err)
	}

	database, err := db.Init()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

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

	wsHub := ws.NewHub()

	sshMgr := sshmanager.NewManager(database, func(machineID, status string) {
		wsHub.BroadcastStatus(machineID, status)
	})
	defer sshMgr.Stop()

	termMgr := terminal.NewManager()
	portsMgr := ports.NewManager()

	injectChromiumEnv := func(client *ssh.Client) {
		sessions, err := termMgr.ListTmuxSessions(client)
		if err != nil {
			return
		}
		for _, name := range sessions {
			if strings.HasPrefix(name, "sbo-") {
				_ = terminal.SetTmuxEnv(client, name, "CHROMIUM_FLAGS", "--no-sandbox --disable-gpu --headless=new")
			}
		}
	}

	_ = injectChromiumEnv

	sessionHandler := &api.SessionHandler{DB: database, SSH: sshMgr, Terminal: termMgr, Hub: wsHub}
	sshMgr.OnReconnect = func(machineID string) {
		sessionHandler.RecoverSessions(machineID)
	}

	database.Model(&models.Machine{}).Where("status = ?", "connected").Update("status", models.MachineStatusDisconnected)

	router := api.NewRouterFromDeps(api.RouterDeps{
		DB:       database,
		SSH:      sshMgr,
		WS:       wsHub,
		Terminal: termMgr,
		Ports:    portsMgr,
	})

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
		WriteTimeout: 0,
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
