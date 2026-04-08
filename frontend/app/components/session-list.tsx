import { useState, useRef, useEffect } from 'react'
import { Link } from '@tanstack/react-router'
import { Terminal, MoreHorizontal, Plus } from 'lucide-react'
import { Button } from '~/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '~/components/ui/dropdown-menu'
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogCancel,
  AlertDialogAction,
} from '~/components/ui/alert-dialog'
import { Input } from '~/components/ui/input'
import {
  useSessions,
  useCreateSession,
  useDeleteSession,
  useUpdateSession,
} from '~/lib/sessions'
import type { Session } from '~/lib/api'

interface SessionListProps {
  projectId: string
}

export function SessionList({ projectId }: SessionListProps) {
  const { data: sessions, isLoading } = useSessions(projectId)
  const createSession = useCreateSession()

  if (isLoading) {
    return (
      <div className="pl-4 py-1">
        <span className="text-xs text-muted-foreground">Loading...</span>
      </div>
    )
  }

  return (
    <div className="flex flex-col">
      {sessions?.map((session) => (
        <SessionItem key={session.id} session={session} />
      ))}

      <button
        type="button"
        className="flex items-center gap-1.5 py-1 pl-7 pr-2 text-xs text-muted-foreground hover:text-foreground"
        onClick={() =>
          createSession.mutate({
            projectId,
            data: { name: `Session ${(sessions?.length ?? 0) + 1}` },
          })
        }
        disabled={createSession.isPending}
      >
        <Plus className="size-3" />
        New Session
      </button>
    </div>
  )
}

const sessionStatusColors: Record<string, string> = {
  active: 'bg-green-500',
  idle: 'bg-yellow-500',
  terminated: 'bg-gray-400',
}

function SessionItem({ session }: { session: Session }) {
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [renaming, setRenaming] = useState(false)
  const [renameName, setRenameName] = useState(session.name)
  const renameRef = useRef<HTMLInputElement>(null)
  const deleteSession = useDeleteSession()
  const updateSession = useUpdateSession()

  useEffect(() => {
    if (renaming && renameRef.current) {
      renameRef.current.focus()
      renameRef.current.select()
    }
  }, [renaming])

  function handleRenameSubmit() {
    if (renameName.trim() && renameName !== session.name) {
      updateSession.mutate({
        id: session.id,
        data: { name: renameName.trim() },
      })
    }
    setRenaming(false)
  }

  const statusColor = sessionStatusColors[session.status] ?? 'bg-gray-400'

  return (
    <>
      <div className="group flex items-center gap-1 py-1 pl-5 pr-2 hover:bg-sidebar-accent rounded-sm">
        {renaming ? (
          <Input
            ref={renameRef}
            value={renameName}
            onChange={(e) => setRenameName(e.target.value)}
            onBlur={handleRenameSubmit}
            onKeyDown={(e) => {
              if (e.key === 'Enter') handleRenameSubmit()
              if (e.key === 'Escape') {
                setRenameName(session.name)
                setRenaming(false)
              }
            }}
            className="h-5 px-1 text-xs"
          />
        ) : (
          <Link
            to="/sessions/$sessionId"
            params={{ sessionId: session.id }}
            className="flex min-w-0 flex-1 items-center gap-1.5"
            onDoubleClick={(e) => {
              e.preventDefault()
              setRenaming(true)
            }}
          >
            <Terminal className="size-3 shrink-0 text-muted-foreground" />
            <span className="truncate text-xs">{session.name}</span>
            <span
              className={`inline-block size-1.5 shrink-0 rounded-full ${statusColor}`}
              title={session.status}
            />
          </Link>
        )}

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="size-5 shrink-0 opacity-0 group-hover:opacity-100"
              aria-label={`Actions for ${session.name}`}
            >
              <MoreHorizontal className="size-3" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-32">
            <DropdownMenuItem onClick={() => setRenaming(true)}>
              Rename
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              className="text-destructive focus:text-destructive"
              onClick={() => setDeleteOpen(true)}
            >
              Terminate
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Terminate {session.name}?</AlertDialogTitle>
            <AlertDialogDescription>
              This will end the session and close all terminals.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-white hover:bg-destructive/90"
              onClick={() => deleteSession.mutate(session.id)}
            >
              {deleteSession.isPending ? 'Terminating...' : 'Terminate'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
