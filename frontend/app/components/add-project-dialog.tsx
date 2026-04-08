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
import { FileBrowser } from '~/components/file-browser'
import { useCreateProject } from '~/lib/projects'

interface AddProjectDialogProps {
  machineId: string
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function AddProjectDialog({
  machineId,
  open,
  onOpenChange,
}: AddProjectDialogProps) {
  const [name, setName] = useState('')
  const [directoryPath, setDirectoryPath] = useState('')
  const [showBrowser, setShowBrowser] = useState(false)

  const createProject = useCreateProject()

  function resetForm() {
    setName('')
    setDirectoryPath('')
    setShowBrowser(false)
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    createProject.mutate(
      {
        machine_id: machineId,
        name,
        directory_path: directoryPath,
      },
      {
        onSuccess: () => {
          resetForm()
          onOpenChange(false)
        },
      },
    )
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(value) => {
        if (!value) resetForm()
        onOpenChange(value)
      }}
    >
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Add Project</DialogTitle>
          <DialogDescription>
            Create a new project on this machine.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="project-name">Name</Label>
            <Input
              id="project-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="My Project"
              required
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="project-dir">Directory Path</Label>
            <div className="flex gap-2">
              <Input
                id="project-dir"
                value={directoryPath}
                onChange={(e) => setDirectoryPath(e.target.value)}
                placeholder="/home/user/project"
                required
                className="font-mono text-sm"
              />
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setShowBrowser(!showBrowser)}
              >
                Browse
              </Button>
            </div>
          </div>

          {showBrowser && (
            <FileBrowser
              machineId={machineId}
              initialPath={directoryPath || '/'}
              onSelect={(path) => {
                setDirectoryPath(path)
                setShowBrowser(false)
              }}
            />
          )}

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={createProject.isPending}>
              {createProject.isPending ? 'Creating...' : 'Create Project'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
