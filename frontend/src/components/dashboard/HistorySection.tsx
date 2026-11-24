import { useState } from 'react'
import { HistoryChartCard } from '@/components/charts/HistoryChartCard'
import { PeriodSelector } from '@/components/dashboard/PeriodSelector'
import { useHistory } from '@/hooks/useHistory'
import { HISTORY_PERIODS, type HistoryPeriodOption } from '@/lib/constants'

export function HistorySection() {
  const [period, setPeriod] = useState<HistoryPeriodOption>(HISTORY_PERIODS[0])
  const history = useHistory(period)
  const showDay = period.id === '24h'

  return (
    <section className="space-y-3" aria-label="Histórico de medições">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="text-sm font-medium text-muted-foreground">Histórico</h2>
        <PeriodSelector value={period} options={HISTORY_PERIODS} onChange={setPeriod} />
      </div>
      <div className="grid gap-4 md:grid-cols-2">
        <HistoryChartCard
          title="Temperatura"
          description={`Média a cada ${period.bucketLabel}`}
          metric="temperature"
          unit="°C"
          color="var(--chart-1)"
          state={history}
          showDay={showDay}
        />
        <HistoryChartCard
          title="Umidade"
          description={`Média a cada ${period.bucketLabel}`}
          metric="humidity"
          unit="%"
          color="var(--chart-2)"
          state={history}
          showDay={showDay}
        />
      </div>
    </section>
  )
}
