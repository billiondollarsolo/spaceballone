import { Globe, FolderTree, Loader2, Wifi } from 'lucide-react'
import { Button } from '~/components/ui/button'
import { usePorts } from '~/lib/ports'
import { useBrowserlessStatus, useStartBrowserless } from '~/lib/browserless'
import type { DiscoveredPort } from '~/lib/api'

interface WorkspaceToolbarProps {
  machineId: string
  projectDir?: string
  showBrowser: boolean
  onToggleBrowser: () => void
  showFiles: boolean
  onToggleFiles: () => void
  onNavigateToPort: (url: string) => void
}

export function WorkspaceToolbar({
  machineId,
  projectDir,
  showBrowser,
  onToggleBrowser,
  showFiles,
  onToggleFiles,
  onNavigateToPort,
}: WorkspaceToolbarProps) {
  const { data: ports, isLoading: portsLoading } = usePorts(machineId, projectDir)
  const { data: browserStatus } = useBrowserlessStatus(machineId)
  const startBrowserless = useStartBrowserless(machineId)

  function handlePortClick(port: DiscoveredPort) {
    if (!browserStatus?.running) {
      startBrowserless.mutate(undefined, {
        onSuccess: () => {
          setTimeout(() => onNavigateToPort(port.url ?? `http://127.0.0.1:${port.port}`), 2000)
        },
      })
    } else {
      onNavigateToPort(port.url ?? `http://127.0.0.1:${port.port}`)
    }
  }

  return (
    <div className="flex shrink-0 items-center gap-2 border-b bg-muted/30 px-3 py-1.5">
      <div className="flex items-center gap-1">
        <Button
          variant="ghost"
          size="sm"
          className={`h-7 gap-1.5 text-xs ${showFiles ? 'bg-accent' : ''}`}
          onClick={onToggleFiles}
        >
          <FolderTree className="size-3.5" />
          Files
        </Button>
        <Button
          variant="ghost"
          size="sm"
          className={`h-7 gap-1.5 text-xs ${showBrowser ? 'bg-accent' : ''}`}
          onClick={onToggleBrowser}
        >
          <Globe className="size-3.5" />
          Browser
        </Button>
      </div>

      <div className="mx-2 h-4 w-px bg-border" />

      <div className="flex items-center gap-1.5 overflow-x-auto">
        {portsLoading && (
          <Loader2 className="size-3 animate-spin text-muted-foreground" />
        )}
        {!portsLoading && ports && ports.length === 0 && (
          <span className="text-xs text-muted-foreground">No services detected</span>
        )}
        {ports?.map((port) => (
          <button
            key={port.port}
            type="button"
            onClick={() => handlePortClick(port)}
            className="flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs hover:bg-accent transition-colors"
          >
            <span className={`size-1.5 rounded-full ${port.is_http ? 'bg-green-500' : 'bg-gray-400'}`} />
            <span className="font-mono">:{port.port}</span>
            <span className="text-muted-foreground">
              {extractCommandName(port.command)}
            </span>
          </button>
        ))}
      </div>
    </div>
  )
}

function extractCommandName(cmd: string): string {
  if (!cmd) return ''
  const parts = cmd.trim().split(/\s+/)
  const last = parts[0].split('/').pop() ?? parts[0]
  return last.length > 12 ? last.slice(0, 12) + '...' : last
}
