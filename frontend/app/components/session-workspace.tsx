import { useState, useCallback } from 'react'
import { Terminal, Globe, FolderTree, Loader2 } from 'lucide-react'
import { Panel, Group, Separator } from 'react-resizable-panels'
import { Button } from '~/components/ui/button'
import { TerminalTabs } from '~/components/terminal-tabs'
import { BrowserTab } from '~/components/browser-tab'
import { FileBrowserSidebar } from '~/components/file-browser-sidebar'
import { WorkspaceToolbar } from '~/components/workspace-toolbar'
import { useProject } from '~/lib/projects'
import type { Session } from '~/lib/api'

interface SessionWorkspaceProps {
  session: Session
}

export function SessionWorkspace({ session }: SessionWorkspaceProps) {
  const [showBrowser, setShowBrowser] = useState(false)
  const [showFiles, setShowFiles] = useState(false)
  const [browserUrl, setBrowserUrl] = useState<string | undefined>()

  const { data: project } = useProject(session.project_id)
  const machineId = project?.machine_id ?? ''
  const projectDir = project?.directory_path

  const handleToggleBrowser = useCallback(() => {
    setShowBrowser((prev) => !prev)
  }, [])

  const handleToggleFiles = useCallback(() => {
    setShowFiles((prev) => !prev)
  }, [])

  const handleNavigateToPort = useCallback((url: string) => {
    setShowBrowser(true)
    setBrowserUrl(url)
  }, [])

  return (
    <div className="flex h-full flex-col">
      <WorkspaceToolbar
        machineId={machineId}
        projectDir={projectDir}
        showBrowser={showBrowser}
        onToggleBrowser={handleToggleBrowser}
        showFiles={showFiles}
        onToggleFiles={handleToggleFiles}
        onNavigateToPort={handleNavigateToPort}
      />

      <div className="relative flex flex-1 min-h-0">
        <Group orientation="horizontal">
          <Panel defaultSize={showBrowser ? 55 : 100} minSize={30}>
            <TerminalTabs
              sessionId={session.id}
              tabs={session.terminal_tabs ?? []}
            />
          </Panel>

          {showBrowser && (
            <>
              <Separator className="w-px bg-border hover:bg-primary/50 transition-colors" />
              <Panel defaultSize={45} minSize={25}>
                <BrowserTab session={session} initialUrl={browserUrl} />
              </Panel>
            </>
          )}
        </Group>

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
