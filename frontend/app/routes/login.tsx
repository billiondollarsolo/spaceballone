import { createFileRoute, redirect } from '@tanstack/react-router'
import { LoginForm } from '~/components/login-form'
import { AUTH_QUERY_KEY } from '~/lib/auth'

export const Route = createFileRoute('/login')({
  beforeLoad: async ({ context }) => {
    // If already authenticated, redirect to dashboard
    try {
      const user = await context.queryClient.ensureQueryData({
        queryKey: AUTH_QUERY_KEY,
        queryFn: async () => {
          const { authApi } = await import('~/lib/api')
          return authApi.me()
        },
        staleTime: 5 * 60 * 1000,
      })
      if (user) {
        throw redirect({ to: '/dashboard' })
      }
    } catch (error) {
      if (error instanceof Error && 'to' in error) {
        // This is a redirect, rethrow it
        throw error
      }
      // Not authenticated, continue to login page
    }
  },
  component: LoginPage,
})

function LoginPage() {
  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <LoginForm />
    </div>
  )
}
