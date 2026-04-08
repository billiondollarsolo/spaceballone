import { useState, useCallback } from 'react'
import { FolderTree } from 'lucide-react'
import { TerminalTabs } from '~/components/terminal-tabs'
import { FileBrowserSidebar } from '~/components/file-browser-sidebar'
import { WorkspaceToolbar } from '~/components/workspace-toolbar'
import { useProject } from '~/lib/projects'
import type { Session } from '~/lib/api'

interface SessionWorkspaceProps {
  session: Session
}

export function SessionWorkspace({ session }: SessionWorkspaceProps) {
  const [showFiles, setShowFiles] = useState(false)

  const { data: project } = useProject(session.project_id)
  const machineId = project?.machine_id ?? ''
  const projectDir = project?.directory_path

  const handleToggleFiles = useCallback(() => {
    setShowFiles((prev) => !prev)
  }, [])

  return (
    <div className="flex h-full flex-col">
      <WorkspaceToolbar
        machineId={machineId}
        projectDir={projectDir}
        showFiles={showFiles}
        onToggleFiles={handleToggleFiles}
      />

      <div className="relative flex flex-1 min-h-0">
        <div className="flex-1 min-w-0">
          <TerminalTabs
            sessionId={session.id}
            tabs={session.terminal_tabs ?? []}
          />
        </div>

        {showFiles && machineId && (
          <FileBrowserSidebar
            machineId={machineId}
            projectDir={projectDir ?? '/'}
            onClose={() => setShowFiles(false)}
          />
        )}
      </div>
    </div>
  )
}
