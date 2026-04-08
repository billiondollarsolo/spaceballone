import { useState, useCallback, useMemo, useEffect, useRef, createContext, useContext } from 'react'
import { Bell } from 'lucide-react'
import { Button } from '~/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '~/components/ui/dropdown-menu'
import { ScrollArea } from '~/components/ui/scroll-area'

export interface Notification {
  id: string
  message: string
  timestamp: Date
  read: boolean
}

interface NotificationsContextType {
  notifications: Notification[]
  addNotification: (message: string) => void
  markAllRead: () => void
  unreadCount: number
}

const NotificationsContext = createContext<NotificationsContextType>({
  notifications: [],
  addNotification: () => {},
  markAllRead: () => {},
  unreadCount: 0,
})

const MAX_NOTIFICATIONS = 50

export function useNotifications() {
  return useContext(NotificationsContext)
}

export function NotificationsProvider({ children }: { children: React.ReactNode }) {
  const [notifications, setNotifications] = useState<Notification[]>([])
  const counterRef = useRef(0)

  const addNotification = useCallback((message: string) => {
    counterRef.current += 1
    const notification: Notification = {
      id: `notif-${counterRef.current}-${Date.now()}`,
      message,
      timestamp: new Date(),
      read: false,
    }

    setNotifications((prev) => {
      const next = [notification, ...prev]
      return next.slice(0, MAX_NOTIFICATIONS)
    })
  }, [])

  const markAllRead = useCallback(() => {
    setNotifications((prev) => prev.map((n) => ({ ...n, read: true })))
  }, [])

  const unreadCount = useMemo(() => notifications.filter((n) => !n.read).length, [notifications])

  const value = useMemo(() => ({
    notifications, unreadCount, addNotification, markAllRead
  }), [notifications, unreadCount, addNotification, markAllRead])

  return (
    <NotificationsContext.Provider value={value}>
      {children}
    </NotificationsContext.Provider>
  )
}

export function NotificationsBell() {
  const { notifications, markAllRead, unreadCount } = useNotifications()

  return (
    <DropdownMenu onOpenChange={(open) => { if (open) markAllRead() }}>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className="relative min-h-[44px] min-w-[44px]"
          aria-label="Notifications"
        >
          <Bell className="size-5" />
          {unreadCount > 0 && (
            <span className="absolute -right-0.5 -top-0.5 flex size-5 items-center justify-center rounded-full bg-destructive text-[10px] font-bold text-white">
              {unreadCount > 99 ? '99+' : unreadCount}
            </span>
          )}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-80">
        <DropdownMenuLabel>Notifications</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {notifications.length === 0 ? (
          <div className="px-3 py-6 text-center text-sm text-muted-foreground">
            No notifications
          </div>
        ) : (
          <ScrollArea className="max-h-[300px]">
            <div className="space-y-1 p-1">
              {notifications.map((n) => (
                <div
                  key={n.id}
                  className={`rounded-md px-3 py-2 text-sm ${
                    n.read ? 'text-muted-foreground' : 'bg-accent font-medium'
                  }`}
                >
                  <p className="leading-snug">{n.message}</p>
                  <p className="mt-0.5 text-xs text-muted-foreground">
                    {n.timestamp.toLocaleTimeString()}
                  </p>
                </div>
              ))}
            </div>
          </ScrollArea>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
