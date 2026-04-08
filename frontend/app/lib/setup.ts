import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { setupApi } from './api'
import type { SetupCapabilities } from './api'

export const SETUP_STATUS_KEY = (machineId: string) =>
  ['setup', 'status', machineId] as const

export function useSetupStatus(machineId: string, enabled = true) {
  return useQuery({
    queryKey: SETUP_STATUS_KEY(machineId),
    queryFn: () => setupApi.status(machineId),
    enabled,
    staleTime: 60 * 1000,
  })
}

export function useSetupDiscover(machineId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => setupApi.discover(machineId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: SETUP_STATUS_KEY(machineId) })
    },
  })
}

export function hasMissingCoreCapabilities(caps: SetupCapabilities | null | undefined): boolean {
  if (!caps) return true
  return !caps.tmux || !caps.docker
}
