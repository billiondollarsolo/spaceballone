import { useMemo } from 'react'
import { Loader2, Code2, Power, PowerOff } from 'lucide-react'
import { Button } from '~/components/ui/button'
import { useCodeServerStatus, useStartCodeServer, useStopCodeServer } from '~/lib/code-server'
import type { Session } from '~/lib/api'
import { useProject } from '~/lib/projects'

interface CodeTabProps {
  session: Session
}

export function CodeTab({ session }: CodeTabProps) {
  const { data: project } = useProject(session.project_id)
  const machineId = project?.machine_id ?? ''
  const directoryPath = project?.directory_path ?? ''

  const { data: status, isLoading: statusLoading } = useCodeServerStatus(machineId)
  const startMutation = useStartCodeServer(machineId)
  const stopMutation = useStopCodeServer(machineId)

  const iframeSrc = useMemo(() => {
    if (!status?.running || !status.url) return null
    const url = new URL(status.url)
    if (directoryPath) {
      url.searchParams.set('folder', directoryPath)
    }
    return url.toString()
  }, [status?.running, status?.url, directoryPath])

  // If no machineId yet, show loading
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
          <Code2 className="mx-auto mb-3 size-10 text-muted-foreground" />
          <p className="mb-1 text-sm font-medium">Code Server</p>
          <p className="mb-4 max-w-xs text-xs text-muted-foreground">
            Launch a VS Code environment to edit files directly on the remote
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
                Start Code Server
              </>
            )}
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col">
      <div className="flex shrink-0 items-center justify-between border-b px-3 py-1.5">
        <span className="text-xs text-muted-foreground">
          {directoryPath || 'Code Server'}
        </span>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => stopMutation.mutate()}
          disabled={stopMutation.isPending}
        >
          {stopMutation.isPending ? (
            <Loader2 className="size-3.5 animate-spin" />
          ) : (
            <PowerOff className="size-3.5" />
          )}
          Stop
        </Button>
      </div>
      {iframeSrc && (
        <iframe
          src={iframeSrc}
          className="h-full w-full flex-1 border-0"
          allow="clipboard-read; clipboard-write; downloads; forms; scripts"
          sandbox="allow-scripts allow-same-origin allow-forms allow-popups allow-downloads"
          title="Code Server"
        />
      )}
    </div>
  )
}
