import { createFileRoute } from '@tanstack/react-router'
import { Loader2 } from 'lucide-react'
import { SessionWorkspace } from '~/components/session-workspace'
import { useSession } from '~/lib/sessions'

export const Route = createFileRoute('/_authenticated/sessions/$sessionId')({
  component: SessionPage,
})

function SessionPage() {
  const { sessionId } = Route.useParams()
  const { data: session, isLoading, error } = useSession(sessionId)

  if (isLoading) {
    return (
      <div className="flex h-[calc(100vh-8rem)] items-center justify-center">
        <Loader2 className="size-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (error || !session) {
    return (
      <div className="space-y-2 py-12 text-center">
        <p className="text-lg font-semibold">Session not found</p>
        <p className="text-sm text-muted-foreground">
          The session you are looking for does not exist or has been terminated.
        </p>
      </div>
    )
  }

  return (
    <div className="h-[calc(100vh-8rem)]">
      <div className="mb-3 flex items-center justify-between">
        <h1 className="text-xl font-bold tracking-tight">{session.name}</h1>
        <span className="text-xs text-muted-foreground capitalize">
          {session.status}
        </span>
      </div>
      <div className="h-[calc(100%-3rem)]">
        <SessionWorkspace session={session} />
      </div>
    </div>
  )
}
