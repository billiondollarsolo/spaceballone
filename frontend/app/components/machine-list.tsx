import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { Monitor, MoreHorizontal, ChevronRight, ChevronDown, Power, PowerOff } from 'lucide-react'
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
import { StatusDot } from '~/components/status-dot'
import { MachineContextMenu } from '~/components/machine-context-menu'
import { EditMachineDialog } from '~/components/edit-machine-dialog'
import { ProjectList } from '~/components/project-list'
import { useMachines, useDeleteMachine, useConnectMachine, useDisconnectMachine } from '~/lib/machines'
import type { Machine } from '~/lib/api'

export function MachineList() {
  const { data: machines, isLoading } = useMachines()
  const [editingMachine, setEditingMachine] = useState<Machine | null>(null)

  if (isLoading) {
    return (
      <div className="flex flex-col items-center justify-center py-8 text-center">
        <p className="text-sm text-muted-foreground">Loading machines...</p>
      </div>
    )
  }

  if (!machines || machines.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-8 text-center">
        <Monitor className="mb-2 size-8 text-muted-foreground" />
        <p className="text-sm text-muted-foreground">No machines yet</p>
        <p className="text-xs text-muted-foreground">
          Add a machine to get started
        </p>
      </div>
    )
  }

  return (
    <>
      <div className="flex flex-col gap-0.5">
        {machines.map((machine) => (
          <MachineContextMenu
            key={machine.id}
            machine={machine}
            onEdit={() => setEditingMachine(machine)}
          >
            <MachineListItem
              machine={machine}
              onEdit={() => setEditingMachine(machine)}
            />
          </MachineContextMenu>
        ))}
      </div>

      <EditMachineDialog
        machine={editingMachine}
        open={editingMachine !== null}
        onOpenChange={(open) => {
          if (!open) setEditingMachine(null)
        }}
      />
    </>
  )
}

function MachineListItem({
  machine,
  onEdit,
}: {
  machine: Machine
  onEdit: () => void
}) {
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [connectOpen, setConnectOpen] = useState(false)
  const [disconnectOpen, setDisconnectOpen] = useState(false)
  const deleteMachine = useDeleteMachine()
  const connectMachine = useConnectMachine()
  const disconnectMachine = useDisconnectMachine()

  const isConnected = machine.status === 'connected'
  const isDisconnected = machine.status === 'disconnected'

  return (
    <>
      <div className="group flex items-center gap-1 rounded-md px-2 py-1.5 hover:bg-sidebar-accent">
        <button
          type="button"
          className="size-4 shrink-0 flex items-center justify-center"
          onClick={() => {
            if (isConnected) setDisconnectOpen(true)
            else if (isDisconnected) setConnectOpen(true)
          }}
          aria-label={isConnected ? 'Disconnect' : 'Connect'}
        >
          <StatusDot status={machine.status} />
        </button>

        <Link
          to="/machines/$machineId"
          params={{ machineId: machine.id }}
          className="flex min-w-0 flex-1 items-center gap-2"
        >
          <span className="truncate text-sm">{machine.name}</span>
        </Link>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="size-6 shrink-0 opacity-0 group-hover:opacity-100"
              aria-label={`Actions for ${machine.name}`}
            >
              <MoreHorizontal className="size-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-40">
            <DropdownMenuItem onClick={onEdit}>Edit</DropdownMenuItem>
            <DropdownMenuSeparator />
            {isConnected ? (
              <DropdownMenuItem
                onClick={() => setDisconnectOpen(true)}
              >
                <PowerOff className="size-3.5 mr-2" />
                Disconnect
              </DropdownMenuItem>
            ) : (
              <DropdownMenuItem
                onClick={() => setConnectOpen(true)}
              >
                <Power className="size-3.5 mr-2" />
                Connect
              </DropdownMenuItem>
            )}
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

      {isConnected && (
        <ProjectList machineId={machine.id} />
      )}

      {!isConnected && (
        <div className="pl-7 py-1">
          <button
            type="button"
            className="flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground"
            onClick={() => setConnectOpen(true)}
          >
            <Power className="size-3" />
            Connect
          </button>
        </div>
      )}

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

      <AlertDialog open={connectOpen} onOpenChange={setConnectOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Connect to {machine.name}?</AlertDialogTitle>
            <AlertDialogDescription>
              This will establish an SSH connection to {machine.host}.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => connectMachine.mutate(machine.id)}
            >
              {connectMachine.isPending ? 'Connecting...' : 'Connect'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={disconnectOpen} onOpenChange={setDisconnectOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Disconnect from {machine.name}?</AlertDialogTitle>
            <AlertDialogDescription>
              Active terminal sessions will be preserved by tmux and can be resumed when you reconnect.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => disconnectMachine.mutate(machine.id)}
            >
              {disconnectMachine.isPending ? 'Disconnecting...' : 'Disconnect'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
