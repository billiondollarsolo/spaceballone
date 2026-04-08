import { useState, useEffect } from 'react'
import { createFileRoute, Outlet, redirect, useLocation } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import { Header } from '~/components/header'
import { SidebarContent } from '~/components/sidebar'
import { ChangePasswordDialog } from '~/components/change-password-dialog'
import { Sheet, SheetContent, SheetTitle } from '~/components/ui/sheet'
import { useAuth, AUTH_QUERY_KEY } from '~/lib/auth'
import { connectStatusWebSocket } from '~/lib/websocket'
import { NotificationsProvider, useNotifications } from '~/components/notifications'
import { MACHINES_QUERY_KEY } from '~/lib/machines'
import type { Machine } from '~/lib/api'

export const Route = createFileRoute('/_authenticated')({
  beforeLoad: async ({ context }) => {
    try {
      const user = await context.queryClient.ensureQueryData({
        queryKey: AUTH_QUERY_KEY,
        queryFn: async () => {
          const { authApi } = await import('~/lib/api')
          return authApi.me()
        },
        staleTime: 5 * 60 * 1000,
      })
      if (!user) {
        throw redirect({ to: '/login' })
      }
    } catch (error) {
      if (error instanceof Error && 'to' in error) {
        throw error
      }
      throw redirect({ to: '/login' })
    }
  },
  component: AuthenticatedLayoutWrapper,
})

function AuthenticatedLayoutWrapper() {
  return (
    <NotificationsProvider>
      <AuthenticatedLayout />
    </NotificationsProvider>
  )
}

function AuthenticatedLayout() {
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const [prevPathname, setPrevPathname] = useState('')
  const { data: user } = useAuth()
  const queryClient = useQueryClient()
  const location = useLocation()
  const { addNotification } = useNotifications()

  // Auto-close mobile sidebar on route change (setState-during-render pattern)
  if (prevPathname !== location.pathname) {
    setPrevPathname(location.pathname)
    if (sidebarOpen) {
      setSidebarOpen(false)
    }
  }

  // Connect to status WebSocket for real-time machine status updates
  useEffect(() => {
    const cleanup = connectStatusWebSocket(queryClient, (machineId, status) => {
      const machines = queryClient.getQueryData<Machine[]>(MACHINES_QUERY_KEY)
      const machine = machines?.find((m) => m.id === machineId)
      const label = machine?.name ?? machineId.slice(0, 8)
      switch (status) {
        case 'connected':
          addNotification(`Machine ${label} connected`)
          break
        case 'disconnected':
          addNotification(`Machine ${label} disconnected`)
          break
        case 'reconnecting':
          addNotification(`Machine ${label} reconnecting`)
          break
      }
    })
    return cleanup
  }, [queryClient, addNotification])

  return (
    <div className="flex min-h-screen flex-col">
      <Header onToggleSidebar={() => setSidebarOpen((prev) => !prev)} />

      <div className="flex flex-1">
        {/* Desktop sidebar */}
        <aside className="hidden w-[280px] shrink-0 border-r bg-sidebar md:block">
          <SidebarContent />
        </aside>

        {/* Mobile sidebar (Sheet drawer) */}
        <Sheet open={sidebarOpen} onOpenChange={setSidebarOpen}>
          <SheetContent side="left" className="w-[280px] p-0">
            <SheetTitle className="sr-only">Navigation</SheetTitle>
            <SidebarContent />
          </SheetContent>
        </Sheet>

        {/* Main content */}
        <main className="flex-1 overflow-auto p-4 lg:p-6">
          <Outlet />
        </main>
      </div>

      {/* Force password change dialog */}
      {user?.must_change_password && (
        <ChangePasswordDialog open forced />
      )}
    </div>
  )
}
