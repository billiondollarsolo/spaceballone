import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { projectApi, browseDirectory } from './api'
import type { CreateProjectInput, UpdateProjectInput } from './api'

export const PROJECTS_QUERY_KEY = ['projects'] as const

export function projectsQueryKey(machineId: string) {
  return ['projects', 'machine', machineId] as const
}

export function projectQueryKey(id: string) {
  return ['projects', id] as const
}

export function browseQueryKey(machineId: string, path: string) {
  return ['browse', machineId, path] as const
}

export function useProjects(machineId: string) {
  return useQuery({
    queryKey: projectsQueryKey(machineId),
    queryFn: () => projectApi.list(machineId),
    staleTime: 30 * 1000,
    enabled: !!machineId,
  })
}

export function useProject(id: string) {
  return useQuery({
    queryKey: projectQueryKey(id),
    queryFn: () => projectApi.get(id),
    staleTime: 30 * 1000,
    enabled: !!id,
  })
}

export function useCreateProject() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateProjectInput) => projectApi.create(data),
    onSuccess: (_result, variables) => {
      void queryClient.invalidateQueries({
        queryKey: projectsQueryKey(variables.machine_id),
      })
    },
  })
}

export function useUpdateProject() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateProjectInput }) =>
      projectApi.update(id, data),
    onSuccess: (_result, variables) => {
      void queryClient.invalidateQueries({ queryKey: PROJECTS_QUERY_KEY })
      void queryClient.invalidateQueries({
        queryKey: projectQueryKey(variables.id),
      })
    },
  })
}

export function useDeleteProject() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => projectApi.delete(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: PROJECTS_QUERY_KEY })
    },
  })
}

export function useBrowseDirectory(machineId: string, path: string) {
  return useQuery({
    queryKey: browseQueryKey(machineId, path),
    queryFn: () => browseDirectory(machineId, path),
    staleTime: 10 * 1000,
    enabled: !!machineId && !!path,
  })
}
