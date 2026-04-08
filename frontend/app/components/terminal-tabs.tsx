import { useState } from 'react'
import { Plus, X } from 'lucide-react'
import { Button } from '~/components/ui/button'
import { TerminalComponent } from '~/components/terminal'
import { useCreateTerminal, useDeleteTerminal } from '~/lib/sessions'
import type { TerminalTab } from '~/lib/api'

interface TerminalTabsProps {
  sessionId: string
  tabs: TerminalTab[]
}

export function TerminalTabs({ sessionId, tabs }: TerminalTabsProps) {
  const [activeTabId, setActiveTabId] = useState<string>(
    tabs[0]?.id ?? '',
  )
  const createTerminal = useCreateTerminal()
  const deleteTerminal = useDeleteTerminal()

  // Sync activeTabId if the active tab gets deleted
  const activeTab = tabs.find((t) => t.id === activeTabId) ?? tabs[0]

  function handleAddTab() {
    createTerminal.mutate(sessionId, {
      onSuccess: (newTab) => {
        setActiveTabId(newTab.id)
      },
    })
  }

  function handleCloseTab(tabId: string) {
    deleteTerminal.mutate(
      { terminalId: tabId, sessionId },
      {
        onSuccess: () => {
          if (activeTabId === tabId) {
            const remaining = tabs.filter((t) => t.id !== tabId)
            if (remaining.length > 0) {
              setActiveTabId(remaining[0].id)
            }
          }
        },
      },
    )
  }

  if (tabs.length === 0) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-center">
          <p className="mb-2 text-sm text-muted-foreground">
            No terminal tabs
          </p>
          <Button size="sm" onClick={handleAddTab} disabled={createTerminal.isPending}>
            <Plus className="mr-1 size-3" />
            New Terminal
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col">
      {/* Tab bar */}
      <div className="flex items-center gap-0.5 border-b bg-muted/50 px-1">
        {tabs.map((tab) => (
          <div
            key={tab.id}
            className={`group flex items-center gap-1 rounded-t-md px-3 py-1.5 text-xs cursor-pointer ${
              activeTab?.id === tab.id
                ? 'bg-background text-foreground border-b-2 border-b-primary'
                : 'text-muted-foreground hover:text-foreground'
            }`}
            onClick={() => setActiveTabId(tab.id)}
          >
            <span>{tab.name || `Terminal ${tab.tmux_window_index}`}</span>
            <button
              type="button"
              className="ml-1 size-3.5 rounded-sm opacity-0 hover:bg-destructive/20 group-hover:opacity-100"
              onClick={(e) => {
                e.stopPropagation()
                handleCloseTab(tab.id)
              }}
              aria-label={`Close ${tab.name}`}
            >
              <X className="size-3" />
            </button>
          </div>
        ))}
        <Button
          variant="ghost"
          size="icon"
          className="size-6 ml-1"
          onClick={handleAddTab}
          disabled={createTerminal.isPending}
          aria-label="New terminal tab"
        >
          <Plus className="size-3" />
        </Button>
      </div>

      {/* Active terminal */}
      <div className="flex-1 min-h-0">
        {activeTab && <TerminalComponent terminalId={activeTab.id} />}
      </div>
    </div>
  )
}
