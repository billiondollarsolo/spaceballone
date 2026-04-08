import { useState, useCallback } from 'react'
import { X, FolderOpen, Folder, File, ChevronRight, ChevronDown, Loader2 } from 'lucide-react'
import { ScrollArea } from '~/components/ui/scroll-area'
import { useBrowseDirectory } from '~/lib/projects'
import type { DirectoryEntry } from '~/lib/api'

interface FileBrowserSidebarProps {
  machineId: string
  projectDir: string
  onClose: () => void
}

export function FileBrowserSidebar({ machineId, projectDir, onClose }: FileBrowserSidebarProps) {
  return (
    <div className="flex h-full w-72 flex-col border-l bg-background">
      <div className="flex shrink-0 items-center justify-between border-b px-3 py-2">
        <span className="text-sm font-medium">Files</span>
        <button
          type="button"
          onClick={onClose}
          className="size-6 rounded-sm hover:bg-accent flex items-center justify-center"
        >
          <X className="size-3.5" />
        </button>
      </div>
      <ScrollArea className="flex-1">
        <div className="p-2">
          <FileTreeNode
            name={projectDir.split('/').pop() || projectDir}
            path={projectDir}
            type="dir"
            machineId={machineId}
            depth={0}
            defaultExpanded
          />
        </div>
      </ScrollArea>
    </div>
  )
}

interface FileTreeNodeProps {
  name: string
  path: string
  type: 'file' | 'dir'
  machineId: string
  depth: number
  defaultExpanded?: boolean
}

function FileTreeNode({ name, path, type, machineId, depth, defaultExpanded }: FileTreeNodeProps) {
  const [expanded, setExpanded] = useState(defaultExpanded ?? false)

  const { data: dirData, isLoading } = useBrowseDirectory(
    expanded && type === 'dir' ? machineId : '',
    expanded && type === 'dir' ? path : '',
  )

  const entries = dirData?.entries

  const toggleExpand = useCallback(() => {
    if (type === 'dir') {
      setExpanded((prev) => !prev)
    }
  }, [type])

  return (
    <div>
      <button
        type="button"
        onClick={toggleExpand}
        className="flex w-full items-center gap-1 rounded-sm px-1 py-0.5 text-xs hover:bg-accent text-left"
        style={{ paddingLeft: `${depth * 12 + 4}px` }}
      >
        {type === 'dir' ? (
          <>
            {expanded ? (
              <ChevronDown className="size-3 shrink-0 text-muted-foreground" />
            ) : (
              <ChevronRight className="size-3 shrink-0 text-muted-foreground" />
            )}
            {expanded ? (
              <FolderOpen className="size-3.5 shrink-0 text-blue-500" />
            ) : (
              <Folder className="size-3.5 shrink-0 text-blue-500" />
            )}
          </>
        ) : (
          <>
            <span className="size-3 shrink-0" />
            <File className="size-3.5 shrink-0 text-muted-foreground" />
          </>
        )}
        <span className="truncate">{name}</span>
      </button>

      {type === 'dir' && expanded && (
        <>
          {isLoading && (
            <div style={{ paddingLeft: `${(depth + 1) * 12 + 4}px` }}>
              <Loader2 className="size-3 animate-spin text-muted-foreground" />
            </div>
          )}
          {entries && entries.length > 0 && (
            <div>
              {entries
                .filter((e) => e.name !== '..' && e.name !== '.')
                .sort((a, b) => {
                  if (a.type !== b.type) return a.type === 'dir' ? -1 : 1
                  return a.name.localeCompare(b.name)
                })
                .map((entry: DirectoryEntry) => (
                  <FileTreeNode
                    key={entry.name}
                    name={entry.name}
                    path={`${path === '/' ? '' : path}/${entry.name}`}
                    type={entry.type}
                    machineId={machineId}
                    depth={depth + 1}
                  />
                ))}
            </div>
          )}
          {entries && entries.length === 0 && !isLoading && (
            <div
              className="text-xs text-muted-foreground italic"
              style={{ paddingLeft: `${(depth + 1) * 12 + 4}px` }}
            >
              Empty
            </div>
          )}
        </>
      )}
    </div>
  )
}
