import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useRouter } from '@tanstack/react-router'
import { authApi, ApiError } from './api'
import type { LoginRequest, ChangePasswordRequest, User } from './api'

export const AUTH_QUERY_KEY = ['auth', 'me'] as const

export function useAuth() {
  return useQuery({
    queryKey: AUTH_QUERY_KEY,
    queryFn: () => authApi.me(),
    retry: false,
    staleTime: 5 * 60 * 1000, // 5 minutes
  })
}

export function useLogin() {
  const queryClient = useQueryClient()
  const router = useRouter()

  return useMutation({
    mutationFn: (data: LoginRequest) => authApi.login(data),
    onSuccess: (user: User) => {
      queryClient.setQueryData(AUTH_QUERY_KEY, user)
      // The _authenticated layout will handle showing the change password dialog if needed
      void router.navigate({ to: '/dashboard' })
    },
  })
}

export function useLogout() {
  const queryClient = useQueryClient()
  const router = useRouter()

  return useMutation({
    mutationFn: () => authApi.logout(),
    onSuccess: () => {
      queryClient.setQueryData(AUTH_QUERY_KEY, null)
      queryClient.clear()
      void router.navigate({ to: '/login' })
    },
  })
}

export function useChangePassword() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: ChangePasswordRequest) => authApi.changePassword(data),
    onSuccess: () => {
      // Refresh the auth state to clear must_change_password
      void queryClient.invalidateQueries({ queryKey: AUTH_QUERY_KEY })
    },
  })
}

export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError
}
