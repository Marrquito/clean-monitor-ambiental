import { AppHeader } from '@/components/layout/AppHeader'
import { ChatPanel } from '@/components/chat/ChatPanel'
import { HistorySection } from '@/components/dashboard/HistorySection'
import { SummaryTiles } from '@/components/dashboard/SummaryTiles'
import { SensorCards } from '@/components/sensors/SensorCards'
import { useDashboardSummary } from '@/hooks/useDashboardSummary'
import { useHealth } from '@/hooks/useHealth'
import { useLatestReading } from '@/hooks/useLatestReading'

export default function App() {
  const { health, backend } = useHealth()
  const latest = useLatestReading()
  const summary = useDashboardSummary()

  const sourceKind = summary.data?.source.kind ?? latest.data?.source ?? null
  const lastUpdated = latest.data?.measured_at ?? summary.data?.generated_at ?? null

  return (
    <div className="min-h-dvh bg-background text-foreground">
      <AppHeader backend={backend} health={health} sourceKind={sourceKind} lastUpdated={lastUpdated} />

      <main className="mx-auto grid max-w-7xl gap-4 px-4 py-4 sm:px-6 sm:py-6 lg:grid-cols-[minmax(0,1fr)_400px] lg:gap-6">
        <div className="flex min-w-0 flex-col gap-4 lg:gap-6">
          <SensorCards state={latest} />
          <SummaryTiles state={summary} />
          <HistorySection />
        </div>
        <div className="min-w-0">
          <ChatPanel aiAvailable={health ? health.ollama === 'ok' : null} />
        </div>
      </main>
    </div>
  )
}
