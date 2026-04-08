import { useQuery } from '@tanstack/react-query'
import { portApi } from './api'

export function portsQueryKey(machineId: string, projectDir?: string) {
  return ['ports', machineId, projectDir ?? ''] as const
}

export function usePorts(machineId: string, projectDir?: string) {
  return useQuery({
    queryKey: portsQueryKey(machineId, projectDir),
    queryFn: () => portApi.list(machineId, projectDir),
    staleTime: 15 * 1000,
    refetchInterval: 15 * 1000,
    enabled: !!machineId,
  })
}
