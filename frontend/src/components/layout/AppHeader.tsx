import { Activity, Clock, Database, FlaskConical, Radio, Sparkles, Wifi, WifiOff } from 'lucide-react'
import { StatusPill } from '@/components/common/StatusPill'
import type { BackendStatus } from '@/hooks/useHealth'
import type { HealthResponse } from '@/lib/api'
import { formatDateTime } from '@/lib/format'

interface AppHeaderProps {
  backend: BackendStatus
  health: HealthResponse | null
  sourceKind: string | null
  lastUpdated: string | null
}

function BackendPill({ backend }: { backend: BackendStatus }) {
  if (backend === 'online') {
    return (
      <StatusPill tone="good" icon={Wifi}>
        Backend online
      </StatusPill>
    )
  }
  if (backend === 'offline') {
    return (
      <StatusPill tone="critical" icon={WifiOff}>
        Backend offline
      </StatusPill>
    )
  }
  return (
    <StatusPill tone="neutral" icon={Clock}>
      Verificando backend
    </StatusPill>
  )
}

function SystemPill({ health }: { health: HealthResponse }) {
  if (health.database !== 'ok') {
    return (
      <StatusPill tone="critical" icon={Database}>
        Banco indisponível
      </StatusPill>
    )
  }
  return (
    <StatusPill tone="good" icon={Activity}>
      Sistema operacional
    </StatusPill>
  )
}

function SourcePill({ sourceKind }: { sourceKind: string | null }) {
  if (sourceKind === 'simulated') {
    return (
      <StatusPill tone="warning" icon={FlaskConical}>
        Dados simulados
      </StatusPill>
    )
  }
  if (sourceKind) {
    return (
      <StatusPill tone="good" icon={Radio}>
        Dados reais
      </StatusPill>
    )
  }
  return (
    <StatusPill tone="neutral" icon={FlaskConical}>
      Origem indisponível
    </StatusPill>
  )
}

export function AppHeader({ backend, health, sourceKind, lastUpdated }: AppHeaderProps) {
  return (
    <header className="border-b border-border bg-card/60">
      <div className="mx-auto flex max-w-7xl flex-col gap-4 px-4 py-4 sm:px-6 lg:flex-row lg:items-center lg:justify-between">
        <div className="flex min-w-0 items-center gap-3">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-primary/15 text-primary">
            <Sparkles className="size-5" aria-hidden />
          </div>
          <div className="min-w-0">
            <h1 className="text-lg font-semibold tracking-tight">CLEAN</h1>
            <p className="truncate text-xs text-muted-foreground sm:text-sm">
              Conversational Local Environmental Analyzer Node
            </p>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <BackendPill backend={backend} />
          {health && <SystemPill health={health} />}
          <SourcePill sourceKind={sourceKind} />
          <span className="text-xs text-muted-foreground">
            Última atualização:{' '}
            {lastUpdated ? (
              <time dateTime={lastUpdated} className="font-medium text-foreground">
                {formatDateTime(lastUpdated)}
              </time>
            ) : (
              '—'
            )}
          </span>
        </div>
      </div>
    </header>
  )
}
