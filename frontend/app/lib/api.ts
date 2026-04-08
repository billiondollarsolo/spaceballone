// eslint-disable-next-line @typescript-eslint/no-explicit-any
const _process = typeof globalThis !== 'undefined' ? (globalThis as any).process : undefined
export const API_URL = typeof window !== 'undefined'
  ? ''  // Browser: use same-origin relative paths (Caddy proxies /api/* to backend)
  : (_process?.env?.API_URL || 'http://localhost:8080')  // SSR: internal Docker URL

export function getWsUrl(path: string): string {
  if (typeof window === 'undefined') {
    // SSR context — shouldn't create WebSockets, but just in case
    return `ws://localhost:8080${path}`
  }
  // Browser: derive WS URL from current page location (same origin)
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}${path}`
}

export class ApiError extends Error {
  constructor(
    public status: number,
    public statusText: string,
    public body?: unknown,
  ) {
    super(`API Error: ${status} ${statusText}`)
    this.name = 'ApiError'
  }
}

async function request<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const url = `${API_URL}${path}`

  const res = await fetch(url, {
    ...options,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
  })

  if (!res.ok) {
    let body: unknown
    try {
      body = await res.json()
    } catch {
      // ignore parse errors
    }
    throw new ApiError(res.status, res.statusText, body)
  }

  if (res.status === 204) {
    return undefined as T
  }

  return res.json() as Promise<T>
}

export const api = {
  get<T>(path: string, options?: { signal?: AbortSignal }) {
    return request<T>(path, { method: 'GET', signal: options?.signal })
  },
  post<T>(path: string, body?: unknown) {
    return request<T>(path, {
      method: 'POST',
      body: body ? JSON.stringify(body) : undefined,
    })
  },
  put<T>(path: string, body?: unknown) {
    return request<T>(path, {
      method: 'PUT',
      body: body ? JSON.stringify(body) : undefined,
    })
  },
  delete<T>(path: string) {
    return request<T>(path, { method: 'DELETE' })
  },
}

// Machine types
export interface Machine {
  id: string
  name: string
  host: string
  port: number
  auth_type: 'password' | 'key'
  status: 'connected' | 'disconnected' | 'reconnecting' | 'error'
  capabilities: Record<string, boolean> | null
  last_heartbeat: string | null
  created_at: string
  updated_at: string
}

export interface CreateMachineInput {
  name: string
  host: string
  port: number
  auth_type: 'password' | 'key'
  credentials: string
}

export interface UpdateMachineInput {
  name?: string
  host?: string
  port?: number
  auth_type?: 'password' | 'key'
  credentials?: string
}

// Machine API
export const machineApi = {
  list() {
    return api.get<Machine[]>('/api/machines')
  },
  get(id: string) {
    return api.get<Machine>(`/api/machines/${id}`)
  },
  create(data: CreateMachineInput) {
    return api.post<Machine>('/api/machines', data)
  },
  update(id: string, data: UpdateMachineInput) {
    return api.put<Machine>(`/api/machines/${id}`, data)
  },
  delete(id: string) {
    return api.delete<void>(`/api/machines/${id}`)
  },
  connect(id: string) {
    return api.post<Machine>(`/api/machines/${id}/connect`)
  },
  disconnect(id: string) {
    return api.post<Machine>(`/api/machines/${id}/disconnect`)
  },
}

// Project types
export interface Project {
  id: string
  machine_id: string
  name: string
  directory_path: string
  created_at: string
  updated_at: string
}

export interface CreateProjectInput {
  machine_id: string
  name: string
  directory_path: string
}

export interface UpdateProjectInput {
  name?: string
  directory_path?: string
}

export interface DirectoryEntry {
  name: string
  type: 'file' | 'dir'
  size: string
  modified: string
  permissions: string
}

export interface BrowseDirectoryResponse {
  path: string
  entries: DirectoryEntry[]
}

// Project API
export const projectApi = {
  list(machineId: string) {
    return api.get<Project[]>(`/api/machines/${machineId}/projects`)
  },
  get(id: string) {
    return api.get<Project>(`/api/projects/${id}`)
  },
  create(data: CreateProjectInput) {
    return api.post<Project>(`/api/machines/${data.machine_id}/projects`, data)
  },
  update(id: string, data: UpdateProjectInput) {
    return api.put<Project>(`/api/projects/${id}`, data)
  },
  delete(id: string) {
    return api.delete<void>(`/api/projects/${id}`)
  },
}

export function browseDirectory(machineId: string, path: string) {
  return api.get<BrowseDirectoryResponse>(
    `/api/machines/${machineId}/browse?path=${encodeURIComponent(path)}`,
  )
}

// Session types
export interface Session {
  id: string
  project_id: string
  name: string
  status: string
  last_active: string
  created_at: string
  updated_at: string
  terminal_tabs?: TerminalTab[]
}

export interface TerminalTab {
  id: string
  session_id: string
  tmux_window_index: number
  name: string
  created_at: string
}

export interface CreateSessionInput {
  name?: string
}

export interface UpdateSessionInput {
  name?: string
  status?: string
}

// Session API
export const sessionApi = {
  list(projectId: string) {
    return api.get<Session[]>(`/api/projects/${projectId}/sessions`)
  },
  get(id: string) {
    return api.get<Session>(`/api/sessions/${id}`)
  },
  create(projectId: string, data?: CreateSessionInput) {
    return api.post<Session>(`/api/projects/${projectId}/sessions`, data)
  },
  update(id: string, data: UpdateSessionInput) {
    return api.put<Session>(`/api/sessions/${id}`, data)
  },
  delete(id: string) {
    return api.delete<void>(`/api/sessions/${id}`)
  },
}

// Terminal API
export const terminalApi = {
  create(sessionId: string) {
    return api.post<TerminalTab>(`/api/sessions/${sessionId}/terminals`)
  },
  delete(terminalId: string) {
    return api.delete<void>(`/api/terminals/${terminalId}`)
  },
}

// Code-Server types
export interface CodeServerStatus {
  running: boolean
  url: string | null
}

// Code-Server API
export const codeServerApi = {
  start(machineId: string) {
    return api.post<{ url: string }>(`/api/machines/${machineId}/code-server/start`)
  },
  stop(machineId: string) {
    return api.post<void>(`/api/machines/${machineId}/code-server/stop`)
  },
  status(machineId: string) {
    return api.get<CodeServerStatus>(`/api/machines/${machineId}/code-server/status`)
  },
  open(machineId: string, folder: string) {
    return api.post<{ url: string }>(`/api/machines/${machineId}/code-server/open`, { folder })
  },
}

// Browserless types
export interface BrowserlessStatus {
  running: boolean
}

// Browserless API
export const browserlessApi = {
  start(machineId: string) {
    return api.post<void>(`/api/machines/${machineId}/browserless/start`)
  },
  stop(machineId: string) {
    return api.post<void>(`/api/machines/${machineId}/browserless/stop`)
  },
  status(machineId: string) {
    return api.get<BrowserlessStatus>(`/api/machines/${machineId}/browserless/status`)
  },
}

// Setup types
export interface SetupCapabilities {
  tmux: boolean
  tmux_version?: string
  docker: boolean
  docker_version?: string
  code_server: boolean
  code_server_version?: string
  node: boolean
  node_version?: string
  go_lang: boolean
  go_version?: string
  claude_code: boolean
  opencode: boolean
  codex: boolean
}

export interface SetupRecommendation {
  package: string
  reason: string
  required: boolean
  description: string
}

export interface SetupStatus {
  capabilities: SetupCapabilities
  recommendations: SetupRecommendation[]
}

export interface SSEMessage {
  line: string
  done: boolean
  error?: boolean
}

// Setup API
export const setupApi = {
  discover(machineId: string) {
    return api.post<SetupCapabilities>(`/api/machines/${machineId}/setup/discover`)
  },
  install(machineId: string, packageName: string): EventSource {
    // We use a custom fetch-based SSE since POST is needed
    // Return a pseudo EventSource-like object
    const url = `${API_URL}/api/machines/${machineId}/setup/install`
    const eventSource = new EventTarget() as EventTarget & { close: () => void; _controller?: AbortController }
    const controller = new AbortController()
    eventSource._controller = controller
    eventSource.close = () => controller.abort()

    fetch(url, {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ package: packageName }),
      signal: controller.signal,
    }).then(async (res) => {
      if (!res.ok || !res.body) {
        eventSource.dispatchEvent(new CustomEvent('error', { detail: `HTTP ${res.status}` }))
        return
      }
      const reader = res.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n\n')
        buffer = lines.pop() || ''
        for (const chunk of lines) {
          const dataLine = chunk.trim()
          if (dataLine.startsWith('data: ')) {
            const jsonStr = dataLine.slice(6)
            try {
              const msg = JSON.parse(jsonStr) as SSEMessage
              eventSource.dispatchEvent(new CustomEvent('message', { detail: msg }))
            } catch {
              // skip malformed JSON
            }
          }
        }
      }
    }).catch((err) => {
      if (err.name !== 'AbortError') {
        eventSource.dispatchEvent(new CustomEvent('error', { detail: err }))
      }
    })

    return eventSource as unknown as EventSource
  },
  status(machineId: string) {
    return api.get<SetupStatus>(`/api/machines/${machineId}/setup/status`)
  },
}

// Search types are in ~/lib/search

// Auth types
export interface User {
  id: string
  username: string
  must_change_password: boolean
  created_at: string
  updated_at: string
}

export interface LoginRequest {
  username: string
  password: string
}

export interface ChangePasswordRequest {
  current_password: string
  new_password: string
}

// Auth API
export const authApi = {
  me() {
    return api.get<User>('/api/auth/me')
  },
  login(data: LoginRequest) {
    return api.post<User>('/api/auth/login', data)
  },
  logout() {
    return api.post<void>('/api/auth/logout')
  },
  changePassword(data: ChangePasswordRequest) {
    return api.post<void>('/api/auth/change-password', data)
  },
}
