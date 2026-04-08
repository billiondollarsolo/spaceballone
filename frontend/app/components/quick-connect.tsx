import { Zap, Loader2 } from 'lucide-react'
import { Button } from '~/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '~/components/ui/dropdown-menu'
import { ScrollArea } from '~/components/ui/scroll-area'
import { StatusDot } from '~/components/status-dot'
import { useMachines, useConnectMachine } from '~/lib/machines'

export function QuickConnect() {
  const { data: machines } = useMachines()
  const connectMachine = useConnectMachine()

  const disconnected = machines?.filter((m) => m.status !== 'connected') ?? []

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className="relative min-h-[44px] min-w-[44px]"
          aria-label="Quick connect"
        >
          <Zap className="size-5" />
          {disconnected.length > 0 && (
            <span className="absolute -right-0.5 -top-0.5 flex size-4 items-center justify-center rounded-full bg-muted text-[10px] font-bold text-muted-foreground">
              {disconnected.length}
            </span>
          )}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-72">
        <DropdownMenuLabel>Quick Connect</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {disconnected.length === 0 ? (
          <div className="px-3 py-6 text-center text-sm text-muted-foreground">
            All machines connected
          </div>
        ) : (
          <ScrollArea className="max-h-[300px]">
            <div className="space-y-1 p-1">
              {disconnected.map((machine) => (
                <div
                  key={machine.id}
                  className="flex items-center justify-between rounded-md px-3 py-2 hover:bg-accent"
                >
                  <div className="flex items-center gap-2 overflow-hidden">
                    <StatusDot status={machine.status} />
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium">{machine.name}</p>
                      <p className="truncate text-xs text-muted-foreground">
                        {machine.host}
                      </p>
                    </div>
                  </div>
                  <Button
                    size="sm"
                    variant="outline"
                    className="ml-2 h-8 shrink-0"
                    disabled={connectMachine.isPending}
                    onClick={() => connectMachine.mutate(machine.id)}
                  >
                    {connectMachine.isPending ? (
                      <Loader2 className="size-3 animate-spin" />
                    ) : (
                      'Connect'
                    )}
                  </Button>
                </div>
              ))}
            </div>
          </ScrollArea>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
