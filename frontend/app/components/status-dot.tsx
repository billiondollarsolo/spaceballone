import { cn } from '~/lib/utils'
import type { Machine } from '~/lib/api'

const statusColors: Record<Machine['status'], string> = {
  connected: 'bg-green-500',
  reconnecting: 'bg-yellow-500',
  disconnected: 'bg-red-500',
  error: 'bg-red-500',
}

const statusLabels: Record<Machine['status'], string> = {
  connected: 'Connected',
  reconnecting: 'Reconnecting',
  disconnected: 'Disconnected',
  error: 'Error',
}

interface StatusDotProps {
  status: Machine['status'] | undefined
  className?: string
}

export function StatusDot({ status, className }: StatusDotProps) {
  const color = status ? statusColors[status] : 'bg-gray-400'
  const label = status ? statusLabels[status] : 'Never connected'

  return (
    <span
      className={cn('inline-block size-2.5 shrink-0 rounded-full', color, className)}
      title={label}
      aria-label={label}
    />
  )
}
