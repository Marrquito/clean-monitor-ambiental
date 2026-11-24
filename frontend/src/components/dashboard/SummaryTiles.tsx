import { Gauge, TriangleAlert } from 'lucide-react'
import { StateMessage } from '@/components/common/StateMessage'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import type { PollingState } from '@/hooks/usePolling'
import type { DashboardSummary, MetricRange } from '@/lib/api'
import { SUMMARY_PERIOD_LABEL } from '@/lib/constants'
import { formatMeasure, formatNumber } from '@/lib/format'

interface SummaryTilesProps {
  state: PollingState<DashboardSummary>
}

function MetricRow({ title, unit, range }: { title: string; unit: string; range: MetricRange | null }) {
  const tiles = [
    { label: 'Mínima', value: range?.min },
    { label: 'Média', value: range?.avg },
    { label: 'Máxima', value: range?.max },
  ]

  return (
    <div className="space-y-2">
      <p className="text-sm font-medium">{title}</p>
      <div className="grid grid-cols-3 gap-2">
        {tiles.map(tile => (
          <div key={tile.label} className="rounded-lg bg-muted/50 px-3 py-2">
            <p className="text-xs text-muted-foreground">{tile.label}</p>
            <p className="text-lg font-semibold sm:text-xl">
              {tile.value === undefined ? '—' : formatMeasure(tile.value, unit)}
            </p>
          </div>
        ))}
      </div>
    </div>
  )
}

export function SummaryTiles({ state }: SummaryTilesProps) {
  const { data: summary, isLoading, error, refresh } = state

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <Gauge className="size-4 text-primary" aria-hidden />
          Resumo do período
        </CardTitle>
        <CardDescription>
          {summary ? `${SUMMARY_PERIOD_LABEL} · ${formatNumber(summary.count)} medições` : SUMMARY_PERIOD_LABEL}
        </CardDescription>
      </CardHeader>
      <CardContent>
        {!summary && isLoading && (
          <div className="space-y-4">
            <Skeleton className="h-16 w-full" />
            <Skeleton className="h-16 w-full" />
          </div>
        )}
        {!summary && !isLoading && error && (
          <StateMessage
            icon={TriangleAlert}
            tone="error"
            title="Não foi possível carregar o resumo."
            description={error.message}
            onRetry={refresh}
          />
        )}
        {summary && (
          <div className="space-y-4">
            {summary.count === 0 && <p className="text-sm text-muted-foreground">Sem medições neste período.</p>}
            <MetricRow title="Temperatura" unit="°C" range={summary.temperature} />
            <MetricRow title="Umidade" unit="%" range={summary.humidity} />
          </div>
        )}
      </CardContent>
    </Card>
  )
}
