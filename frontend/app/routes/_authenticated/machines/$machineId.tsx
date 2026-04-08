import { useState, useEffect } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { Check, X, Plug, Unplug, Loader2, Wrench } from 'lucide-react'
import { Button } from '~/components/ui/button'
import { Badge } from '~/components/ui/badge'
import { StatusDot } from '~/components/status-dot'
import { useMachine, useConnectMachine, useDisconnectMachine } from '~/lib/machines'
import { useSetupStatus, hasMissingCoreCapabilities } from '~/lib/setup'
import { SetupWizard } from '~/components/setup-wizard'
import type { SetupCapabilities } from '~/lib/api'

export const Route = createFileRoute('/_authenticated/machines/$machineId')({
  component: MachineDetailPage,
})

function MachineDetailPage() {
  const { machineId } = Route.useParams()
  const { data: machine, isLoading, error } = useMachine(machineId)
  const connectMachine = useConnectMachine()
  const disconnectMachine = useDisconnectMachine()
  const [wizardOpen, setWizardOpen] = useState(false)
  const [autoOpenDone, setAutoOpenDone] = useState(false)

  const isConnected = machine?.status === 'connected'

  const { data: setupStatus } = useSetupStatus(machineId, isConnected)

  // Auto-open wizard when connecting to a machine with missing core capabilities
  useEffect(() => {
    if (
      isConnected &&
      setupStatus &&
      !autoOpenDone &&
      hasMissingCoreCapabilities(setupStatus.capabilities)
    ) {
      setWizardOpen(true)
      setAutoOpenDone(true)
    }
  }, [isConnected, setupStatus, autoOpenDone])

  // Reset auto-open flag when machine disconnects
  useEffect(() => {
    if (!isConnected) {
      setAutoOpenDone(false)
    }
  }, [isConnected])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="size-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (error || !machine) {
    return (
      <div className="space-y-2 py-12 text-center">
        <p className="text-lg font-semibold">Machine not found</p>
        <p className="text-sm text-muted-foreground">
          The machine you are looking for does not exist or has been deleted.
        </p>
      </div>
    )
  }

  const isActionPending = connectMachine.isPending || disconnectMachine.isPending

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="space-y-1">
          <div className="flex items-center gap-3">
            <h1 className="text-3xl font-bold tracking-tight">
              {machine.name}
            </h1>
            <StatusDot status={machine.status} className="size-3" />
          </div>
          <p className="text-muted-foreground">
            {machine.host}:{machine.port}
          </p>
        </div>

        <div className="flex gap-2">
          {isConnected && (
            <Button
              variant="outline"
              onClick={() => setWizardOpen(true)}
            >
              <Wrench className="mr-2 size-4" />
              Run Setup Wizard
            </Button>
          )}
          {isConnected ? (
            <Button
              variant="outline"
              onClick={() => disconnectMachine.mutate(machine.id)}
              disabled={isActionPending}
            >
              {disconnectMachine.isPending ? (
                <Loader2 className="mr-2 size-4 animate-spin" />
              ) : (
                <Unplug className="mr-2 size-4" />
              )}
              Disconnect
            </Button>
          ) : (
            <Button
              onClick={() => connectMachine.mutate(machine.id)}
              disabled={isActionPending}
            >
              {connectMachine.isPending ? (
                <Loader2 className="mr-2 size-4 animate-spin" />
              ) : (
                <Plug className="mr-2 size-4" />
              )}
              Connect
            </Button>
          )}
        </div>
      </div>

      {/* Info Card */}
      <div className="rounded-lg border bg-card p-6">
        <h2 className="mb-4 text-lg font-semibold">Details</h2>
        <dl className="grid gap-3 sm:grid-cols-2">
          <div>
            <dt className="text-sm font-medium text-muted-foreground">Host</dt>
            <dd className="text-sm">{machine.host}</dd>
          </div>
          <div>
            <dt className="text-sm font-medium text-muted-foreground">Port</dt>
            <dd className="text-sm">{machine.port}</dd>
          </div>
          <div>
            <dt className="text-sm font-medium text-muted-foreground">
              Status
            </dt>
            <dd className="flex items-center gap-2 text-sm capitalize">
              <StatusDot status={machine.status} />
              {machine.status}
            </dd>
          </div>
          <div>
            <dt className="text-sm font-medium text-muted-foreground">
              Auth Type
            </dt>
            <dd className="text-sm capitalize">{machine.auth_type === 'key' ? 'SSH Key' : 'Password'}</dd>
          </div>
          {machine.last_heartbeat && (
            <div>
              <dt className="text-sm font-medium text-muted-foreground">
                Last Heartbeat
              </dt>
              <dd className="text-sm">
                {new Date(machine.last_heartbeat).toLocaleString()}
              </dd>
            </div>
          )}
          <div>
            <dt className="text-sm font-medium text-muted-foreground">
              Created
            </dt>
            <dd className="text-sm">
              {new Date(machine.created_at).toLocaleString()}
            </dd>
          </div>
        </dl>
      </div>

      {/* Capabilities */}
      <div className="rounded-lg border bg-card p-6">
        <h2 className="mb-4 text-lg font-semibold">Capabilities</h2>
        {machine.capabilities &&
        Object.keys(machine.capabilities).length > 0 ? (
          <div className="flex flex-wrap gap-2">
            {Object.entries(machine.capabilities).map(([name, available]) => (
              <Badge
                key={name}
                variant={available ? 'default' : 'secondary'}
                className="gap-1"
              >
                {available ? (
                  <Check className="size-3" />
                ) : (
                  <X className="size-3" />
                )}
                {name}
              </Badge>
            ))}
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">
            {machine.status === 'connected'
              ? 'No capabilities detected.'
              : 'Connect to the machine to detect capabilities.'}
          </p>
        )}
      </div>

      {/* Setup Wizard Dialog */}
      <SetupWizard
        open={wizardOpen}
        onOpenChange={setWizardOpen}
        machineId={machineId}
        machineName={machine.name}
      />
    </div>
  )
}
