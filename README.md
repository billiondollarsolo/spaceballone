# SpaceBallOne

**A web-based remote development workspace manager.**

> **Status: Pre-alpha / Building in Public**
>
> This project is brand new, completely untested against real infrastructure, and absolutely not ready for production use. The concept itself may not work as envisioned. We're building in public because the idea is interesting, not because the software is ready. **Use at your own risk.** Expect breaking changes, missing features, and rough edges everywhere.

---

## What is this?

SpaceBallOne aims to be a single web UI for managing multiple remote development machines. Instead of juggling SSH terminals, VS Code remote sessions, and port forwards across different machines, you'd open one browser tab and get:

- **Terminal** -- persistent tmux sessions via xterm.js that survive page reloads and reconnections
- **Code Editor** -- code-server (VS Code in the browser) embedded as an iframe
- **Browser Preview** -- a remote Chromium instance (Browserless) streamed to your browser for previewing running apps

All proxied through a Go backend that manages SSH connections to your remote machines.

### The idea

```
Your Browser
    |
    v
SpaceBallOne (Go API + TanStack Start frontend)
    |
    v  (SSH)
Remote Machine 1 -- tmux, code-server, Browserless
Remote Machine 2 -- tmux, code-server, Browserless
Remote Machine N -- ...
```

You add machines (with SSH credentials), create projects (mapped to directories), and open sessions that give you a tabbed workspace with Terminal, Code, and Browser tabs. The backend maintains persistent SSH connections with automatic reconnection.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.23, Chi router, GORM (SQLite/PostgreSQL) |
| Frontend | TanStack Start (Vite 7), TanStack Router/Query, React 19 |
| UI | shadcn/ui, Tailwind CSS v4, dark/light mode |
| Terminal | xterm.js + tmux over WebSocket |
| Code Editor | code-server (iframe, SSH-tunneled) |
| Browser | Browserless Chrome (CDP screencast over WebSocket) |
| Auth | Argon2id + server-side sessions, AES-256-GCM credential encryption |
| Deployment | Docker Compose (Go API + frontend + PostgreSQL) |

## Project Structure

```
spaceballone/
  backend/           Go API server
    cmd/server/        Entry point
    internal/
      api/             HTTP handlers
      auth/            Argon2id hashing, sessions
      crypto/          AES-256-GCM credential encryption
      db/              GORM setup (SQLite/PostgreSQL)
      models/          Data models
      middleware/       Auth middleware
      ssh/             SSH connection manager + health monitoring
      terminal/        tmux session management
      codeserver/      code-server lifecycle + SSH tunneling
      browser/         Browserless management + CDP client
      setup/           Machine capability discovery + installation
      ws/              WebSocket handlers (status, terminal, browser)
  frontend/          TanStack Start app
    app/
      routes/          File-based routing
      components/      React components + shadcn/ui
      lib/             API client, hooks, utilities
      styles/          Tailwind + theme
  docker/            Dockerfiles
  docs/specs/        Feature specification
```

## Getting Started (Development)

### Prerequisites

- Go 1.22+
- Node.js 20+
- A remote machine with SSH access (for actual functionality)

### Backend

```bash
cd backend

# Required: set a master key for credential encryption
export SPACEBALLONE_MASTER_KEY="your-secret-key-at-least-32-characters-long"

go mod download
go build ./...
go test ./...

# Run the server (default port 8080)
go run ./cmd/server
# Admin password will be printed to stdout on first run
```

### Frontend

```bash
cd frontend
npm install
npm run dev
# Opens on http://localhost:3000
```

### Docker Compose

```bash
cp .env.example .env
# Edit .env -- at minimum set SPACEBALLONE_MASTER_KEY

docker compose up --build
# API on :8080, Frontend on :3000, PostgreSQL on :5432
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SPACEBALLONE_MASTER_KEY` | Yes | -- | AES-256 key for encrypting SSH credentials |
| `DATABASE_URL` | No | `sqlite://spaceballone.db` | Database connection (prefix with `postgres://` for PostgreSQL) |
| `PORT` | No | `8080` | API server port |
| `FRONTEND_URL` | No | `http://localhost:3000` | Frontend origin for CORS |
| `SESSION_EXPIRY` | No | `24h` | Auth session lifetime |
| `HEARTBEAT_INTERVAL` | No | `30s` | SSH health check interval |
| `TLS_CERT_PATH` | No | -- | Path to TLS certificate (enables HTTPS) |
| `TLS_KEY_PATH` | No | -- | Path to TLS private key |

## Features (Planned/In Progress)

- [x] Go API scaffold with Chi + GORM
- [x] Auth system (Argon2id, sessions, forced password change)
- [x] Machine CRUD with encrypted credentials
- [x] SSH connection manager with heartbeat + auto-reconnect
- [x] Project management with remote file browser
- [x] Terminal sessions (xterm.js + tmux + WebSocket proxy)
- [x] Code-server integration (iframe + SSH tunnel)
- [x] Browserless integration (CDP screencast + input forwarding)
- [x] Machine setup wizard (capability discovery + guided install)
- [x] Global search, notifications, quick-connect
- [x] Dark/light mode, responsive design
- [x] Docker Compose deployment
- [ ] Actually tested against real remote machines
- [ ] End-to-end integration testing
- [ ] Error recovery in real-world network conditions
- [ ] Performance profiling under load
- [ ] Security audit

## Honest Assessment

This entire codebase was generated in a single session. Here's what that means:

**What exists:** ~12,600 lines of Go and TypeScript implementing the full architecture described in the spec. The code compiles, type-checks, and passes unit tests. The patterns are reasonable (proper auth middleware, encrypted credentials, WebSocket proxying, etc.).

**What hasn't happened:**
- Nobody has connected this to a real remote machine
- The SSH tunneling for code-server and Browserless is untested against actual services
- The CDP screencast proxy is theoretical -- it may need significant debugging
- The tmux session management works in unit tests but hasn't been validated end-to-end
- Docker builds haven't been run against real containers
- We don't know if the UX actually makes sense in practice

**Known risks:**
- The Browserless Docker image version/port mapping may need adjustment
- SSH tunneling edge cases (dropped connections mid-tunnel, port conflicts) are not battle-tested
- The single-user auth model is intentionally simple for MVP
- `ssh.InsecureIgnoreHostKey()` is used -- no host key verification

## Contributing

This is an experiment. If you find the concept interesting:

1. Try running it against a real machine and report what breaks
2. Open issues for bugs you find
3. The spec at `docs/specs/` describes the full intended behavior

## License

MIT
