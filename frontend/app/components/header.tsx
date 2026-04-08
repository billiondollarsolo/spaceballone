import { useState, useRef, useEffect } from 'react'
import {
  Menu,
  LogOut,
  KeyRound,
  Search,
  X,
  ChevronRight,
  Server,
  FolderOpen,
  MonitorPlay,
} from 'lucide-react'
import { useRouter, useMatches } from '@tanstack/react-router'
import { Button } from '~/components/ui/button'
import { Input } from '~/components/ui/input'
import { ThemeToggle } from '~/components/theme-toggle'
import { Avatar, AvatarFallback } from '~/components/ui/avatar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '~/components/ui/dropdown-menu'
import { useAuth, useLogout } from '~/lib/auth'
import { useSearch } from '~/lib/search'
import type { SearchResult } from '~/lib/search'
import { NotificationsBell } from '~/components/notifications'
import { QuickConnect } from '~/components/quick-connect'
import { ChangePasswordDialog } from '~/components/change-password-dialog'
import { useQueryClient } from '@tanstack/react-query'
import { MACHINES_QUERY_KEY } from '~/lib/machines'
import { sessionQueryKey } from '~/lib/sessions'
import type { Machine, Session } from '~/lib/api'

interface HeaderProps {
  onToggleSidebar: () => void
}

function Breadcrumbs() {
  const matches = useMatches()
  const queryClient = useQueryClient()

  function getMachineName(machineId: string): string {
    const machines = queryClient.getQueryData<Machine[]>(MACHINES_QUERY_KEY)
    const machine = machines?.find((m) => m.id === machineId)
    return machine?.name ?? machineId.slice(0, 8)
  }

  function getSessionName(sessionId: string): string {
    const session = queryClient.getQueryData<Session>(sessionQueryKey(sessionId))
    return session?.name ?? sessionId.slice(0, 8)
  }

  const crumbs: { label: string; path?: string }[] = []

  for (const match of matches) {
    const routeId = match.routeId
    if (routeId === '/_authenticated/dashboard') {
      crumbs.push({ label: 'Dashboard' })
    } else if (routeId === '/_authenticated/machines/$machineId') {
      const params = match.params as { machineId?: string }
      crumbs.push({ label: 'Machines' })
      crumbs.push({ label: params.machineId ? getMachineName(params.machineId) : 'Machine', path: match.pathname })
    } else if (routeId === '/_authenticated/sessions/$sessionId') {
      const params = match.params as { sessionId?: string }
      crumbs.push({ label: 'Sessions' })
      crumbs.push({ label: params.sessionId ? getSessionName(params.sessionId) : 'Session', path: match.pathname })
    }
  }

  if (crumbs.length === 0) {
    crumbs.push({ label: 'Dashboard' })
  }

  return (
    <nav className="hidden items-center gap-1 text-sm text-muted-foreground md:flex" aria-label="Breadcrumb">
      {crumbs.map((crumb, i) => (
        <span key={i} className="flex items-center gap-1">
          {i > 0 && <ChevronRight className="size-3" />}
          <span className="max-w-[120px] truncate">{crumb.label}</span>
        </span>
      ))}
    </nav>
  )
}

function SearchResultIcon({ type }: { type: SearchResult['type'] }) {
  switch (type) {
    case 'machine':
      return <Server className="size-4 shrink-0 text-muted-foreground" />
    case 'project':
      return <FolderOpen className="size-4 shrink-0 text-muted-foreground" />
    case 'session':
      return <MonitorPlay className="size-4 shrink-0 text-muted-foreground" />
  }
}

function GlobalSearch() {
  const [query, setQuery] = useState('')
  const [open, setOpen] = useState(false)
  const [mobileExpanded, setMobileExpanded] = useState(false)
  const { grouped, isLoading } = useSearch(query)
  const router = useRouter()
  const inputRef = useRef<HTMLInputElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  const hasResults =
    grouped.machines.length > 0 ||
    grouped.projects.length > 0 ||
    grouped.sessions.length > 0

  const showDropdown = open && query.trim().length >= 2

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false)
        setMobileExpanded(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  function navigateTo(result: SearchResult) {
    setOpen(false)
    setQuery('')
    setMobileExpanded(false)
    const meta = result.metadata as Record<string, string> | undefined
    switch (result.type) {
      case 'machine':
        void router.navigate({ to: '/machines/$machineId', params: { machineId: result.id } })
        break
      case 'project':
        if (meta?.machine_id) {
          void router.navigate({ to: '/machines/$machineId', params: { machineId: meta.machine_id } })
        }
        break
      case 'session':
        void router.navigate({ to: '/sessions/$sessionId', params: { sessionId: result.id } })
        break
    }
  }

  function renderGroup(label: string, items: SearchResult[]) {
    if (items.length === 0) return null
    return (
      <div>
        <div className="px-3 py-1.5 text-xs font-semibold text-muted-foreground uppercase tracking-wider">
          {label}
        </div>
        {items.map((item) => (
          <button
            key={`${item.type}-${item.id}`}
            className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm hover:bg-accent min-h-[44px]"
            onClick={() => navigateTo(item)}
          >
            <SearchResultIcon type={item.type} />
            <span className="truncate">{item.name}</span>
          </button>
        ))}
      </div>
    )
  }

  // Mobile: show icon button that expands to full-width overlay
  if (mobileExpanded) {
    return (
      <div
        ref={containerRef}
        className="absolute inset-x-0 top-0 z-50 flex h-14 items-center gap-2 border-b bg-background px-4 md:hidden"
      >
        <div className="relative flex-1">
          <Search className="absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            ref={inputRef}
            value={query}
            onChange={(e) => {
              setQuery(e.target.value)
              setOpen(true)
            }}
            onFocus={() => setOpen(true)}
            placeholder="Search machines, projects, sessions..."
            className="pl-9"
            autoFocus
          />
          {showDropdown && (
            <div className="absolute left-0 right-0 top-full mt-1 rounded-md border bg-popover shadow-lg">
              {isLoading ? (
                <div className="px-3 py-4 text-center text-sm text-muted-foreground">
                  Searching...
                </div>
              ) : !hasResults ? (
                <div className="px-3 py-4 text-center text-sm text-muted-foreground">
                  No results found
                </div>
              ) : (
                <div className="max-h-[300px] overflow-y-auto py-1">
                  {renderGroup('Machines', grouped.machines)}
                  {renderGroup('Projects', grouped.projects)}
                  {renderGroup('Sessions', grouped.sessions)}
                </div>
              )}
            </div>
          )}
        </div>
        <Button
          variant="ghost"
          size="icon"
          className="min-h-[44px] min-w-[44px]"
          onClick={() => {
            setMobileExpanded(false)
            setQuery('')
            setOpen(false)
          }}
        >
          <X className="size-5" />
        </Button>
      </div>
    )
  }

  return (
    <>
      {/* Mobile: just icon */}
      <Button
        variant="ghost"
        size="icon"
        className="min-h-[44px] min-w-[44px] md:hidden"
        onClick={() => setMobileExpanded(true)}
        aria-label="Search"
      >
        <Search className="size-5" />
      </Button>

      {/* Desktop: full input */}
      <div ref={containerRef} className="relative hidden md:block">
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            ref={inputRef}
            value={query}
            onChange={(e) => {
              setQuery(e.target.value)
              setOpen(true)
            }}
            onFocus={() => setOpen(true)}
            placeholder="Search..."
            className="w-[240px] pl-9"
          />
        </div>
        {showDropdown && (
          <div className="absolute left-0 right-0 top-full mt-1 w-[360px] rounded-md border bg-popover shadow-lg">
            {isLoading ? (
              <div className="px-3 py-4 text-center text-sm text-muted-foreground">
                Searching...
              </div>
            ) : !hasResults ? (
              <div className="px-3 py-4 text-center text-sm text-muted-foreground">
                No results found
              </div>
            ) : (
              <div className="max-h-[300px] overflow-y-auto py-1">
                {renderGroup('Machines', grouped.machines)}
                {renderGroup('Projects', grouped.projects)}
                {renderGroup('Sessions', grouped.sessions)}
              </div>
            )}
          </div>
        )}
      </div>
    </>
  )
}

export function Header({ onToggleSidebar }: HeaderProps) {
  const { data: user } = useAuth()
  const logout = useLogout()
  const [changePasswordOpen, setChangePasswordOpen] = useState(false)

  const initials = user?.email
    ? user.email.slice(0, 2).toUpperCase()
    : '??'

  return (
    <header className="sticky top-0 z-40 flex h-14 items-center gap-4 border-b bg-background px-4 lg:px-6">
      {/* Left: hamburger (mobile) + logo */}
      <div className="flex items-center gap-2">
        <Button
          variant="ghost"
          size="icon"
          className="min-h-[44px] min-w-[44px] md:hidden"
          onClick={onToggleSidebar}
          aria-label="Toggle sidebar"
        >
          <Menu className="size-5" />
        </Button>
        <div className="flex items-center gap-1.5">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
            className="size-5 text-primary"
          >
            <path d="M12 2L4 7v10l8 5 8-5V7l-8-5z" />
            <path d="M12 7v10" />
            <path d="M7 9.5l5 3 5-3" />
          </svg>
          <span className="text-lg font-bold tracking-tight">SpaceBallOne</span>
        </div>
      </div>

      {/* Center: breadcrumb */}
      <div className="flex flex-1 items-center gap-4">
        <Breadcrumbs />
      </div>

      {/* Right: search + notifications + quick-connect + theme + user */}
      <div className="flex items-center gap-1">
        <GlobalSearch />
        <NotificationsBell />
        <QuickConnect />
        <ThemeToggle />

        {user && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="relative size-9 rounded-full min-h-[44px] min-w-[44px]">
                <Avatar className="size-8">
                  <AvatarFallback className="text-xs">{initials}</AvatarFallback>
                </Avatar>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-56">
              <DropdownMenuLabel className="font-normal">
                <div className="flex flex-col space-y-1">
                  <p className="text-sm font-medium leading-none">{user.email}</p>
                  <p className="text-xs leading-none text-muted-foreground">
                    Administrator
                  </p>
                </div>
              </DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={() => setChangePasswordOpen(true)}>
                <KeyRound className="mr-2 size-4" />
                Change Password
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={() => logout.mutate()}
                className="text-destructive focus:text-destructive"
              >
                <LogOut className="mr-2 size-4" />
                Logout
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>
      <ChangePasswordDialog
        open={changePasswordOpen}
        onOpenChange={setChangePasswordOpen}
      />
    </header>
  )
}
