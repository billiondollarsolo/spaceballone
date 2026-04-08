import { useState } from 'react'
import { Plus } from 'lucide-react'
import { Button } from '~/components/ui/button'
import { ScrollArea } from '~/components/ui/scroll-area'
import { Separator } from '~/components/ui/separator'
import { MachineList } from '~/components/machine-list'
import { AddMachineDialog } from '~/components/add-machine-dialog'

export function SidebarContent() {
  const [addOpen, setAddOpen] = useState(false)

  return (
    <div className="flex h-full flex-col">
      {/* Machines section */}
      <div className="px-4 py-3">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold tracking-tight">Machines</h2>
        </div>
      </div>
      <Separator />

      {/* Machine list */}
      <ScrollArea className="flex-1 px-2 py-2">
        <MachineList />
      </ScrollArea>

      {/* Add Machine button */}
      <Separator />
      <div className="p-4">
        <Button
          variant="outline"
          className="w-full justify-start gap-2"
          onClick={() => setAddOpen(true)}
        >
          <Plus className="size-4" />
          Add Machine
        </Button>
      </div>

      <AddMachineDialog open={addOpen} onOpenChange={setAddOpen} />
    </div>
  )
}
