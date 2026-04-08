import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { machineApi } from './api'
import type { CreateMachineInput, UpdateMachineInput } from './api'

export const MACHINES_QUERY_KEY = ['machines'] as const

export function machineQueryKey(id: string) {
  return ['machines', id] as const
}

export function useMachines() {
  return useQuery({
    queryKey: MACHINES_QUERY_KEY,
    queryFn: () => machineApi.list(),
    staleTime: 30 * 1000, // 30 seconds
  })
}

export function useMachine(id: string) {
  return useQuery({
    queryKey: machineQueryKey(id),
    queryFn: () => machineApi.get(id),
    staleTime: 30 * 1000,
  })
}

export function useCreateMachine() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateMachineInput) => machineApi.create(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: MACHINES_QUERY_KEY })
    },
  })
}

export function useUpdateMachine() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateMachineInput }) =>
      machineApi.update(id, data),
    onSuccess: (_result, variables) => {
      void queryClient.invalidateQueries({ queryKey: MACHINES_QUERY_KEY })
      void queryClient.invalidateQueries({
        queryKey: machineQueryKey(variables.id),
      })
    },
  })
}

export function useDeleteMachine() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => machineApi.delete(id),
    onSuccess: (_result, id) => {
      void queryClient.invalidateQueries({ queryKey: MACHINES_QUERY_KEY })
      queryClient.removeQueries({ queryKey: machineQueryKey(id) })
    },
  })
}

export function useConnectMachine() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => machineApi.connect(id),
    onSuccess: (_result, id) => {
      void queryClient.invalidateQueries({ queryKey: MACHINES_QUERY_KEY })
      void queryClient.invalidateQueries({ queryKey: machineQueryKey(id) })
    },
  })
}

export function useDisconnectMachine() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => machineApi.disconnect(id),
    onSuccess: (_result, id) => {
      void queryClient.invalidateQueries({ queryKey: MACHINES_QUERY_KEY })
      void queryClient.invalidateQueries({ queryKey: machineQueryKey(id) })
    },
  })
}
