import { useState } from 'react'
import {
  ContextMenu,
  ContextMenuTrigger,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
} from '~/components/ui/context-menu'
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
import {
  useDeleteMachine,
  useConnectMachine,
  useDisconnectMachine,
} from '~/lib/machines'
import type { Machine } from '~/lib/api'

interface MachineContextMenuProps {
  machine: Machine
  onEdit: () => void
  children: React.ReactNode
}

export function MachineContextMenu({
  machine,
  onEdit,
  children,
}: MachineContextMenuProps) {
  const [deleteOpen, setDeleteOpen] = useState(false)
  const deleteMachine = useDeleteMachine()
  const connectMachine = useConnectMachine()
  const disconnectMachine = useDisconnectMachine()

  const isConnected = machine.status === 'connected'

  return (
    <>
      <ContextMenu>
        <ContextMenuTrigger asChild>{children}</ContextMenuTrigger>
        <ContextMenuContent>
          <ContextMenuItem onClick={onEdit}>Edit</ContextMenuItem>
          <ContextMenuSeparator />
          {isConnected ? (
            <ContextMenuItem
              onClick={() => disconnectMachine.mutate(machine.id)}
            >
              Disconnect
            </ContextMenuItem>
          ) : (
            <ContextMenuItem
              onClick={() => connectMachine.mutate(machine.id)}
            >
              Connect
            </ContextMenuItem>
          )}
          <ContextMenuSeparator />
          <ContextMenuItem
            className="text-destructive focus:text-destructive"
            onClick={() => setDeleteOpen(true)}
          >
            Delete
          </ContextMenuItem>
        </ContextMenuContent>
      </ContextMenu>

      <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete {machine.name}?</AlertDialogTitle>
            <AlertDialogDescription>
              This will remove all associated projects and sessions. This action
              cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-white hover:bg-destructive/90"
              onClick={() => deleteMachine.mutate(machine.id)}
            >
              {deleteMachine.isPending ? 'Deleting...' : 'Delete'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
