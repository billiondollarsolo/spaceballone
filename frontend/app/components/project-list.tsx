import { useState, useEffect, useRef } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { FolderOpen, MoreHorizontal, Plus, Loader2 } from 'lucide-react'
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
import { AddProjectDialog } from '~/components/add-project-dialog'
import { EditProjectDialog } from '~/components/edit-project-dialog'
import { SessionList } from '~/components/session-list'
import { useProjects, useDeleteProject } from '~/lib/projects'
import { useCreateSession, useSessions } from '~/lib/sessions'
import type { Project } from '~/lib/api'

interface ProjectListProps {
  machineId: string
  isConnected: boolean
}

export function ProjectList({ machineId, isConnected }: ProjectListProps) {
  const { data: projects, isLoading } = useProjects(machineId)
  const [addOpen, setAddOpen] = useState(false)
  const [editingProject, setEditingProject] = useState<Project | null>(null)

  if (isLoading) {
    return (
      <div className="pl-4 py-1">
        <span className="text-xs text-muted-foreground">Loading...</span>
      </div>
    )
  }

  return (
    <div className="flex flex-col">
      {projects?.map((project) => (
        <ProjectItem
          key={project.id}
          project={project}
          onEdit={() => setEditingProject(project)}
        />
      ))}

      {isConnected && (
        <button
          type="button"
          className="flex items-center gap-1.5 py-1 pl-7 pr-2 text-xs text-muted-foreground hover:text-foreground"
          onClick={() => setAddOpen(true)}
        >
          <Plus className="size-3" />
          Add Project
        </button>
      )}

      <AddProjectDialog
        machineId={machineId}
        open={addOpen}
        onOpenChange={setAddOpen}
      />

      <EditProjectDialog
        project={editingProject}
        open={editingProject !== null}
        onOpenChange={(open) => {
          if (!open) setEditingProject(null)
        }}
      />
    </div>
  )
}

function ProjectItem({
  project,
  onEdit,
}: {
  project: Project
  onEdit: () => void
}) {
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [expanded, setExpanded] = useState(false)
  const deleteProject = useDeleteProject()
  const createSession = useCreateSession()
  const navigate = useNavigate()
  const { data: sessions } = useSessions(expanded ? project.id : '')

  function handleProjectClick() {
    if (!expanded) {
      setExpanded(true)
      // Auto-create session if none exist - check is done after sessions load
    } else {
      setExpanded(false)
    }
  }

  // When sessions load and the project was just expanded, auto-create if empty
  const autoCreateFired = useRef(false)

  useEffect(() => {
    if (
      sessions &&
      sessions.length === 0 &&
      expanded &&
      !createSession.isPending &&
      !createSession.isSuccess &&
      !autoCreateFired.current
    ) {
      autoCreateFired.current = true
      createSession.mutate(
        { projectId: project.id, data: { name: 'Session 1' } },
        {
          onSuccess: (session) => {
            void navigate({ to: '/sessions/$sessionId', params: { sessionId: session.id } })
          },
        },
      )
    }
  }, [sessions, expanded, createSession.isPending, createSession.isSuccess, project.id, navigate, createSession])

  // Reset the auto-create guard when the project is collapsed
  useEffect(() => {
    if (!expanded) {
      autoCreateFired.current = false
    }
  }, [expanded])

  return (
    <>
      <div className="group flex items-center gap-1 py-1 pl-5 pr-2 hover:bg-sidebar-accent rounded-sm">
        <button
          type="button"
          className="flex min-w-0 flex-1 items-center gap-1.5 text-left"
          onClick={handleProjectClick}
        >
          <FolderOpen className="size-3.5 shrink-0 text-muted-foreground" />
          <span className="truncate text-xs">{project.name}</span>
        </button>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="size-5 shrink-0 opacity-0 group-hover:opacity-100"
              aria-label={`Actions for ${project.name}`}
            >
              <MoreHorizontal className="size-3" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-32">
            <DropdownMenuItem onClick={onEdit}>Edit</DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              className="text-destructive focus:text-destructive"
              onClick={() => setDeleteOpen(true)}
            >
              Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      {expanded && (
        <div className="pl-3">
          {createSession.isPending ? (
            <div className="flex items-center gap-1 pl-5 py-1">
              <Loader2 className="size-3 animate-spin text-muted-foreground" />
              <span className="text-xs text-muted-foreground">Creating session...</span>
            </div>
          ) : (
            <SessionList projectId={project.id} />
          )}
        </div>
      )}

      <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete {project.name}?</AlertDialogTitle>
            <AlertDialogDescription>
              This will remove all associated sessions. This action cannot be
              undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-white hover:bg-destructive/90"
              onClick={() => deleteProject.mutate(project.id)}
            >
              {deleteProject.isPending ? 'Deleting...' : 'Delete'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
