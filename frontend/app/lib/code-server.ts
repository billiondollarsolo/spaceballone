import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { codeServerApi } from './api'

export function codeServerStatusKey(machineId: string) {
  return ['code-server', 'status', machineId] as const
}

export function useCodeServerStatus(machineId: string) {
  return useQuery({
    queryKey: codeServerStatusKey(machineId),
    queryFn: () => codeServerApi.status(machineId),
    refetchInterval: 10_000,
    refetchIntervalInBackground: false,
    enabled: !!machineId,
  })
}

export function useStartCodeServer(machineId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => codeServerApi.start(machineId),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: codeServerStatusKey(machineId),
      })
    },
  })
}

export function useStopCodeServer(machineId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => codeServerApi.stop(machineId),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: codeServerStatusKey(machineId),
      })
    },
  })
}
