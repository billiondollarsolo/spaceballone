import { useState } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '~/components/ui/dialog'
import { Button } from '~/components/ui/button'
import { Input } from '~/components/ui/input'
import { Label } from '~/components/ui/label'
import { useUpdateProject } from '~/lib/projects'
import type { Project } from '~/lib/api'

interface EditProjectDialogProps {
  project: Project | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

function EditProjectDialogContent({
  project,
  onOpenChange,
}: {
  project: Project
  onOpenChange: (open: boolean) => void
}) {
  const [name, setName] = useState(project.name)
  const updateProject = useUpdateProject()

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    updateProject.mutate(
      { id: project.id, data: { name } },
      {
        onSuccess: () => {
          onOpenChange(false)
        },
      },
    )
  }

  return (
    <>
      <DialogHeader>
        <DialogTitle>Edit Project</DialogTitle>
        <DialogDescription>Update the project name.</DialogDescription>
      </DialogHeader>
      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="edit-project-name">Name</Label>
          <Input
            id="edit-project-name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
          />
        </div>
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
          >
            Cancel
          </Button>
          <Button type="submit" disabled={updateProject.isPending}>
            {updateProject.isPending ? 'Saving...' : 'Save'}
          </Button>
        </DialogFooter>
      </form>
    </>
  )
}

export function EditProjectDialog({
  project,
  open,
  onOpenChange,
}: EditProjectDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        {project && (
          <EditProjectDialogContent
            key={project.id}
            project={project}
            onOpenChange={onOpenChange}
          />
        )}
      </DialogContent>
    </Dialog>
  )
}
