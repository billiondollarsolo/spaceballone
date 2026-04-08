import type { QueryClient } from '@tanstack/react-query'
import { MACHINES_QUERY_KEY, machineQueryKey } from './machines'
import type { Machine } from './api'
import { getWsUrl as buildWsUrl } from './api'

interface MachineStatusMessage {
  type: 'machine_status'
  machine_id: string
  status: Machine['status']
}

type WSMessage = MachineStatusMessage

type StatusCallback = (machineId: string, status: Machine['status']) => void

let ws: WebSocket | null = null
let retryTimeout: ReturnType<typeof setTimeout> | null = null
let retryCount = 0
const MAX_RETRY_DELAY = 30000

export function connectStatusWebSocket(
  queryClient: QueryClient,
  onStatusChange?: StatusCallback,
): () => void {
  // Only run in browser
  if (typeof window === 'undefined') return () => {}

  function connect() {
    if (ws?.readyState === WebSocket.OPEN || ws?.readyState === WebSocket.CONNECTING) {
      return
    }

    try {
      ws = new WebSocket(buildWsUrl('/api/ws/status'))
    } catch {
      scheduleRetry()
      return
    }

    ws.onopen = () => {
      retryCount = 0
    }

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data as string) as WSMessage
        if (msg.type === 'machine_status') {
          // Update machine in the list cache
          queryClient.setQueryData<Machine[]>(
            MACHINES_QUERY_KEY,
            (old) => {
              if (!old) return old
              return old.map((m) =>
                m.id === msg.machine_id ? { ...m, status: msg.status } : m,
              )
            },
          )
          // Update individual machine cache
          queryClient.setQueryData<Machine>(
            machineQueryKey(msg.machine_id),
            (old) => {
              if (!old) return old
              return { ...old, status: msg.status }
            },
          )
          // Notify callback for notifications
          onStatusChange?.(msg.machine_id, msg.status)
        }
      } catch {
        // Ignore malformed messages
      }
    }

    ws.onclose = () => {
      ws = null
      scheduleRetry()
    }

    ws.onerror = () => {
      ws?.close()
    }
  }

  function scheduleRetry() {
    if (retryTimeout) return
    const delay = Math.min(1000 * Math.pow(2, retryCount), MAX_RETRY_DELAY)
    retryCount++
    retryTimeout = setTimeout(() => {
      retryTimeout = null
      connect()
    }, delay)
  }

  connect()

  // Return cleanup function
  return () => {
    if (retryTimeout) {
      clearTimeout(retryTimeout)
      retryTimeout = null
    }
    if (ws) {
      ws.onclose = null // Prevent retry on intentional close
      ws.close()
      ws = null
    }
    retryCount = 0
  }
}
