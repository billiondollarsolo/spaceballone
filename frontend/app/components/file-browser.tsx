import { useState } from 'react'
import { Folder, File, ChevronRight, Loader2 } from 'lucide-react'
import { Button } from '~/components/ui/button'
import { ScrollArea } from '~/components/ui/scroll-area'
import { useBrowseDirectory } from '~/lib/projects'

interface FileBrowserProps {
  machineId: string
  onSelect: (path: string) => void
  initialPath?: string
}

export function FileBrowser({ machineId, onSelect, initialPath = '/' }: FileBrowserProps) {
  const [currentPath, setCurrentPath] = useState(initialPath)
  const { data, isLoading, error } = useBrowseDirectory(machineId, currentPath)

  const pathParts = currentPath.split('/').filter(Boolean)

  function navigateTo(path: string) {
    setCurrentPath(path)
  }

  function navigateUp() {
    const parts = currentPath.split('/').filter(Boolean)
    parts.pop()
    setCurrentPath('/' + parts.join('/'))
  }

  function handleBreadcrumbClick(index: number) {
    const newPath = '/' + pathParts.slice(0, index + 1).join('/')
    setCurrentPath(newPath)
  }

  return (
    <div className="flex flex-col gap-3">
      {/* Breadcrumb */}
      <div className="flex items-center gap-1 text-sm">
        <button
          type="button"
          className="text-muted-foreground hover:text-foreground"
          onClick={() => setCurrentPath('/')}
        >
          /
        </button>
        {pathParts.map((part, i) => (
          <span key={i} className="flex items-center gap-1">
            <ChevronRight className="size-3 text-muted-foreground" />
            <button
              type="button"
              className="text-muted-foreground hover:text-foreground"
              onClick={() => handleBreadcrumbClick(i)}
            >
              {part}
            </button>
          </span>
        ))}
      </div>

      {/* Directory listing */}
      <ScrollArea className="h-[280px] rounded-md border">
        {isLoading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="size-5 animate-spin text-muted-foreground" />
          </div>
        ) : error ? (
          <div className="p-4 text-center text-sm text-destructive">
            Failed to browse directory
          </div>
        ) : (
          <div className="flex flex-col">
            {/* Go up entry */}
            {currentPath !== '/' && (
              <button
                type="button"
                className="flex items-center gap-2 px-3 py-1.5 text-sm hover:bg-accent text-left"
                onClick={navigateUp}
              >
                <Folder className="size-4 text-muted-foreground" />
                <span>..</span>
              </button>
            )}

            {data?.entries?.map((entry) => (
              <button
                key={entry.name}
                type="button"
                className="flex items-center gap-2 px-3 py-1.5 text-sm hover:bg-accent text-left"
                onClick={() => {
                  if (entry.type === 'dir') {
                    const sep = currentPath.endsWith('/') ? '' : '/'
                    navigateTo(`${currentPath}${sep}${entry.name}`)
                  }
                }}
              >
                {entry.type === 'dir' ? (
                  <Folder className="size-4 text-blue-500" />
                ) : (
                  <File className="size-4 text-muted-foreground" />
                )}
                <span className="flex-1 truncate">{entry.name}</span>
                {entry.type === 'file' && (
                  <span className="text-xs text-muted-foreground">
                    {formatSize(entry.size)}
                  </span>
                )}
              </button>
            ))}

            {data?.entries && data.entries.length === 0 && (
              <div className="p-4 text-center text-sm text-muted-foreground">
                Empty directory
              </div>
            )}
          </div>
        )}
      </ScrollArea>

      {/* Selected path and confirm */}
      <div className="flex items-center justify-between gap-2">
        <span className="truncate text-sm text-muted-foreground">
          Selected: <span className="font-mono">{currentPath}</span>
        </span>
        <Button type="button" size="sm" onClick={() => onSelect(currentPath)}>
          Select
        </Button>
      </div>
    </div>
  )
}

function formatSize(value: string): string {
  const bytes = Number(value)
  if (!Number.isFinite(bytes)) return value
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}
