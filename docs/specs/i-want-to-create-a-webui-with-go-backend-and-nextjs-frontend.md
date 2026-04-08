# SpaceBallOne — Feature Specification

> Remote development workspace manager with Go backend and TanStack Start frontend

**Created:** 2026-04-07
**Status:** Ready for Implementation

---

## Overview

SpaceBallOne is a web-based remote development workspace manager. It allows a developer to manage multiple remote machines via SSH, organize projects per machine, and work in persistent development sessions — each providing a terminal (xterm.js + tmux), code editor (code-server iframe), and browser (Browserless pixel stream). The entire system is accessible from any device (PC, tablet, phone) through a responsive web UI.

## Problem Statement

Developers working across multiple remote machines need a unified interface to manage SSH connections, coding sessions, and browser-based previews without juggling multiple terminal windows, SSH configs, and port forwards. SpaceBallOne centralizes this into a single web application.

---

## Scope

### In Scope (MVP)

- Machine management (CRUD with SSH key or password credentials)
- Guided machine setup wizard (auto-discover + assisted installation)
- Project management (CRUD, mapped to directories on remote machines)
- Remote file browser for directory selection
- Persistent terminal sessions (xterm.js + tmux, multiple terminals per session)
- Code-server integration (iframe, one shared instance per machine)
- Browserless Chrome container on remote machines (screencast WebSocket pixel stream)
- Coding agent support on remote machines (OpenCode, Claude Code, Codex via CDP)
- Single-user local auth (Argon2, server-side sessions, default admin with forced password change)
- Machine health monitoring (30s heartbeat, status indicators)
- Session auto-recovery after machine reboot (auto-recreate tmux + toast notification)
- Auto-retry SSH on connection drop (exponential backoff, silent reconnect)
- Dark/light mode
- Fully responsive design (hamburger drawer on mobile)
- Full header bar (logo, breadcrumb, global search, notifications, quick-connect, theme toggle, user menu)
- Global search across machines, projects, and sessions
- Notifications for connection events only
- Docker Compose deployment (separate containers)
- SQLite (dev) + PostgreSQL (prod) support
- TLS support
- Monorepo structure (`/backend`, `/frontend`, `/docker`, `/docs`)

### Out of Scope (MVP)

- Git UI integration (use terminal or code-server's built-in git)
- Team collaboration / shared sessions / pair programming
- Multi-user RBAC (roles, permissions, sharing)
- OAuth / SSO / external auth providers
- CPU/memory/disk monitoring dashboards for remote machines
- File upload/download UI (use terminal or code-server)

---

## Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────┐
│                    User's Browser                        │
│  ┌─────────────────────────────────────────────────┐    │
│  │         TanStack Start Frontend (Vite)           │    │
│  │  ┌──────┐ ┌──────────┐ ┌────────────────────┐  │    │
│  │  │xterm │ │code-svr  │ │ Browserless stream │  │    │
│  │  │.js   │ │iframe    │ │ (JPEG frames)      │  │    │
│  │  └──┬───┘ └────┬─────┘ └────────┬───────────┘  │    │
│  └─────┼──────────┼────────────────┼───────────────┘    │
└────────┼──────────┼────────────────┼────────────────────┘
         │WS        │HTTP(S)         │WS
         ▼          ▼                ▼
┌─────────────────────────────────────────────────────────┐
│              Go Backend (Chi + GORM)                     │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │
│  │ REST API │ │ WS Proxy │ │ SSH Mgr  │ │ Auth     │  │
│  │ (CRUD)   │ │(terminal)│ │(persistent│ │(Argon2  │  │
│  │          │ │          │ │connections│ │+sessions)│  │
│  └──────────┘ └──────────┘ └────┬─────┘ └──────────┘  │
│                                  │                      │
│  ┌──────────────────────────────┐│                      │
│  │ DB (SQLite/PostgreSQL)       ││                      │
│  │ via GORM                     ││                      │
│  └──────────────────────────────┘│                      │
└──────────────────────────────────┼──────────────────────┘
                                   │ SSH (persistent)
                                   ▼
┌─────────────────────────────────────────────────────────┐
│              Remote Machine                              │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │
│  │  tmux    │ │code-server│ │Browserless│ │ Coding   │  │
│  │ sessions │ │ :8443    │ │ :9222    │ │ Agents   │  │
│  │          │ │          │ │ (loopback)│ │(Claude,  │  │
│  │          │ │          │ │ (CDP)     │ │ OpenCode,│  │
│  │          │ │          │ │           │ │ Codex)   │  │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘  │
└─────────────────────────────────────────────────────────┘
```

### Key Architecture Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| SSH lifecycle | Continuous connections | Zero reconnect latency, real-time terminal proxying |
| Browser location | On remote machine | Localhost access to dev servers, low latency |
| Browser container | Browserless Docker image | Mature project, built-in screencast + CDP |
| Browser streaming | Screencast WebSocket (JPEG frames) | Lower bandwidth, simpler than noVNC |
| Agent location | On remote machine | Local CDP access to Browserless |
| Code-server | One per machine, iframe | Low overhead, full VS Code experience |
| API transport | REST + WebSocket hybrid | REST for CRUD, WS for real-time (one WS per terminal) |
| Credentials | AES-256 encrypted in DB | Master key via `SPACEBALLONE_MASTER_KEY` env var |
| Auth | Argon2 + server-side sessions | Simple, revocable, httpOnly cookies |
| Database | GORM (SQLite/PostgreSQL) | Dialect switching, fast development |
| Frontend | TanStack Start (Vite + Vinxi) | Full TanStack ecosystem (Router, Query, Table, Form, Virtual) |
| UI components | shadcn for Vite | Beautiful components, Tailwind-based, b3PCtM preset styles |
| Session layout | Fixed tabbed (Terminal/Code/Browser) | Mobile-friendly, simple |
| Mobile nav | Hamburger drawer | Standard pattern, works on all screen sizes |
| Health check | 30s heartbeat ping | Real-time status indicators |
| Connection failure | Auto-retry silently | Exponential backoff, yellow status indicator |
| Session recovery | Auto-recreate + notify | Recreate tmux sessions after reboot, toast notification |
| Machine setup | Guided assisted installation | Auto-discover, install core reqs, prompt for optional |
| Repo structure | Monorepo | `/backend`, `/frontend`, `/docker`, `/docs` |
| Deployment | Docker Compose, separate containers | Go API + TanStack Start + DB containers |

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.22+ |
| Router | go-chi/chi |
| ORM | GORM |
| Frontend Framework | TanStack Start (Vite + Vinxi) |
| Routing | TanStack Router |
| Data Fetching | TanStack Query |
| Tables | TanStack Table |
| Forms | TanStack Form |
| Virtualization | TanStack Virtual |
| UI Components | shadcn (Vite config, b3PCtM preset) |
| Styling | Tailwind CSS |
| Terminal | xterm.js |
| Remote Terminal | tmux |
| Code Editor | code-server (iframe) |
| Browser | Browserless (Docker) |
| Password Hashing | Argon2 |
| Credential Encryption | AES-256-GCM |
| Database (dev) | SQLite |
| Database (prod) | PostgreSQL |
| Deployment | Docker Compose |

---

## Data Model

### Users
| Field | Type | Notes |
|-------|------|-------|
| id | UUID | Primary key |
| username | string | Unique |
| password_hash | string | Argon2 hash |
| must_change_password | bool | True on first launch |
| created_at | timestamp | |
| updated_at | timestamp | |

### Machines
| Field | Type | Notes |
|-------|------|-------|
| id | UUID | Primary key |
| name | string | Display name |
| host | string | Hostname or IP |
| port | int | SSH port (default 22) |
| auth_type | enum | `key` or `password` |
| encrypted_credentials | blob | AES-256 encrypted SSH key or password |
| status | enum | `connected`, `disconnected`, `reconnecting`, `error` |
| capabilities | jsonb | Auto-discovered: `{docker: true, codeServer: true, ...}` |
| last_heartbeat | timestamp | Last successful health check |
| created_at | timestamp | |
| updated_at | timestamp | |

### Projects
| Field | Type | Notes |
|-------|------|-------|
| id | UUID | Primary key |
| machine_id | UUID | FK → Machines |
| name | string | Display name |
| directory_path | string | Absolute path on remote machine |
| created_at | timestamp | |
| updated_at | timestamp | |

### Sessions
| Field | Type | Notes |
|-------|------|-------|
| id | UUID | Primary key |
| project_id | UUID | FK → Projects |
| name | string | Auto-generated, user-renamable |
| status | enum | `active`, `idle`, `terminated` |
| last_active | timestamp | |
| created_at | timestamp | |
| updated_at | timestamp | |

### TerminalTabs
| Field | Type | Notes |
|-------|------|-------|
| id | UUID | Primary key |
| session_id | UUID | FK → Sessions |
| tmux_window_index | int | tmux window number |
| name | string | Tab display name |
| created_at | timestamp | |

### AppSessions (Auth)
| Field | Type | Notes |
|-------|------|-------|
| id | UUID | Primary key |
| user_id | UUID | FK → Users |
| session_token | string | Unique, httpOnly cookie value |
| expires_at | timestamp | |
| created_at | timestamp | |

---

## API Design

### Auth
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/auth/login` | Login with username/password |
| POST | `/api/auth/logout` | Invalidate session |
| POST | `/api/auth/change-password` | Change password (required on first login) |
| GET | `/api/auth/me` | Get current user info |

### Machines
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/machines` | List all machines |
| POST | `/api/machines` | Add a machine (SSH credentials) |
| GET | `/api/machines/:id` | Get machine details + status |
| PUT | `/api/machines/:id` | Update machine settings |
| DELETE | `/api/machines/:id` | Remove machine |
| POST | `/api/machines/:id/connect` | Establish SSH connection |
| POST | `/api/machines/:id/disconnect` | Close SSH connection |
| GET | `/api/machines/:id/capabilities` | Get auto-discovered capabilities |
| POST | `/api/machines/:id/setup` | Run guided setup/installation |
| GET | `/api/machines/:id/browse` | List remote directory contents (for file browser) |

### Projects
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/machines/:id/projects` | List projects for a machine |
| POST | `/api/machines/:id/projects` | Create project |
| GET | `/api/projects/:id` | Get project details |
| PUT | `/api/projects/:id` | Update project |
| DELETE | `/api/projects/:id` | Remove project |

### Sessions
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/projects/:id/sessions` | List sessions for a project |
| POST | `/api/projects/:id/sessions` | Create session (auto-created on project open) |
| GET | `/api/sessions/:id` | Get session details |
| PUT | `/api/sessions/:id` | Update session (rename) |
| DELETE | `/api/sessions/:id` | Terminate session |

### Terminal Tabs
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/sessions/:id/terminals` | Create new terminal tab (tmux window) |
| DELETE | `/api/terminals/:id` | Close terminal tab |

### WebSocket Endpoints
| Endpoint | Description |
|----------|-------------|
| `ws://…/api/ws/terminal/:terminalId` | Dedicated terminal I/O stream |
| `ws://…/api/ws/status` | Machine status updates (heartbeat, connection events) |
| `ws://…/api/ws/browser/:sessionId` | Browserless screencast frame stream + input events |

### Search
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/search?q=…` | Search machines, projects, sessions by name |

---

## User Stories

### US-1: Go API Scaffold
**Description:** As a developer, I want the Go backend scaffolded with Chi router, GORM, and basic project structure so I can start building endpoints.

**Acceptance Criteria:**
- [ ] `cmd/server/main.go` starts an HTTP server on configurable port
- [ ] Chi router configured with middleware (logging, recovery, CORS)
- [ ] GORM configured with SQLite driver
- [ ] All models defined with auto-migration
- [ ] Health check endpoint `GET /api/health` returns `200 {"status": "ok"}`
- [ ] `go build ./...` succeeds with no errors

### US-2: TanStack Start Frontend Scaffold
**Description:** As a developer, I want the TanStack Start frontend scaffolded with shadcn, Tailwind, and the app layout so I can start building pages.

**Acceptance Criteria:**
- [ ] TanStack Start app initializes and serves on configurable port
- [ ] TanStack Router configured with root layout
- [ ] shadcn components installed and configured for Vite
- [ ] Tailwind CSS configured with b3PCtM preset styles
- [ ] Dark/light mode toggle works (persisted to localStorage)
- [ ] Persistent header with logo placeholder and theme toggle renders
- [ ] Persistent sidebar with placeholder content renders
- [ ] Hamburger drawer works on mobile viewport (< 768px)
- [ ] `npm run build` succeeds

### US-3: Auth System
**Description:** As a user, I want to log in with username/password so that the app is protected.

**Acceptance Criteria:**
- [ ] Default admin user created on first server start (username: `admin`, random password logged to stdout)
- [ ] `POST /api/auth/login` returns session cookie on valid credentials, 401 on invalid
- [ ] `POST /api/auth/logout` invalidates the session
- [ ] `GET /api/auth/me` returns user info when authenticated, 401 when not
- [ ] All other API endpoints return 401 without valid session cookie
- [ ] Password stored as Argon2 hash in DB
- [ ] `must_change_password` flag forces password change before any other action
- [ ] `POST /api/auth/change-password` updates password and clears the flag
- [ ] Login page renders with username/password form
- [ ] Unauthenticated users are redirected to login page

### US-4: Machine CRUD
**Description:** As a user, I want to add, edit, and remove remote machines with SSH credentials.

**Acceptance Criteria:**
- [ ] `POST /api/machines` accepts name, host, port, auth_type, credentials
- [ ] SSH credentials encrypted with AES-256 before storage (using `SPACEBALLONE_MASTER_KEY`)
- [ ] `GET /api/machines` returns list (credentials excluded from response)
- [ ] `PUT /api/machines/:id` updates machine settings, re-encrypts if credentials changed
- [ ] `DELETE /api/machines/:id` removes machine and all associated projects/sessions
- [ ] Add Machine dialog in sidebar with form (name, host, port, key file upload or password)
- [ ] Machine list renders in sidebar with name and status dot
- [ ] Edit/delete actions accessible via machine context menu

### US-5: SSH Connection Manager
**Description:** As a user, I want to connect to a machine and see its real-time status.

**Acceptance Criteria:**
- [ ] `POST /api/machines/:id/connect` establishes persistent SSH connection
- [ ] Connection stored in-memory in the Go backend's connection pool
- [ ] 30-second heartbeat ping via SSH exec (`echo ok`)
- [ ] Status WebSocket (`/api/ws/status`) pushes machine status changes to frontend
- [ ] Sidebar shows green dot for connected, yellow for reconnecting, red for disconnected
- [ ] On connection drop, auto-retry with exponential backoff (1s, 2s, 4s, 8s, max 60s)
- [ ] Silent reconnection with yellow indicator during retry
- [ ] `POST /api/machines/:id/disconnect` cleanly closes SSH connection

### US-6: Project CRUD + Remote File Browser
**Description:** As a user, I want to create projects mapped to directories on remote machines, using a file browser to select the directory.

**Acceptance Criteria:**
- [ ] `GET /api/machines/:id/browse?path=/` returns directory listing via SSH (`ls -la`)
- [ ] Remote file browser component renders directory tree with navigation
- [ ] Clicking a directory navigates into it, clicking ".." goes up
- [ ] "Select" button confirms directory selection
- [ ] `POST /api/machines/:id/projects` creates project with name + selected directory
- [ ] Projects appear under their machine in the sidebar hierarchy
- [ ] `DELETE /api/projects/:id` removes project and associated sessions
- [ ] Project context menu with edit/delete actions

### US-7: Terminal Sessions (xterm.js + tmux)
**Description:** As a user, I want persistent terminal sessions that survive page reloads and reconnections.

**Acceptance Criteria:**
- [ ] Clicking a project auto-creates a session (or opens existing) with auto-generated name
- [ ] Session creates a tmux session on the remote machine in the project directory
- [ ] xterm.js renders in the Terminal tab with working input/output
- [ ] WebSocket (`/api/ws/terminal/:id`) proxies I/O between xterm.js and tmux via SSH
- [ ] Closing the browser tab and reopening reattaches to the same tmux session (output preserved)
- [ ] Multiple terminal tabs per session (tmux windows), rendered as sub-tabs
- [ ] "+" button creates new terminal tab (new tmux window)
- [ ] "x" button closes a terminal tab (kills tmux window)
- [ ] After machine reboot, tmux sessions are auto-recreated in the correct directory with a toast notification

### US-8: Code-Server Integration
**Description:** As a user, I want to edit code via code-server in the Code tab.

**Acceptance Criteria:**
- [ ] Backend manages code-server lifecycle on remote machine (start if not running)
- [ ] code-server runs on fixed port 8443 on remote machine
- [ ] SSH port tunnel forwards remote 8443 to a local port
- [ ] Code tab renders code-server in an iframe pointed at the tunneled port
- [ ] Switching projects in the sidebar updates the code-server workspace to the new project directory
- [ ] code-server authentication disabled (tunneled, not publicly exposed)

### US-9: Browserless Integration
**Description:** As a user, I want to see a browser preview of my running application in the Browser tab.

**Acceptance Criteria:**
- [ ] Backend manages Browserless Docker container on remote machine (start if not running)
- [ ] Browserless runs on loopback-only port 9222 (CDP over SSH tunnel)
- [ ] Backend proxies Browserless screencast WebSocket to frontend
- [ ] Browser tab renders JPEG frame stream from Browserless screencast
- [ ] User mouse clicks and keyboard input forwarded to Browserless via CDP input events
- [ ] URL bar at top of Browser tab allows navigating to different URLs
- [ ] Coding agents (OpenCode, Claude Code, Codex) on remote machine can connect to Browserless via localhost CDP

### US-10: Machine Setup Wizard
**Description:** As a user, I want guided assistance to install required and optional software on a new machine.

**Acceptance Criteria:**
- [ ] On first connect, backend runs discovery commands: `docker --version`, `which tmux`, `which code-server`, etc.
- [ ] Capabilities stored in machine's `capabilities` JSON field
- [ ] If core requirements missing (tmux), wizard prompts to install with one click
- [ ] Installation runs via SSH, progress streamed to a log panel in the wizard
- [ ] Optional items (code-server, Browserless, agents) presented with recommendations
- [ ] Missing optional items show warning about reduced functionality
- [ ] Supported agents: OpenCode, Claude Code, Codex — each with install button
- [ ] Wizard can be re-run from machine settings

### US-11: Header Bar
**Description:** As a user, I want a full-featured header bar for navigation and quick actions.

**Acceptance Criteria:**
- [ ] App logo/name on the left
- [ ] Breadcrumb showing current location (Machine > Project > Session)
- [ ] Global search input (searches machines, projects, sessions by name)
- [ ] Search results appear in a dropdown, clicking navigates to the item
- [ ] Notifications bell with unread count badge
- [ ] Notifications dropdown shows connection events (connected, disconnected, reconnecting)
- [ ] Quick-connect dropdown listing machines with one-click connect
- [ ] Dark/light mode toggle
- [ ] User menu dropdown (Settings, Change Password, Logout)

### US-12: Session Management
**Description:** As a user, I want to manage multiple sessions per project with auto-creation and renaming.

**Acceptance Criteria:**
- [ ] Clicking a project creates "Session 1" automatically if no sessions exist
- [ ] Subsequent sessions auto-named "Session 2", "Session 3", etc.
- [ ] Sessions can be renamed via double-click on the name or context menu
- [ ] Sessions appear under projects in the sidebar hierarchy
- [ ] Clicking a session opens the tabbed workspace (Terminal/Code/Browser)
- [ ] Session can be terminated via context menu (kills tmux session)
- [ ] Terminated sessions removed from sidebar

### US-13: Docker Compose Deployment
**Description:** As a developer, I want to deploy SpaceBallOne with Docker Compose.

**Acceptance Criteria:**
- [ ] `docker-compose.yml` at repo root with three services: `api`, `frontend`, `db`
- [ ] `api` service builds Go backend, exposes API port
- [ ] `frontend` service builds TanStack Start app, exposes web port
- [ ] `db` service runs PostgreSQL with persistent volume
- [ ] Environment variables: `SPACEBALLONE_MASTER_KEY`, `DATABASE_URL`, `TLS_CERT_PATH`, `TLS_KEY_PATH`
- [ ] `docker compose up` starts all services and the app is accessible
- [ ] TLS termination supported via env-configured cert/key paths
- [ ] `.env.example` documents all environment variables

### US-14: Responsive Polish
**Description:** As a user, I want the app to work well on tablets and phones.

**Acceptance Criteria:**
- [ ] Sidebar collapses to hamburger drawer on viewports < 768px
- [ ] Session tabs (Terminal/Code/Browser) stack appropriately on small screens
- [ ] Header adapts: search collapses to icon, items move to overflow menu
- [ ] Touch targets are at least 44x44px on mobile
- [ ] Terminal renders and accepts input on mobile (virtual keyboard)
- [ ] Code-server iframe is scrollable/zoomable on mobile
- [ ] Browser stream renders and accepts touch input on mobile

---

## User Flows

### First Launch
1. User runs `docker compose up` or starts the Go server
2. Server creates default admin account, logs random password to stdout
3. User opens browser, sees login page
4. User enters `admin` + logged password
5. Forced password change dialog appears
6. User sets new password
7. Redirected to empty dashboard with sidebar prompt: "Add your first machine"

### Adding a Machine
1. User clicks "+" in sidebar or "Add Machine" button
2. Dialog opens: name, host, port, auth type (key/password)
3. For key auth: file picker for private key
4. For password auth: password field
5. User clicks "Connect"
6. Backend establishes SSH, runs capability discovery
7. If core requirements missing → Setup Wizard opens
8. If optional items missing → recommendations shown
9. Machine appears in sidebar with green status dot

### Opening a Project Session
1. User clicks a project in sidebar
2. Session auto-created ("Session 1") or existing session opened
3. Tabbed workspace appears: Terminal (active), Code, Browser tabs
4. Terminal shows tmux session attached to project directory
5. User can switch tabs to Code (code-server iframe) or Browser (Browserless stream)
6. User can create additional terminal tabs via "+" button
7. User can create additional sessions via "New Session" button

### Connection Drop Recovery
1. SSH connection drops (network issue)
2. Sidebar machine dot turns yellow ("Reconnecting...")
3. Backend retries with exponential backoff (1s, 2s, 4s, 8s... max 60s)
4. On reconnect: dot turns green, terminal sessions reattach automatically
5. User sees no interruption if reconnect is fast

### Machine Reboot Recovery
1. Remote machine reboots
2. SSH connection drops, auto-retry begins
3. On reconnect, backend detects tmux sessions are gone
4. Backend auto-creates new tmux sessions in the same directories
5. Toast notification: "Machine [name] rebooted — sessions were recreated"
6. User sees fresh terminal in the correct directory

---

## Non-Functional Requirements

| ID | Requirement |
|----|-------------|
| NFR-1 | Terminal input latency < 100ms (backend to remote via SSH) |
| NFR-2 | Browser screencast frame rate ≥ 10 FPS at 720p |
| NFR-3 | SSH heartbeat every 30 seconds per connected machine |
| NFR-4 | Auto-reconnect within 60 seconds or show error state |
| NFR-5 | Credentials encrypted at rest with AES-256-GCM |
| NFR-6 | Passwords hashed with Argon2id |
| NFR-7 | All API endpoints require authentication (except login) |
| NFR-8 | Session cookies httpOnly, Secure (when TLS), SameSite=Strict |
| NFR-9 | Support 10+ concurrent SSH connections |
| NFR-10 | Frontend bundle < 500KB gzipped (excluding xterm.js) |
| NFR-11 | Docker Compose cold start < 60 seconds |

---

## Implementation Phases

### Phase 1: Foundation
**Goal:** Go API + TanStack Start scaffold + auth + DB + layout

**Tasks:**
- Go project scaffold (cmd/server, internal/*, Chi, GORM)
- All database models + auto-migration (SQLite)
- Auth system (Argon2, sessions, login/logout/change-password endpoints)
- TanStack Start scaffold (Router, shadcn, Tailwind, theme)
- App layout (header, sidebar, main content area)
- Login page + auth flow
- Dark/light mode toggle
- Auth middleware on all API routes

**User Stories:** US-1, US-2, US-3

**Verification:**
```bash
cd backend && go build ./... && go test ./...
cd frontend && npm run build && npm run typecheck
curl -X POST http://localhost:8080/api/auth/login -d '{"username":"admin","password":"..."}' # returns 200 + cookie
curl http://localhost:8080/api/health # returns 200
```

### Phase 2: Machine Management
**Goal:** SSH connections, machine CRUD, sidebar hierarchy, health monitoring

**Tasks:**
- Machine CRUD API endpoints
- AES-256 credential encryption/decryption
- SSH connection manager (persistent connections, connection pool)
- Health monitoring (30s heartbeat)
- Status WebSocket endpoint
- Auto-retry on connection drop (exponential backoff)
- Machine list in sidebar with status indicators
- Add/Edit/Delete machine dialogs
- TanStack Query integration for data fetching

**User Stories:** US-4, US-5

**Verification:**
```bash
# Add a machine and verify SSH connection
curl -X POST http://localhost:8080/api/machines -d '{"name":"test","host":"...","port":22,"auth_type":"password","credentials":"..."}' # returns 201
curl -X POST http://localhost:8080/api/machines/{id}/connect # returns 200
# Sidebar shows machine with green dot
# Kill SSH and verify yellow dot + auto-reconnect
```

### Phase 3: Projects & Terminal
**Goal:** Project CRUD, remote file browser, xterm.js terminal sessions

**Tasks:**
- Project CRUD API endpoints
- Remote directory browser API (SSH `ls`)
- Remote file browser UI component
- Session management (auto-create, rename, terminate)
- tmux session management via SSH
- xterm.js integration
- WebSocket terminal proxy (dedicated per session)
- Multiple terminal tabs (tmux windows)
- Session auto-recovery after reboot
- Project/session hierarchy in sidebar

**User Stories:** US-6, US-7, US-12

**Verification:**
```bash
# Create project, open session, verify terminal works
curl -X POST http://localhost:8080/api/machines/{id}/projects -d '{"name":"test","directory_path":"/home/user/project"}'
# Click project in sidebar → terminal opens in /home/user/project
# Type commands → output appears
# Close tab, reopen → same tmux session reattaches
# Create multiple terminal tabs → tmux windows created
```

### Phase 4: Code Server & Browser
**Goal:** code-server iframe, Browserless integration, tabbed workspace

**Tasks:**
- code-server lifecycle management on remote (start/stop via SSH)
- SSH port tunnel for code-server (8443)
- Code tab with code-server iframe
- Browserless container management on remote (Docker commands via SSH)
- Browserless screencast WebSocket proxy
- Browser tab with JPEG frame renderer + input forwarding
- URL bar in Browser tab
- Tabbed workspace UI (Terminal/Code/Browser)

**User Stories:** US-8, US-9

**Verification:**
```bash
# Open a session → switch to Code tab → code-server loads in iframe
# Switch to Browser tab → Browserless stream renders
# Navigate to localhost:3000 in browser tab → app preview loads
# Click/type in browser tab → input forwarded correctly
```

### Phase 5: Machine Setup Wizard
**Goal:** Guided installation, agent configuration, capability discovery

**Tasks:**
- Capability auto-discovery (run detection commands via SSH)
- Setup wizard UI (multi-step flow)
- Installation scripts for core requirements (tmux, Docker)
- Installation scripts for optional items (code-server, agents)
- Progress streaming during installation
- Agent configuration (OpenCode, Claude Code, Codex)
- Re-run wizard from machine settings

**User Stories:** US-10

**Verification:**
```bash
# Connect to a fresh machine → wizard auto-opens
# Missing tmux → prompt to install → click Install → tmux installed
# Optional items listed with recommendations
# Decline code-server → warning about missing Code tab functionality
# Re-run wizard from machine settings → same flow
```

### Phase 6: Docker Compose & Polish
**Goal:** Deployment, TLS, header bar, responsive design, final polish

**Tasks:**
- Dockerfile for Go API
- Dockerfile for TanStack Start frontend
- docker-compose.yml (api, frontend, db services)
- PostgreSQL support (GORM dialect switch)
- TLS configuration
- Full header bar (breadcrumb, search, notifications, quick-connect)
- Global search implementation
- Notification system (connection events)
- Responsive polish (hamburger drawer, mobile adaptations)
- `.env.example` with all configuration options

**User Stories:** US-11, US-13, US-14

**Verification:**
```bash
docker compose up --build
# App accessible at https://localhost
# Login works, add machine, create project, open terminal
# Resize to mobile → hamburger drawer works
# Search for machine → results appear
# Disconnect machine → notification appears
```

---

## Definition of Done

This feature is complete when:
- [ ] All 14 user stories pass their acceptance criteria
- [ ] All 6 implementation phases verified
- [ ] Go tests pass: `cd backend && go test ./...`
- [ ] Frontend builds: `cd frontend && npm run build`
- [ ] Frontend typechecks: `cd frontend && npm run typecheck`
- [ ] Docker Compose starts successfully: `docker compose up --build`
- [ ] App accessible via browser on desktop and mobile viewports
- [ ] Dark and light mode both render correctly
- [ ] Terminal sessions persist across page reloads
- [ ] SSH reconnection works after simulated network drop

---

## Repo Structure

```
spaceballone/
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go
│   ├── internal/
│   │   ├── api/          # HTTP handlers
│   │   ├── auth/         # Argon2, sessions
│   │   ├── crypto/       # AES-256 credential encryption
│   │   ├── db/           # GORM setup, migrations
│   │   ├── models/       # Data models
│   │   ├── ssh/          # SSH connection manager
│   │   ├── terminal/     # tmux session management
│   │   ├── browser/      # Browserless management
│   │   ├── codeserver/   # code-server lifecycle
│   │   ├── setup/        # Machine setup wizard logic
│   │   └── ws/           # WebSocket handlers
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── app/
│   │   ├── routes/       # TanStack Router routes
│   │   ├── components/   # React components
│   │   ├── lib/          # Utilities, API client
│   │   └── styles/       # Tailwind config, globals
│   ├── package.json
│   └── tsconfig.json
├── docker/
│   ├── Dockerfile.api
│   ├── Dockerfile.frontend
│   └── nginx.conf        # Optional reverse proxy
├── docs/
│   └── specs/
├── docker-compose.yml
├── .env.example
└── README.md
```

---

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SPACEBALLONE_MASTER_KEY` | AES-256 key for credential encryption | **Required** |
| `DATABASE_URL` | Database connection string | `sqlite://spaceballone.db` |
| `PORT` | API server port | `8080` |
| `FRONTEND_URL` | Frontend origin for CORS | `http://localhost:3000` |
| `TLS_CERT_PATH` | Path to TLS certificate | *(optional)* |
| `TLS_KEY_PATH` | Path to TLS private key | *(optional)* |
| `SESSION_EXPIRY` | Session lifetime | `24h` |
| `HEARTBEAT_INTERVAL` | Machine health check interval | `30s` |

---

## Open Questions

1. **Browserless version:** Which Browserless Docker image version/tag to pin?
2. **Machines without Docker:** Should we support a degraded mode (no Browser tab) for machines where Docker can't be installed?
3. **Session limits:** Should there be a max number of sessions per project? (Probably not for MVP)
4. **Agent installation:** Exact install scripts for OpenCode, Claude Code, Codex on various Linux distros
5. **code-server version:** Pin to specific version or use latest?

---

## Ralph Loop Command

```bash
/ralph-loop "Implement SpaceBallOne per spec at docs/specs/i-want-to-create-a-webui-with-go-backend-and-nextjs-frontend.md

PHASES:
1. Foundation: Go scaffold + TanStack Start scaffold + auth + DB + layout - verify with go build, npm run build, curl /api/health
2. Machine Management: SSH connections + CRUD + sidebar + health monitoring - verify with machine add/connect/status flow
3. Projects & Terminal: Project CRUD + file browser + xterm.js + tmux + sessions - verify with terminal I/O and session persistence
4. Code Server & Browser: code-server iframe + Browserless stream + tabbed workspace - verify with all three tabs functional
5. Machine Setup Wizard: Capability discovery + guided installation + agent config - verify with fresh machine setup flow
6. Docker Compose & Polish: Deployment + TLS + header + search + notifications + responsive - verify with docker compose up

VERIFICATION (run after each phase):
- cd backend && go build ./... && go test ./...
- cd frontend && npm run build && npm run typecheck
- Manual smoke test of new functionality

ESCAPE HATCH: After 20 iterations without progress:
- Document what's blocking in the spec file under 'Implementation Notes'
- List approaches attempted
- Stop and ask for human guidance

Output <promise>COMPLETE</promise> when all phases pass verification." --max-iterations 30 --completion-promise "COMPLETE"
```
