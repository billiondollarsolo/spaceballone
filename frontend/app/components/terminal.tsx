import { useRef, useEffect, useState } from 'react'
import { Loader2, WifiOff } from 'lucide-react'
import { getWsUrl } from '~/lib/api'

interface TerminalProps {
  terminalId: string
}

type ConnectionState = 'connecting' | 'connected' | 'disconnected'

export function TerminalComponent({ terminalId }: TerminalProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const termRef = useRef<import('@xterm/xterm').Terminal | null>(null)
  const fitAddonRef = useRef<import('@xterm/addon-fit').FitAddon | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const [connState, setConnState] = useState<ConnectionState>('connecting')
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const reconnectCount = useRef(0)

  useEffect(() => {
    if (typeof window === 'undefined' || !containerRef.current) return

    let disposed = false

    async function init() {
      const { Terminal } = await import('@xterm/xterm')
      const { FitAddon } = await import('@xterm/addon-fit')
      const { WebLinksAddon } = await import('@xterm/addon-web-links')

      // Dynamically import the CSS
      await import('@xterm/xterm/css/xterm.css')

      if (disposed || !containerRef.current) return

      const term = new Terminal({
        cursorBlink: true,
        fontSize: 14,
        fontFamily: 'Menlo, Monaco, "Courier New", monospace',
        theme: {
          background: '#1a1b26',
          foreground: '#c0caf5',
          cursor: '#c0caf5',
        },
      })

      const fitAddon = new FitAddon()
      term.loadAddon(fitAddon)
      term.loadAddon(new WebLinksAddon())
      term.open(containerRef.current)
      fitAddon.fit()

      termRef.current = term
      fitAddonRef.current = fitAddon

      // Register terminal input handler once; it sends to the current wsRef
      term.onData((data) => {
        if (wsRef.current?.readyState === WebSocket.OPEN) {
          wsRef.current.send(data)
        }
      })

      function connect() {
        if (disposed) return
        setConnState('connecting')

        const ws = new WebSocket(getWsUrl(`/api/ws/terminal/${terminalId}`))
        wsRef.current = ws

        ws.onopen = () => {
          if (disposed) {
            ws.close()
            return
          }
          setConnState('connected')
          reconnectCount.current = 0

          // Send initial resize
          const dims = fitAddon.proposeDimensions()
          if (dims) {
            ws.send(
              JSON.stringify({
                type: 'resize',
                cols: dims.cols,
                rows: dims.rows,
              }),
            )
          }
        }

        ws.onmessage = (event) => {
          if (typeof event.data === 'string') {
            term.write(event.data)
          }
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
        const delay = Math.min(
          1000 * Math.pow(2, reconnectCount.current),
          15000,
        )
        reconnectCount.current++
        reconnectTimer.current = setTimeout(() => {
          reconnectTimer.current = null
          connect()
        }, delay)
      }

      // Handle resize
      const resizeObserver = new ResizeObserver(() => {
        if (disposed) return
        fitAddon.fit()
        const dims = fitAddon.proposeDimensions()
        if (dims && wsRef.current?.readyState === WebSocket.OPEN) {
          wsRef.current.send(
            JSON.stringify({
              type: 'resize',
              cols: dims.cols,
              rows: dims.rows,
            }),
          )
        }
      })
      resizeObserver.observe(containerRef.current!)

      connect()

      // Cleanup on return
      return () => {
        resizeObserver.disconnect()
      }
    }

    const cleanupPromise = init()

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
      termRef.current?.dispose()
      termRef.current = null
      fitAddonRef.current = null
      void cleanupPromise?.then((cleanup) => cleanup?.())
    }
  }, [terminalId])

  return (
    <div className="relative h-full w-full">
      <div ref={containerRef} className="h-full w-full" />

      {connState === 'connecting' && (
        <div className="absolute inset-0 flex items-center justify-center bg-black/70">
          <div className="flex items-center gap-2 text-sm text-white">
            <Loader2 className="size-4 animate-spin" />
            Connecting...
          </div>
        </div>
      )}

      {connState === 'disconnected' && (
        <div className="absolute inset-0 flex items-center justify-center bg-black/70">
          <div className="flex items-center gap-2 text-sm text-white">
            <WifiOff className="size-4" />
            Disconnected. Reconnecting...
          </div>
        </div>
      )}
    </div>
  )
}
