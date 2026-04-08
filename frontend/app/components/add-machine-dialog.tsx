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
import { useCreateMachine } from '~/lib/machines'

interface AddMachineDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function AddMachineDialog({ open, onOpenChange }: AddMachineDialogProps) {
  const [name, setName] = useState('')
  const [host, setHost] = useState('')
  const [port, setPort] = useState('22')
  const [authType, setAuthType] = useState<'password' | 'key'>('password')
  const [credentials, setCredentials] = useState('')

  const createMachine = useCreateMachine()

  function resetForm() {
    setName('')
    setHost('')
    setPort('22')
    setAuthType('password')
    setCredentials('')
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    createMachine.mutate(
      {
        name,
        host,
        port: parseInt(port, 10) || 22,
        auth_type: authType,
        credentials,
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
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add Machine</DialogTitle>
          <DialogDescription>
            Add a new remote machine to manage.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="add-name">Name</Label>
            <Input
              id="add-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="My Server"
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="add-host">Host</Label>
            <Input
              id="add-host"
              value={host}
              onChange={(e) => setHost(e.target.value)}
              placeholder="192.168.1.100"
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="add-port">Port</Label>
            <Input
              id="add-port"
              type="number"
              value={port}
              onChange={(e) => setPort(e.target.value)}
              min={1}
              max={65535}
              required
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
                <RadioGroupItem value="password" id="add-auth-password" />
                <Label htmlFor="add-auth-password" className="font-normal">
                  Password
                </Label>
              </div>
              <div className="flex items-center space-x-2">
                <RadioGroupItem value="key" id="add-auth-key" />
                <Label htmlFor="add-auth-key" className="font-normal">
                  SSH Key
                </Label>
              </div>
            </RadioGroup>
          </div>
          <div className="space-y-2">
            <Label htmlFor="add-credentials">
              {authType === 'password' ? 'Password' : 'Private Key'}
            </Label>
            {authType === 'password' ? (
              <Input
                id="add-credentials"
                type="password"
                value={credentials}
                onChange={(e) => setCredentials(e.target.value)}
                placeholder="Enter password"
                required
              />
            ) : (
              <Textarea
                id="add-credentials"
                value={credentials}
                onChange={(e) => setCredentials(e.target.value)}
                placeholder="Paste private key content"
                rows={4}
                className="font-mono text-xs"
                required
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
            <Button type="submit" disabled={createMachine.isPending}>
              {createMachine.isPending ? 'Adding...' : 'Add Machine'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
