import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { sessionApi, terminalApi } from './api'
import type { CreateSessionInput, UpdateSessionInput } from './api'

export const SESSIONS_QUERY_KEY = ['sessions'] as const

export function sessionsQueryKey(projectId: string) {
  return ['sessions', 'project', projectId] as const
}

export function sessionQueryKey(id: string) {
  return ['sessions', id] as const
}

export function useSessions(projectId: string) {
  return useQuery({
    queryKey: sessionsQueryKey(projectId),
    queryFn: () => sessionApi.list(projectId),
    staleTime: 30 * 1000,
    enabled: !!projectId,
  })
}

export function useSession(id: string) {
  return useQuery({
    queryKey: sessionQueryKey(id),
    queryFn: () => sessionApi.get(id),
    staleTime: 15 * 1000,
    enabled: !!id,
  })
}

export function useCreateSession() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      projectId,
      data,
    }: {
      projectId: string
      data?: CreateSessionInput
    }) => sessionApi.create(projectId, data),
    onSuccess: (_result, variables) => {
      void queryClient.invalidateQueries({
        queryKey: sessionsQueryKey(variables.projectId),
      })
    },
  })
}

export function useUpdateSession() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateSessionInput }) =>
      sessionApi.update(id, data),
    onSuccess: (_result, variables) => {
      void queryClient.invalidateQueries({ queryKey: SESSIONS_QUERY_KEY })
      void queryClient.invalidateQueries({
        queryKey: sessionQueryKey(variables.id),
      })
    },
  })
}

export function useDeleteSession() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => sessionApi.delete(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: SESSIONS_QUERY_KEY })
    },
  })
}

export function useCreateTerminal() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (sessionId: string) => terminalApi.create(sessionId),
    onSuccess: (_result, sessionId) => {
      void queryClient.invalidateQueries({
        queryKey: sessionQueryKey(sessionId),
      })
    },
  })
}

export function useDeleteTerminal() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      terminalId,
    }: {
      terminalId: string
      sessionId: string
    }) => terminalApi.delete(terminalId),
    onSuccess: (_result, variables) => {
      void queryClient.invalidateQueries({
        queryKey: sessionQueryKey(variables.sessionId),
      })
    },
  })
}
