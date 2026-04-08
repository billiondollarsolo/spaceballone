import { FolderTree, ExternalLink, Loader2 } from 'lucide-react'
import { Button } from '~/components/ui/button'
import { usePorts } from '~/lib/ports'
import type { DiscoveredPort } from '~/lib/api'

interface WorkspaceToolbarProps {
  machineId: string
  projectDir?: string
  showFiles: boolean
  onToggleFiles: () => void
}

export function WorkspaceToolbar({
  machineId,
  projectDir,
  showFiles,
  onToggleFiles,
}: WorkspaceToolbarProps) {
  const { data: ports, isLoading: portsLoading } = usePorts(machineId, projectDir)

  function handlePortClick(port: DiscoveredPort) {
    const url = `/api/proxy/${machineId}/${port.port}/`
    window.open(url, '_blank', 'noopener')
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
            className="flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs hover:bg-accent transition-colors"
          >
            <span className={`size-1.5 rounded-full ${port.is_http ? 'bg-green-500' : 'bg-gray-400'}`} />
            <span className="font-mono">:{port.port}</span>
            <span className="text-muted-foreground">
              {extractCommandName(port.command)}
            </span>
            <ExternalLink className="size-2.5 text-muted-foreground" />
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
