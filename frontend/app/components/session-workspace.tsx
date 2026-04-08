import { useState } from 'react'
import { Terminal, Code2, Globe } from 'lucide-react'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '~/components/ui/tabs'
import { TerminalTabs } from '~/components/terminal-tabs'
import { CodeTab } from '~/components/code-tab'
import { BrowserTab } from '~/components/browser-tab'
import type { Session } from '~/lib/api'

interface SessionWorkspaceProps {
  session: Session
}

export function SessionWorkspace({ session }: SessionWorkspaceProps) {
  const [activeTab, setActiveTab] = useState('terminal')

  return (
    <div className="flex h-full flex-col">
      <Tabs value={activeTab} onValueChange={setActiveTab} className="flex h-full flex-col">
        <TabsList className="shrink-0">
          <TabsTrigger value="terminal" className="gap-1.5">
            <Terminal className="size-3.5" />
            Terminal
          </TabsTrigger>
          <TabsTrigger value="code" className="gap-1.5">
            <Code2 className="size-3.5" />
            Code
          </TabsTrigger>
          <TabsTrigger value="browser" className="gap-1.5">
            <Globe className="size-3.5" />
            Browser
          </TabsTrigger>
        </TabsList>

        <TabsContent value="terminal" className="flex-1 min-h-0 mt-0">
          <TerminalTabs
            sessionId={session.id}
            tabs={session.terminal_tabs ?? []}
          />
        </TabsContent>

        <TabsContent value="code" className="flex-1 min-h-0 mt-0">
          <CodeTab session={session} />
        </TabsContent>

        <TabsContent value="browser" className="flex-1 min-h-0 mt-0">
          <BrowserTab session={session} />
        </TabsContent>
      </Tabs>
    </div>
  )
}
