import { useEffect, useRef, useState, useCallback } from 'react'
import { Loader2, Globe, Power, PowerOff, WifiOff } from 'lucide-react'
import { Button } from '~/components/ui/button'
import { Input } from '~/components/ui/input'
import {
  useBrowserlessStatus,
  useStartBrowserless,
  useStopBrowserless,
} from '~/lib/browserless'
import { useProject } from '~/lib/projects'
import { getWsUrl } from '~/lib/api'
import type { Session } from '~/lib/api'

const STREAM_WIDTH = 1280
const STREAM_HEIGHT = 720

interface BrowserTabProps {
  session: Session
}

export function BrowserTab({ session }: BrowserTabProps) {
  const { data: project } = useProject(session.project_id)
  const machineId = project?.machine_id ?? ''

  const { data: status, isLoading: statusLoading } =
    useBrowserlessStatus(machineId)
  const startMutation = useStartBrowserless(machineId)
  const stopMutation = useStopBrowserless(machineId)

  if (!machineId) {
    return (
      <div className="flex h-full items-center justify-center">
        <Loader2 className="size-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (statusLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <Loader2 className="size-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (!status?.running) {
    return (
      <div className="flex h-full items-center justify-center rounded-lg border border-dashed">
        <div className="text-center">
          <Globe className="mx-auto mb-3 size-10 text-muted-foreground" />
          <p className="mb-1 text-sm font-medium">Browser Preview</p>
          <p className="mb-4 max-w-xs text-xs text-muted-foreground">
            Start a remote browser to preview web applications running on the
            machine.
          </p>
          <Button
            onClick={() => startMutation.mutate()}
            disabled={startMutation.isPending}
          >
            {startMutation.isPending ? (
              <>
                <Loader2 className="size-3.5 animate-spin" />
                Starting...
              </>
            ) : (
              <>
                <Power className="size-3.5" />
                Start Browser
              </>
            )}
          </Button>
        </div>
      </div>
    )
  }

  return <BrowserStream sessionId={session.id} onStop={() => stopMutation.mutate()} stopping={stopMutation.isPending} />
}

interface BrowserStreamProps {
  sessionId: string
  onStop: () => void
  stopping: boolean
}

type BrowserConnState = 'connecting' | 'connected' | 'disconnected'

function BrowserStream({ sessionId, onStop, stopping }: BrowserStreamProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const prevBlobUrlRef = useRef<string | null>(null)
  const imgRef = useRef<HTMLImageElement | null>(null)
  const lastMouseSentRef = useRef(0)
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const reconnectCount = useRef(0)
  const [urlInput, setUrlInput] = useState('https://')
  const [connState, setConnState] = useState<BrowserConnState>('connecting')

  // Initialize the Image object once
  useEffect(() => {
    imgRef.current = new Image()
    return () => {
      imgRef.current = null
    }
  }, [])

  // WebSocket connection with reconnection
  useEffect(() => {
    let disposed = false

    function connect() {
      if (disposed) return
      setConnState('connecting')

      const ws = new WebSocket(getWsUrl(`/api/ws/browser/${sessionId}`))
      ws.binaryType = 'arraybuffer'
      wsRef.current = ws

      ws.onopen = () => {
        if (disposed) {
          ws.close()
          return
        }
        setConnState('connected')
        reconnectCount.current = 0
      }

      ws.onmessage = (event) => {
        if (!(event.data instanceof ArrayBuffer)) return

        const blob = new Blob([event.data], { type: 'image/jpeg' })
        const blobUrl = URL.createObjectURL(blob)

        const img = imgRef.current
        if (!img) return

        img.onload = () => {
          const canvas = canvasRef.current
          if (!canvas) return
          const ctx = canvas.getContext('2d')
          if (!ctx) return
          ctx.drawImage(img, 0, 0, canvas.width, canvas.height)

          // Revoke previous blob URL to prevent memory leaks
          if (prevBlobUrlRef.current) {
            URL.revokeObjectURL(prevBlobUrlRef.current)
          }
          prevBlobUrlRef.current = blobUrl
        }

        img.src = blobUrl
      }

      ws.onclose = () => {
        if (disposed) return
        setConnState('disconnected')
        wsRef.current = null
        scheduleReconnect()
      }

      ws.onerror = () => {
        ws.close()
      }
    }

    function scheduleReconnect() {
      if (disposed) return
      const delay = Math.min(1000 * Math.pow(2, reconnectCount.current), 15000)
      reconnectCount.current++
      reconnectTimer.current = setTimeout(() => {
        reconnectTimer.current = null
        connect()
      }, delay)
    }

    connect()

    return () => {
      disposed = true
      if (reconnectTimer.current) {
        clearTimeout(reconnectTimer.current)
      }
      if (wsRef.current) {
        wsRef.current.onclose = null
        wsRef.current.close()
        wsRef.current = null
      }
      if (prevBlobUrlRef.current) {
        URL.revokeObjectURL(prevBlobUrlRef.current)
        prevBlobUrlRef.current = null
      }
    }
  }, [sessionId])

  const sendMessage = useCallback((msg: Record<string, unknown>) => {
    const ws = wsRef.current
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(msg))
    }
  }, [])

  const handleNavigate = useCallback(() => {
    if (urlInput.trim()) {
      sendMessage({ type: 'navigate', url: urlInput.trim() })
    }
  }, [urlInput, sendMessage])

  const getRelativeCoords = useCallback(
    (e: React.MouseEvent<HTMLCanvasElement>) => {
      const canvas = canvasRef.current
      if (!canvas) return null
      const rect = canvas.getBoundingClientRect()
      const x = Math.round(
        ((e.clientX - rect.left) / rect.width) * STREAM_WIDTH,
      )
      const y = Math.round(
        ((e.clientY - rect.top) / rect.height) * STREAM_HEIGHT,
      )
      return { x, y }
    },
    [],
  )

  const getRelativeCoordsFromTouch = useCallback(
    (touch: React.Touch) => {
      const canvas = canvasRef.current
      if (!canvas) return null
      const rect = canvas.getBoundingClientRect()
      const x = Math.round(
        ((touch.clientX - rect.left) / rect.width) * STREAM_WIDTH,
      )
      const y = Math.round(
        ((touch.clientY - rect.top) / rect.height) * STREAM_HEIGHT,
      )
      return { x, y }
    },
    [],
  )

  const handleMouseMove = useCallback(
    (e: React.MouseEvent<HTMLCanvasElement>) => {
      const now = Date.now()
      // Throttle to ~30fps
      if (now - lastMouseSentRef.current < 33) return
      lastMouseSentRef.current = now

      const coords = getRelativeCoords(e)
      if (coords) {
        sendMessage({ type: 'mousemove', ...coords })
      }
    },
    [getRelativeCoords, sendMessage],
  )

  const handleClick = useCallback(
    (e: React.MouseEvent<HTMLCanvasElement>) => {
      const coords = getRelativeCoords(e)
      if (coords) {
        sendMessage({ type: 'click', ...coords, button: 'left' })
      }
    },
    [getRelativeCoords, sendMessage],
  )

  const handleContextMenu = useCallback(
    (e: React.MouseEvent<HTMLCanvasElement>) => {
      e.preventDefault()
      const coords = getRelativeCoords(e)
      if (coords) {
        sendMessage({ type: 'click', ...coords, button: 'right' })
      }
    },
    [getRelativeCoords, sendMessage],
  )

  const handleWheel = useCallback(
    (e: React.WheelEvent<HTMLCanvasElement>) => {
      const coords = getRelativeCoords(e)
      if (coords) {
        sendMessage({
          type: 'scroll',
          ...coords,
          deltaX: Math.round(e.deltaX),
          deltaY: Math.round(e.deltaY),
        })
      }
    },
    [getRelativeCoords, sendMessage],
  )

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLCanvasElement>) => {
      e.preventDefault()
      const modifiers: string[] = []
      if (e.ctrlKey) modifiers.push('Control')
      if (e.shiftKey) modifiers.push('Shift')
      if (e.altKey) modifiers.push('Alt')
      if (e.metaKey) modifiers.push('Meta')
      sendMessage({ type: 'keydown', key: e.key, modifiers })
    },
    [sendMessage],
  )

  const handleKeyUp = useCallback(
    (e: React.KeyboardEvent<HTMLCanvasElement>) => {
      e.preventDefault()
      sendMessage({ type: 'keyup', key: e.key })
    },
    [sendMessage],
  )

  const handleTouchStart = useCallback(
    (e: React.TouchEvent<HTMLCanvasElement>) => {
      if (e.touches.length !== 1) return
      e.preventDefault()
      const coords = getRelativeCoordsFromTouch(e.touches[0])
      if (coords) {
        sendMessage({ type: 'mousemove', ...coords })
      }
    },
    [getRelativeCoordsFromTouch, sendMessage],
  )

  const handleTouchMove = useCallback(
    (e: React.TouchEvent<HTMLCanvasElement>) => {
      if (e.touches.length !== 1) return
      e.preventDefault()
      const now = Date.now()
      if (now - lastMouseSentRef.current < 33) return
      lastMouseSentRef.current = now
      const coords = getRelativeCoordsFromTouch(e.touches[0])
      if (coords) {
        sendMessage({ type: 'mousemove', ...coords })
      }
    },
    [getRelativeCoordsFromTouch, sendMessage],
  )

  const handleTouchEnd = useCallback(
    (e: React.TouchEvent<HTMLCanvasElement>) => {
      if (e.changedTouches.length !== 1) return
      e.preventDefault()
      const coords = getRelativeCoordsFromTouch(e.changedTouches[0])
      if (coords) {
        sendMessage({ type: 'click', ...coords, button: 'left' })
      }
    },
    [getRelativeCoordsFromTouch, sendMessage],
  )

  return (
    <div className="flex h-full flex-col">
      {/* Toolbar */}
      <div className="flex shrink-0 items-center gap-2 border-b px-3 py-1.5">
        <form
          className="flex flex-1 items-center gap-2"
          onSubmit={(e) => {
            e.preventDefault()
            handleNavigate()
          }}
        >
          <Input
            value={urlInput}
            onChange={(e) => setUrlInput(e.target.value)}
            placeholder="https://example.com"
            className="h-7 flex-1 text-xs"
          />
          <Button type="submit" size="sm" variant="secondary" className="h-7">
            Go
          </Button>
        </form>
        <Button
          variant="ghost"
          size="sm"
          className="h-7"
          onClick={onStop}
          disabled={stopping}
        >
          {stopping ? (
            <Loader2 className="size-3.5 animate-spin" />
          ) : (
            <PowerOff className="size-3.5" />
          )}
          Stop
        </Button>
      </div>

      {/* Canvas container - 16:9 aspect ratio, centered */}
      <div className="flex flex-1 items-center justify-center overflow-hidden bg-black/5 p-2">
        <canvas
          ref={canvasRef}
          width={STREAM_WIDTH}
          height={STREAM_HEIGHT}
          tabIndex={0}
          className="max-h-full max-w-full cursor-default outline-none"
          style={{ aspectRatio: '16 / 9' }}
          onMouseMove={handleMouseMove}
          onClick={handleClick}
          onContextMenu={handleContextMenu}
          onWheel={handleWheel}
          onKeyDown={handleKeyDown}
          onKeyUp={handleKeyUp}
          onTouchStart={handleTouchStart}
          onTouchMove={handleTouchMove}
          onTouchEnd={handleTouchEnd}
        />
      </div>

      {/* Connection status */}
      {connState === 'connecting' && (
        <div className="flex shrink-0 items-center justify-center gap-2 border-t bg-yellow-50 px-3 py-1 text-xs text-yellow-700 dark:bg-yellow-950 dark:text-yellow-300">
          <Loader2 className="size-3 animate-spin" />
          Connecting to browser stream...
        </div>
      )}
      {connState === 'disconnected' && (
        <div className="flex shrink-0 items-center justify-center gap-2 border-t bg-yellow-50 px-3 py-1 text-xs text-yellow-700 dark:bg-yellow-950 dark:text-yellow-300">
          <WifiOff className="size-3" />
          Disconnected. Reconnecting...
        </div>
      )}
    </div>
  )
}
