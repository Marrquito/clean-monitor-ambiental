import type { ReactNode } from 'react'
import { ChartColumn, TriangleAlert } from 'lucide-react'
import { StateMessage } from '@/components/common/StateMessage'
import { MetricAreaChart, type ChartPoint } from '@/components/charts/MetricAreaChart'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import type { PollingState } from '@/hooks/usePolling'
import type { HistoryPoint, HistoryResponse } from '@/lib/api'

type Metric = 'temperature' | 'humidity'

interface HistoryChartCardProps {
  title: string
  description: string
  metric: Metric
  unit: string
  color: string
  state: PollingState<HistoryResponse>
  showDay: boolean
}

function toChartPoints(points: HistoryPoint[], metric: Metric): ChartPoint[] {
  return points.map(point => ({ time: point.time, value: point[metric] }))
}

export function HistoryChartCard({
  title,
  description,
  metric,
  unit,
  color,
  state,
  showDay,
}: HistoryChartCardProps) {
  const { data, isLoading, error, refresh } = state
  const points = data?.points ?? []

  let content: ReactNode
  if (!data && isLoading) {
    content = <Skeleton className="h-full w-full" />
  } else if (!data && error) {
    content = (
      <StateMessage
        icon={TriangleAlert}
        tone="error"
        title="Não foi possível carregar o histórico."
        description={error.message}
        onRetry={refresh}
      />
    )
  } else if (points.length === 0) {
    content = (
      <StateMessage
        icon={ChartColumn}
        title="Sem medições neste período."
        description="Nenhum dado foi registrado no intervalo selecionado."
      />
    )
  } else {
    content = (
      <MetricAreaChart
        data={toChartPoints(points, metric)}
        label={title}
        unit={unit}
        color={color}
        showDay={showDay}
      />
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent className="h-56 sm:h-64">{content}</CardContent>
    </Card>
  )
}
