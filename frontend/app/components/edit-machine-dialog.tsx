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
import { Textarea } from '~/components/ui/textarea'
import { RadioGroup, RadioGroupItem } from '~/components/ui/radio-group'
import { useUpdateMachine } from '~/lib/machines'
import type { Machine } from '~/lib/api'

interface EditMachineDialogProps {
  machine: Machine | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

function EditMachineDialogContent({
  machine,
  onOpenChange,
}: {
  machine: Machine
  onOpenChange: (open: boolean) => void
}) {
  const [name, setName] = useState(machine.name)
  const [host, setHost] = useState(machine.host)
  const [port, setPort] = useState(String(machine.port))
  const [username, setUsername] = useState('root')
  const [authType, setAuthType] = useState<'password' | 'key'>(machine.auth_type)
  const [credentials, setCredentials] = useState('')

  const updateMachine = useUpdateMachine()

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!machine) return

    updateMachine.mutate(
      {
        id: machine.id,
        data: {
          name,
          host,
          port: parseInt(port, 10) || 22,
          auth_type: authType,
          ...(credentials ? { credentials: `${username}\n${credentials}` } : {}),
        },
      },
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
        <DialogTitle>Edit Machine</DialogTitle>
        <DialogDescription>
          Update machine configuration. Leave credentials empty to keep unchanged.
        </DialogDescription>
      </DialogHeader>
      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="edit-name">Name</Label>
          <Input
            id="edit-name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="edit-host">Host</Label>
          <Input
            id="edit-host"
            value={host}
            onChange={(e) => setHost(e.target.value)}
            required
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="edit-port">Port</Label>
          <Input
            id="edit-port"
            type="number"
            value={port}
            onChange={(e) => setPort(e.target.value)}
            min={1}
            max={65535}
            required
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="edit-username">Username</Label>
          <Input
            id="edit-username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="root"
            required
            autoComplete="off"
          />
        </div>
        <div className="space-y-2">
          <Label>Auth Type</Label>
          <RadioGroup
            value={authType}
            onValueChange={(v) => setAuthType(v as 'password' | 'key')}
            className="flex gap-4"
          >
            <div className="flex items-center space-x-2">
              <RadioGroupItem value="password" id="edit-auth-password" />
              <Label htmlFor="edit-auth-password" className="font-normal">
                Password
              </Label>
            </div>
            <div className="flex items-center space-x-2">
              <RadioGroupItem value="key" id="edit-auth-key" />
              <Label htmlFor="edit-auth-key" className="font-normal">
                SSH Key
              </Label>
            </div>
          </RadioGroup>
        </div>
        <div className="space-y-2">
          <Label htmlFor="edit-credentials">
            {authType === 'password' ? 'Password' : 'Private Key'}
          </Label>
          {authType === 'password' ? (
            <Input
              id="edit-credentials"
              type="password"
              value={credentials}
              onChange={(e) => setCredentials(e.target.value)}
              placeholder="(unchanged)"
            />
          ) : (
            <Textarea
              id="edit-credentials"
              value={credentials}
              onChange={(e) => setCredentials(e.target.value)}
              placeholder="(unchanged)"
              rows={4}
              className="font-mono text-xs"
            />
          )}
        </div>
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
          >
            Cancel
          </Button>
          <Button type="submit" disabled={updateMachine.isPending}>
            {updateMachine.isPending ? 'Saving...' : 'Save Changes'}
          </Button>
        </DialogFooter>
      </form>
    </>
  )
}

export function EditMachineDialog({
  machine,
  open,
  onOpenChange,
}: EditMachineDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        {machine && (
          <EditMachineDialogContent
            key={machine.id}
            machine={machine}
            onOpenChange={onOpenChange}
          />
        )}
      </DialogContent>
    </Dialog>
  )
}
