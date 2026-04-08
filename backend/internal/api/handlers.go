package api

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/spaceballone/backend/internal/auth"
	"github.com/spaceballone/backend/internal/browser"
	"github.com/spaceballone/backend/internal/codeserver"
	authmw "github.com/spaceballone/backend/internal/middleware"
	"github.com/spaceballone/backend/internal/models"
	"github.com/spaceballone/backend/internal/setup"
	sshmanager "github.com/spaceballone/backend/internal/ssh"
	"github.com/spaceballone/backend/internal/terminal"
	"github.com/spaceballone/backend/internal/ws"
	"gorm.io/gorm"
)

// RouterDeps holds all dependencies needed to construct the API router.
type RouterDeps struct {
	DB       *gorm.DB
	SSH      *sshmanager.Manager
	WS       *ws.Hub
	CS       *codeserver.Manager
	Browser  *browser.Manager
	Terminal *terminal.Manager
}

// NewRouter creates and configures the Chi router with all routes.
// It accepts a DB-only argument for backward compatibility (tests, simple setups).
func NewRouter(db *gorm.DB) *chi.Mux {
	return NewRouterFromDeps(RouterDeps{DB: db})
}

// NewRouterWithDeps creates the router with optional SSH manager and WS hub dependencies.
// Deprecated: prefer NewRouterFromDeps with a RouterDeps struct.
func NewRouterWithDeps(db *gorm.DB, sshMgr *sshmanager.Manager, wsHub *ws.Hub, csMgr *codeserver.Manager, brMgr *browser.Manager) *chi.Mux {
	return NewRouterFromDeps(RouterDeps{
		DB:      db,
		SSH:     sshMgr,
		WS:      wsHub,
		CS:      csMgr,
		Browser: brMgr,
	})
}

// NewRouterFromDeps creates the router from a RouterDeps struct.
func NewRouterFromDeps(deps RouterDeps) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{frontendURL},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Terminal manager — use the one from deps, or create a new one.
	termMgr := deps.Terminal
	if termMgr == nil {
		termMgr = terminal.NewManager()
	}

	// WebSocket endpoints (outside auth middleware — uses manual cookie/session auth)
	if deps.WS != nil {
		deps.WS.DB = deps.DB
		r.Get("/api/ws/status", deps.WS.HandleWebSocket)
	}
	if deps.SSH != nil && deps.WS != nil {
		th := &ws.TerminalHandler{DB: deps.DB, SSH: deps.SSH, Terminal: termMgr}
		r.Get("/api/ws/terminal/{terminalId}", th.HandleTerminalWS)

		if deps.Browser != nil {
			bh := &ws.BrowserHandler{DB: deps.DB, SSH: deps.SSH, Browser: deps.Browser}
			r.Get("/api/ws/browser/{sessionId}", bh.HandleBrowserWS)
		}
	}

	// Auth middleware for /api routes
	r.Route("/api", func(r chi.Router) {
		r.Use(authmw.AuthMiddleware(deps.DB))

		// Public (auth middleware skips these)
		r.Get("/health", healthHandler)
		r.Post("/auth/login", loginHandler(deps.DB))

		// Authenticated
		r.Post("/auth/logout", logoutHandler(deps.DB))
		r.Get("/auth/me", meHandler)
		r.Post("/auth/change-password", changePasswordHandler(deps.DB))

		// Machine CRUD (authenticated)
		if deps.SSH != nil {
			mh := &MachineHandler{DB: deps.DB, SSH: deps.SSH}
			r.Post("/machines", mh.CreateMachine)
			r.Get("/machines", mh.ListMachines)
			r.Get("/machines/{id}", mh.GetMachine)
			r.Put("/machines/{id}", mh.UpdateMachine)
			r.Delete("/machines/{id}", mh.DeleteMachine)
			r.Post("/machines/{id}/connect", mh.ConnectMachine)
			r.Post("/machines/{id}/disconnect", mh.DisconnectMachine)
			r.Get("/machines/{id}/capabilities", mh.GetCapabilities)

			// Project endpoints
			ph := &ProjectHandler{DB: deps.DB, SSH: deps.SSH, Terminal: termMgr}
			r.Get("/machines/{id}/projects", ph.ListProjects)
			r.Post("/machines/{id}/projects", ph.CreateProject)
			r.Get("/machines/{id}/browse", ph.BrowseDirectory)
			r.Get("/projects/{id}", ph.GetProject)
			r.Put("/projects/{id}", ph.UpdateProject)
			r.Delete("/projects/{id}", ph.DeleteProject)

			// Session endpoints
			sh := &SessionHandler{DB: deps.DB, SSH: deps.SSH, Terminal: termMgr, Hub: deps.WS}
			r.Get("/projects/{id}/sessions", sh.ListSessions)
			r.Post("/projects/{id}/sessions", sh.CreateSession)
			r.Get("/sessions/{id}", sh.GetSession)
			r.Put("/sessions/{id}", sh.UpdateSession)
			r.Delete("/sessions/{id}", sh.DeleteSession)
			r.Post("/sessions/{id}/terminals", sh.CreateTerminal)
			r.Delete("/terminals/{id}", sh.DeleteTerminal)

			// Code-server endpoints
			if deps.CS != nil {
				csh := &CodeServerHandler{DB: deps.DB, SSH: deps.SSH, CodeServer: deps.CS}
				r.Post("/machines/{id}/code-server/start", csh.StartCodeServer)
				r.Post("/machines/{id}/code-server/stop", csh.StopCodeServer)
				r.Get("/machines/{id}/code-server/status", csh.CodeServerStatus)
				r.Post("/machines/{id}/code-server/open", csh.OpenFolder)
				r.HandleFunc("/code-server-proxy/{machineId}/*", csh.ProxyCodeServer)
			}

			// Browserless endpoints
			if deps.Browser != nil {
				blh := &BrowserlessHandler{DB: deps.DB, SSH: deps.SSH, Browser: deps.Browser}
				r.Post("/machines/{id}/browserless/start", blh.StartBrowserless)
				r.Post("/machines/{id}/browserless/stop", blh.StopBrowserless)
				r.Get("/machines/{id}/browserless/status", blh.BrowserlessStatus)
			}

			// Setup wizard endpoints
			setupMgr := setup.NewManager(deps.DB, deps.SSH)
			sh2 := &SetupHandler{DB: deps.DB, SSH: deps.SSH, Setup: setupMgr}
			r.Post("/machines/{id}/setup/discover", sh2.Discover)
			r.Post("/machines/{id}/setup/install", sh2.Install)
			r.Get("/machines/{id}/setup/status", sh2.Status)

			// Search
			srch := &SearchHandler{DB: deps.DB}
			r.Get("/search", srch.Search)
		}
	})

	return r
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func isSecureRequest(r *http.Request) bool {
	return r.TLS != nil || os.Getenv("TLS_CERT_PATH") != "" || r.Header.Get("X-Forwarded-Proto") == "https"
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     authmw.SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   maxAge,
	})
}

func clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     authmw.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	ID                 string `json:"id"`
	Email              string `json:"email"`
	MustChangePassword bool   `json:"must_change_password"`
}

func loginHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		var user models.User
		if err := db.Where("email = ?", req.Email).First(&user).Error; err != nil {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}

		if !auth.VerifyPassword(req.Password, user.PasswordHash) {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}

		session, err := auth.CreateSession(db, user.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create session")
			return
		}

		setSessionCookie(w, r, session.SessionToken, int(auth.SessionExpiry().Seconds()))

		writeJSON(w, http.StatusOK, loginResponse{
			ID:                 user.ID,
			Email:              user.Email,
			MustChangePassword: user.MustChangePassword,
		})
	}
}

func logoutHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(authmw.SessionCookieName)
		if err == nil && cookie.Value != "" {
			_ = auth.InvalidateSession(db, cookie.Value)
		}

		clearSessionCookie(w, r)

		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func meHandler(w http.ResponseWriter, r *http.Request) {
	user := authmw.GetUser(r)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":                   user.ID,
		"email":                user.Email,
		"must_change_password": user.MustChangePassword,
	})
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func changePasswordHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := authmw.GetUser(r)
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var req changePasswordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.NewPassword == "" {
			writeError(w, http.StatusBadRequest, "new password is required")
			return
		}

		if !auth.VerifyPassword(req.CurrentPassword, user.PasswordHash) {
			writeError(w, http.StatusUnauthorized, "current password is incorrect")
			return
		}

		hash, err := auth.HashPassword(req.NewPassword)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to hash password")
			return
		}

		var session *models.AppSession
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(user).Updates(map[string]interface{}{
				"password_hash":        hash,
				"must_change_password": false,
			}).Error; err != nil {
				return err
			}

			if err := auth.InvalidateUserSessions(tx, user.ID); err != nil {
				return err
			}

			var err error
			session, err = auth.CreateSession(tx, user.ID)
			return err
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update password")
			return
		}

		setSessionCookie(w, r, session.SessionToken, int(auth.SessionExpiry().Seconds()))

		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
