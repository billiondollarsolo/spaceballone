import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { browserlessApi } from './api'

export function browserlessStatusKey(machineId: string) {
  return ['browserless', 'status', machineId] as const
}

export function useBrowserlessStatus(machineId: string) {
  return useQuery({
    queryKey: browserlessStatusKey(machineId),
    queryFn: () => browserlessApi.status(machineId),
    refetchInterval: 10_000,
    refetchIntervalInBackground: false,
    enabled: !!machineId,
  })
}

export function useStartBrowserless(machineId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => browserlessApi.start(machineId),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: browserlessStatusKey(machineId),
      })
    },
  })
}

export function useStopBrowserless(machineId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => browserlessApi.stop(machineId),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: browserlessStatusKey(machineId),
      })
    },
  })
}
