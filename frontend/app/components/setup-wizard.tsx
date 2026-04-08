import { useState, useRef, useEffect, useCallback } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '~/components/ui/dialog'
import { Button } from '~/components/ui/button'
import { Badge } from '~/components/ui/badge'
import { ScrollArea } from '~/components/ui/scroll-area'
import {
  Check,
  X,
  AlertTriangle,
  Loader2,
  Download,
  ChevronRight,
  ChevronLeft,
} from 'lucide-react'
import { useSetupDiscover } from '~/lib/setup'
import { setupApi } from '~/lib/api'
import type { SetupCapabilities, SSEMessage } from '~/lib/api'

interface SetupWizardProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  machineId: string
  machineName: string
}

type WizardStep = 'discovery' | 'core' | 'optional' | 'agents' | 'complete'

const STEPS: WizardStep[] = ['discovery', 'core', 'optional', 'agents', 'complete']

const STEP_TITLES: Record<WizardStep, string> = {
  discovery: 'Scanning Capabilities',
  core: 'Core Requirements',
  optional: 'Optional Tools',
  agents: 'AI Agents',
  complete: 'Setup Complete',
}

interface PackageItem {
  key: string
  label: string
  description: string
  installed: boolean
  version?: string
  required?: boolean
}

export function SetupWizard({
  open,
  onOpenChange,
  machineId,
  machineName,
}: SetupWizardProps) {
  const [step, setStep] = useState<WizardStep>('discovery')
  const [capabilities, setCapabilities] = useState<SetupCapabilities | null>(null)
  const [installingPkg, setInstallingPkg] = useState<string | null>(null)
  const [installLog, setInstallLog] = useState<string[]>([])
  const [installError, setInstallError] = useState(false)
  const [installedPackages, setInstalledPackages] = useState<Set<string>>(new Set())
  const logEndRef = useRef<HTMLDivElement>(null)
  const activeEventSourceRef = useRef<EventSource | null>(null)

  const discover = useSetupDiscover(machineId)

  // Auto-discover on open
  useEffect(() => {
    if (open && step === 'discovery') {
      discover.mutate(undefined, {
        onSuccess: (caps) => {
          setCapabilities(caps)
        },
      })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open])

  // Reset state when dialog closes
  useEffect(() => {
    if (!open) {
      activeEventSourceRef.current?.close()
      activeEventSourceRef.current = null
      setStep('discovery')
      setCapabilities(null)
      setInstallingPkg(null)
      setInstallLog([])
      setInstallError(false)
      setInstalledPackages(new Set())
    }
  }, [open])

  // Abort active SSE stream on unmount
  useEffect(() => {
    return () => {
      activeEventSourceRef.current?.close()
      activeEventSourceRef.current = null
    }
  }, [])

  // Auto-scroll log
  useEffect(() => {
    logEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [installLog])

  const currentStepIndex = STEPS.indexOf(step)

  const goNext = useCallback(() => {
    const nextIndex = currentStepIndex + 1
    if (nextIndex < STEPS.length) {
      setStep(STEPS[nextIndex])
      setInstallLog([])
      setInstallingPkg(null)
      setInstallError(false)
    }
  }, [currentStepIndex])

  const goBack = useCallback(() => {
    const prevIndex = currentStepIndex - 1
    if (prevIndex >= 0) {
      setStep(STEPS[prevIndex])
      setInstallLog([])
      setInstallingPkg(null)
      setInstallError(false)
    }
  }, [currentStepIndex])

  function handleInstall(packageName: string) {
    setInstallingPkg(packageName)
    setInstallLog([])
    setInstallError(false)

    const eventSource = setupApi.install(machineId, packageName)
    activeEventSourceRef.current = eventSource

    const onMessage = (e: Event) => {
      const msg = (e as CustomEvent).detail as SSEMessage
      setInstallLog((prev) => [...prev, msg.line])
      if (msg.done) {
        if (msg.error) {
          setInstallError(true)
        } else {
          setInstalledPackages((prev) => new Set([...prev, packageName]))
          // Update capabilities locally
          if (capabilities) {
            const updated = { ...capabilities }
            switch (packageName) {
              case 'tmux': updated.tmux = true; break
              case 'docker': updated.docker = true; break
              case 'node': updated.node = true; break
              case 'go': updated.go_lang = true; break
              case 'claude_code': updated.claude_code = true; break
              case 'opencode': updated.opencode = true; break
              case 'codex': updated.codex = true; break
            }
            setCapabilities(updated)
          }
        }
        setInstallingPkg(null)
        eventSource.close()
        activeEventSourceRef.current = null
      }
    }

    const onError = () => {
      setInstallError(true)
      setInstallingPkg(null)
      setInstallLog((prev) => [...prev, 'Connection error'])
    }

    eventSource.addEventListener('message', onMessage)
    eventSource.addEventListener('error', onError)
  }

  function isInstalled(key: string): boolean {
    if (installedPackages.has(key)) return true
    if (!capabilities) return false
    switch (key) {
      case 'tmux': return capabilities.tmux
      case 'docker': return capabilities.docker
      case 'node': return capabilities.node
      case 'go': return capabilities.go_lang
      case 'claude_code': return capabilities.claude_code
      case 'opencode': return capabilities.opencode
      case 'codex': return capabilities.codex
      default: return false
    }
  }

  function getVersion(key: string): string | undefined {
    if (!capabilities) return undefined
    switch (key) {
      case 'tmux': return capabilities.tmux_version
      case 'docker': return capabilities.docker_version
      case 'node': return capabilities.node_version
      case 'go': return capabilities.go_version
      default: return undefined
    }
  }

  const corePackages: PackageItem[] = [
    { key: 'tmux', label: 'tmux', description: 'Terminal multiplexer for persistent sessions', installed: isInstalled('tmux'), version: getVersion('tmux'), required: true },
    { key: 'docker', label: 'Docker', description: 'Container runtime for isolated environments', installed: isInstalled('docker'), version: getVersion('docker'), required: true },
  ]

  const optionalPackages: PackageItem[] = [
    { key: 'node', label: 'Node.js', description: 'JavaScript runtime (recommended for JS/TS development)', installed: isInstalled('node'), version: getVersion('node') },
    { key: 'go', label: 'Go', description: 'Go programming language (recommended for Go development)', installed: isInstalled('go'), version: getVersion('go') },
  ]

  const agentPackages: PackageItem[] = [
    { key: 'claude_code', label: 'Claude Code', description: 'AI coding agent by Anthropic', installed: isInstalled('claude_code') },
    { key: 'opencode', label: 'OpenCode', description: 'Terminal-based AI coding tool', installed: isInstalled('opencode') },
    { key: 'codex', label: 'Codex CLI', description: 'AI coding agent by OpenAI', installed: isInstalled('codex') },
  ]

  function renderPackageRow(pkg: PackageItem) {
    const installed = pkg.installed
    return (
      <div key={pkg.key} className="flex items-center justify-between rounded-lg border p-3">
        <div className="flex items-center gap-3">
          {installed ? (
            <Check className="size-5 text-green-500" />
          ) : pkg.required ? (
            <X className="size-5 text-red-500" />
          ) : (
            <AlertTriangle className="size-5 text-yellow-500" />
          )}
          <div>
            <div className="flex items-center gap-2">
              <span className="font-medium">{pkg.label}</span>
              {installed && pkg.version && (
                <Badge variant="secondary" className="text-xs">{pkg.version}</Badge>
              )}
              {installed && (
                <Badge variant="default" className="text-xs">Installed</Badge>
              )}
            </div>
            <p className="text-xs text-muted-foreground">{pkg.description}</p>
          </div>
        </div>
        {!installed && (
          <Button
            size="sm"
            variant={pkg.required ? 'default' : 'outline'}
            onClick={() => handleInstall(pkg.key)}
            disabled={installingPkg !== null}
          >
            {installingPkg === pkg.key ? (
              <Loader2 className="mr-1 size-3 animate-spin" />
            ) : (
              <Download className="mr-1 size-3" />
            )}
            Install
          </Button>
        )}
      </div>
    )
  }

  function renderInstallLog() {
    if (installLog.length === 0) return null
    return (
      <div className="mt-3 rounded-lg border bg-black/90 p-3">
        <ScrollArea className="h-32">
          <div className="space-y-0.5 font-mono text-xs text-green-400">
            {installLog.map((line, i) => (
              <div key={i}>{line}</div>
            ))}
            <div ref={logEndRef} />
          </div>
        </ScrollArea>
        {installError && (
          <p className="mt-2 text-xs text-red-400">Installation encountered an error. Check the log above.</p>
        )}
      </div>
    )
  }

  function renderStepContent() {
    switch (step) {
      case 'discovery':
        return (
          <div className="space-y-4">
            {discover.isPending ? (
              <div className="flex flex-col items-center gap-3 py-8">
                <Loader2 className="size-8 animate-spin text-primary" />
                <p className="text-sm text-muted-foreground">Scanning machine capabilities...</p>
              </div>
            ) : discover.isError ? (
              <div className="flex flex-col items-center gap-3 py-8">
                <X className="size-8 text-red-500" />
                <p className="text-sm text-red-500">Failed to discover capabilities</p>
                <Button variant="outline" size="sm" onClick={() => discover.mutate()}>
                  Retry
                </Button>
              </div>
            ) : capabilities ? (
              <div className="space-y-3">
                <p className="text-sm text-muted-foreground">
                  Discovery complete. Here is what was found on {machineName}:
                </p>
                <div className="grid gap-2 sm:grid-cols-2">
                  {[
                    { key: 'tmux', label: 'tmux' },
                    { key: 'docker', label: 'Docker' },
                    { key: 'node', label: 'Node.js' },
                    { key: 'go', label: 'Go' },
                    { key: 'claude_code', label: 'Claude Code' },
                    { key: 'opencode', label: 'OpenCode' },
                    { key: 'codex', label: 'Codex' },
                  ].map(({ key, label }) => (
                    <div key={key} className="flex items-center gap-2 rounded border p-2 text-sm">
                      {isInstalled(key) ? (
                        <Check className="size-4 text-green-500" />
                      ) : (
                        <X className="size-4 text-red-500" />
                      )}
                      <span>{label}</span>
                      {isInstalled(key) && getVersion(key) && (
                        <span className="text-xs text-muted-foreground">({getVersion(key)})</span>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            ) : null}
          </div>
        )

      case 'core':
        return (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground">
              These are required for SpaceBallOne to function properly.
            </p>
            {corePackages.map(renderPackageRow)}
            {renderInstallLog()}
          </div>
        )

      case 'optional':
        return (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground">
              These tools enhance your development experience but are not required.
            </p>
            {optionalPackages.map(renderPackageRow)}
            {renderInstallLog()}
          </div>
        )

      case 'agents':
        return (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground">
              AI coding agents that can be used through the terminal.
            </p>
            {agentPackages.map(renderPackageRow)}
            {renderInstallLog()}
          </div>
        )

      case 'complete': {
        const allPackages = [...corePackages, ...optionalPackages, ...agentPackages]
        const installed = allPackages.filter((p) => p.installed)
        const skipped = allPackages.filter((p) => !p.installed)

        return (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Setup wizard complete for {machineName}.
            </p>
            {installed.length > 0 && (
              <div>
                <h4 className="mb-2 text-sm font-medium text-green-600">Installed ({installed.length})</h4>
                <div className="flex flex-wrap gap-2">
                  {installed.map((p) => (
                    <Badge key={p.key} variant="default" className="gap-1">
                      <Check className="size-3" />
                      {p.label}
                    </Badge>
                  ))}
                </div>
              </div>
            )}
            {skipped.length > 0 && (
              <div>
                <h4 className="mb-2 text-sm font-medium text-muted-foreground">Not Installed ({skipped.length})</h4>
                <div className="flex flex-wrap gap-2">
                  {skipped.map((p) => (
                    <Badge key={p.key} variant="secondary" className="gap-1">
                      <X className="size-3" />
                      {p.label}
                    </Badge>
                  ))}
                </div>
              </div>
            )}
          </div>
        )
      }
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>Setup Wizard - {STEP_TITLES[step]}</DialogTitle>
          <DialogDescription>
            Step {currentStepIndex + 1} of {STEPS.length} - {machineName}
          </DialogDescription>
        </DialogHeader>

        {/* Step indicators */}
        <div className="flex items-center gap-1">
          {STEPS.map((s, i) => (
            <div
              key={s}
              className={`h-1 flex-1 rounded-full ${
                i <= currentStepIndex ? 'bg-primary' : 'bg-muted'
              }`}
            />
          ))}
        </div>

        <div className="min-h-[250px]">{renderStepContent()}</div>

        <DialogFooter>
          {step !== 'discovery' && step !== 'complete' && (
            <Button
              type="button"
              variant="ghost"
              onClick={goBack}
              disabled={installingPkg !== null}
            >
              <ChevronLeft className="mr-1 size-4" />
              Back
            </Button>
          )}
          <div className="flex-1" />
          {step === 'complete' ? (
            <Button onClick={() => onOpenChange(false)}>Close</Button>
          ) : step === 'discovery' ? (
            <Button
              onClick={goNext}
              disabled={discover.isPending || !capabilities}
            >
              Next
              <ChevronRight className="ml-1 size-4" />
            </Button>
          ) : (
            <>
              <Button
                variant="ghost"
                onClick={goNext}
                disabled={installingPkg !== null}
              >
                Skip
              </Button>
              <Button
                onClick={goNext}
                disabled={installingPkg !== null}
              >
                Next
                <ChevronRight className="ml-1 size-4" />
              </Button>
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
