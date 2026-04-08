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

- Go 1.23+
- Node.js 22+
- A remote machine with SSH access (for actual functionality)

### Backend

```bash
cd backend

# Required: set a master key for credential encryption
export SPACEBALLONE_MASTER_KEY="your-secret-key-at-least-32-characters-long"
# Optional: set default admin credentials (otherwise random password is generated)
export DEFAULT_ADMIN_EMAIL="admin@spaceballone.local"
export DEFAULT_ADMIN_PASSWORD="changeme"

go mod download
go build ./...
go test ./...

# Run the server (default port 8080)
go run ./cmd/server
# Admin email and password will be printed to stdout on first run
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
# Edit .env -- at minimum set SPACEBALLONE_MASTER_KEY and DEFAULT_ADMIN_PASSWORD

docker compose up --build
# Frontend at http://localhost:3000
# API proxied through frontend at /api/*
# PostgreSQL on :5432 for dev access
```

## Deployment

### How it works

Docker Compose runs 3 services (Caddy optional):

```
Internet → Frontend (port 3000) → API (port 8080, internal)
            PostgreSQL (port 5432)
```

**Frontend** serves the TanStack Start app and proxies `/api/*` to the Go backend:

- Serves static assets and SSR pages from TanStack Start
- Routes `/api/*` (including WebSockets) to the Go backend
- Runs on port 3000 via custom Node.js server entry point

### Local development

```bash
docker compose up --build
# http://localhost:3000
```

### Production

```bash
# Set your admin credentials in .env
DEFAULT_ADMIN_EMAIL=admin@example.com
DEFAULT_ADMIN_PASSWORD=a-strong-password

docker compose up -d --build
# http://your-server:3000
```

For HTTPS, add Caddy or a reverse proxy in front of port 3000.

### Cloudflare Tunnel (optional)

Access SpaceBallOne from anywhere without opening ports or configuring DNS manually:

```bash
# 1. Create a tunnel in the Cloudflare dashboard (Zero Trust > Tunnels)
# 2. Set the tunnel token in .env
CLOUDFLARE_TUNNEL_TOKEN=your-token-here

# 3. Start with the tunnel profile
docker compose --profile tunnel up -d
```

The tunnel connects outbound to Cloudflare, so no open ports are needed. Configure the tunnel's public hostname in Cloudflare to point to `http://frontend:3000`.

### Non-Docker deployment

If running without Docker, the Go backend supports manual TLS:

```bash
export TLS_CERT_PATH=/path/to/cert.pem
export TLS_KEY_PATH=/path/to/key.pem
go run ./cmd/server  # Starts with HTTPS
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SPACEBALLONE_MASTER_KEY` | Yes | -- | AES-256 key for encrypting SSH credentials |
| `DEFAULT_ADMIN_EMAIL` | No | `admin@spaceballone.local` | Default admin email (set before first run) |
| `DEFAULT_ADMIN_PASSWORD` | No | random | Default admin password (set before first run) |
| `DOMAIN` | No | `localhost` | Domain for CORS/URLs |
| `DATABASE_URL` | No | PostgreSQL in compose | Database connection string |
| `PORT` | No | `8080` | API server port |
| `FRONTEND_URL` | No | `http://{DOMAIN}:3000` | Frontend origin for CORS |
| `SESSION_EXPIRY` | No | `24h` | Auth session lifetime |
| `HEARTBEAT_INTERVAL` | No | `30s` | SSH health check interval |
| `CLOUDFLARE_TUNNEL_TOKEN` | No | -- | Cloudflare Tunnel token (enable with `--profile tunnel`) |
| `TLS_CERT_PATH` | No | -- | Manual TLS cert (non-Docker only) |
| `TLS_KEY_PATH` | No | -- | Manual TLS key (non-Docker only) |

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
- SSH host keys are pinned with trust-on-first-use; the first connection still depends on trusting the target machine

## Contributing

This is an experiment. If you find the concept interesting:

1. Try running it against a real machine and report what breaks
2. Open issues for bugs you find
3. The spec at `docs/specs/` describes the full intended behavior

## Built With

- [Claude Code - Opus 4.6](https://claude.ai)
- [OpenCode - GLM 5.1](https://opencode.ai)
- [OpenAI GPT 5.4](https://openai.com)

## License

MIT
