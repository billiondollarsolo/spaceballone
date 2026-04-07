# SpaceBallOne - Draft Specification

*Interview in progress - Started: 2026-04-07*

## Overview
Web UI with Go backend and TanStack Start frontend to manage multiple remote machines, projects, and development sessions from a single interface. Enables developers to manage coding projects across machines from any device (PC, tablet, phone).

## Architecture Decisions

### Frontend Framework
- **TanStack Start** (Vite + Vinxi based) — full commit, no fallback
- TanStack Router for routing
- TanStack Query for data fetching (Go API direct calls)
- TanStack Table for data grids/lists
- TanStack Form for forms
- TanStack Virtual for virtualized lists
- **shadcn components** configured for Vite (manually, not Next.js template)
- Apply b3PCtM preset styles manually

### SSH Connection Lifecycle
- **Continuous connections** — backend maintains persistent SSH connections
- Terminal sessions proxied in real-time via tmux on remote
- **Auto-retry silently** on connection drop (exponential backoff, subtle yellow status indicator)
- **Auto-recreate + notify** when machine reboots (recreate tmux sessions in same dir, show toast)

### Containerized Browser
- **Browserless Docker image** on remote machines
- Live pixel stream (VNC-like) to the frontend
- CDP access for coding agents
- Fixed ports: Chrome on 9222, VNC on 5900

### Coding Agents
- Run **remotely** on the same machine
- Supported: OpenCode, Claude Code, Codex
- Connect to Chrome locally via CDP

### Code Server
- **One shared code-server per machine** — embedded as iframe
- Fixed port: 8443
- Projects open as workspaces/folders

### API Transport
- **REST + WebSocket hybrid**
- REST for CRUD (machines, projects, users, settings)
- WebSocket for terminal I/O and status updates
- **Dedicated WebSocket per terminal session**

### Credential Storage
- **AES-256 encrypted in database**
- Master key via `SPACEBALLONE_MASTER_KEY` environment variable

### Auth
- **Single-user MVP** — default admin account created on first launch
- Forced password change on first login
- Passwords stored with **Argon2** hashing in DB
- **Server-side sessions** with httpOnly session cookie

### Database
- **GORM** ORM with dialect switching
- SQLite for local dev, PostgreSQL for production

### Go Backend
- **Standard layout** (cmd/server, internal/*)
- **go-chi/chi** router
- Go module: `spaceballone`

### Session Layout
- **Fixed tabbed layout** — Terminal, Code, Browser as tabs
- One visible at a time, tab bar at top

### Health Monitoring
- **Periodic heartbeat** every 30s
- Green/yellow/red status dots per machine

### Machine Setup
- **Guided assisted installation** on first connect
- Auto-discover capabilities, install core requirements with consent
- Prompt for optional items with recommendations

### Deployment
- **Docker Compose** with separate containers
- Go API + TanStack Start frontend + DB containers
- TLS support

### Port Strategy (Remote Machines)
- code-server: 8443
- Chrome/Browserless: 9222
- VNC stream: 5900
- All tunneled through SSH

## Tech Stack
- **Backend:** Go 1.22+ / Chi / GORM
- **Frontend:** TanStack Start (Vite + Vinxi) / shadcn / Tailwind
- **Data Fetching:** TanStack Query
- **Tables:** TanStack Table
- **Forms:** TanStack Form
- **Routing:** TanStack Router
- **Database:** SQLite (dev) / PostgreSQL (prod)
- **Terminal:** xterm.js + tmux
- **Browser:** Browserless Docker container (remote)
- **Code Editor:** code-server (iframe)
- **Auth:** Argon2 + server-side sessions
- **Deployment:** Docker Compose

## UI Requirements
- Dark and light mode
- Persistent header and sidebar
- Fully responsive (PC, tablet, phone)
- Machine/project hierarchy in sidebar
- Status indicators (green/yellow/red)
- Tabbed session workspace (Terminal/Code/Browser)

## Scope

### In Scope (MVP)
- Machine management (CRUD + SSH credentials)
- Guided machine setup wizard
- Project management (CRUD, mapped to directories)
- Persistent terminal sessions (xterm.js + tmux)
- Code-server integration (iframe)
- Browserless Chrome (pixel stream)
- Coding agent support (OpenCode, Claude Code, Codex)
- Single-user auth with forced password change
- Dark/light mode, responsive design
- Docker Compose deployment
- SQLite + PostgreSQL support
- TLS support
- Machine health monitoring
- Session auto-recovery after reboot

### Out of Scope (MVP)
- Git UI integration
- Team collaboration / shared sessions
- Multi-user RBAC
- OAuth / SSO

## Data Model

### Users
- id (uuid), username, password_hash (Argon2), must_change_password (bool), created_at, updated_at

### Machines
- id (uuid), name, host, port (int), auth_type (enum: key/password), encrypted_credentials (blob), status (enum), capabilities (jsonb), last_heartbeat, created_at, updated_at

### Projects
- id (uuid), machine_id (FK), name, directory_path, created_at, updated_at

### Sessions
- id (uuid), project_id (FK), type (enum: terminal/code/browser), tmux_session_name, status (enum), last_active, created_at, updated_at

### AppSessions
- id (uuid), user_id (FK), session_token (unique), expires_at, created_at

## Implementation Phases

### Phase 1: Foundation
- Go API scaffold (Chi, GORM, migrations)
- TanStack Start scaffold (Router, shadcn, Tailwind, dark/light mode)
- Auth system (Argon2, sessions, default admin, forced pw change)
- DB setup (SQLite + PostgreSQL support)
- Persistent header + sidebar layout

### Phase 2: Machine Management
- Machine CRUD API + UI
- SSH connection manager (persistent connections)
- Health monitoring (heartbeat)
- Sidebar machine list with status indicators
- Credential encryption/decryption

### Phase 3: Projects & Terminal
- Project CRUD API + UI (under machines)
- xterm.js integration
- tmux session management via SSH
- WebSocket terminal proxy
- Session persistence and auto-recovery

### Phase 4: Code Server & Browser
- code-server lifecycle management on remote
- iframe embedding with SSH port tunnel
- Browserless container management on remote
- Pixel stream integration
- Tabbed session workspace UI

### Phase 5: Machine Setup Wizard
- Capability auto-discovery
- Guided installation flows
- Agent configuration (OpenCode, Claude Code, Codex)
- Optional component prompts

### Phase 6: Docker Compose & Polish
- Docker Compose configuration
- TLS support
- Responsive polish (mobile/tablet)
- Final integration testing

## Open Questions
- Exact Browserless streaming protocol to frontend
- How to handle remote machines without Docker
- Playwright server deployment on remote
- Session limit per project?
