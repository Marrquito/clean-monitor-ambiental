import { Droplets, Radio, SunMedium, Thermometer, TriangleAlert } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import type { LatestReadingState } from '@/hooks/useLatestReading'
import { SensorCard } from '@/components/sensors/SensorCard'
import { StateMessage } from '@/components/common/StateMessage'

export function SensorCards({ state }: { state: LatestReadingState }) {
  const { data: reading, isLoading, isEmpty, error, refresh } = state

  if (!reading) {
    if (isLoading) {
      return (
        <div className="grid gap-3 sm:grid-cols-3">
          {Array.from({ length: 3 }, (_, index) => (
            <Card key={index}>
              <CardContent>
                <Skeleton className="h-10 w-2/3" />
              </CardContent>
            </Card>
          ))}
        </div>
      )
    }
    if (isEmpty) {
      return (
        <Card>
          <StateMessage
            icon={Radio}
            title="Aguardando medições"
            description="Nenhuma medição foi registrada ainda."
          />
        </Card>
      )
    }
    return (
      <Card>
        <StateMessage
          icon={TriangleAlert}
          tone="error"
          title="Não foi possível carregar as medições atuais."
          description={error?.message}
          onRetry={refresh}
        />
      </Card>
    )
  }

  return (
    <div className="space-y-2">
      <div className="grid gap-3 sm:grid-cols-3">
        <SensorCard
          label="Temperatura"
          value={reading.temperature}
          unit="°C"
          icon={Thermometer}
          iconClassName="text-chart-1"
        />
        <SensorCard
          label="Umidade"
          value={reading.humidity}
          unit="%"
          icon={Droplets}
          iconClassName="text-chart-2"
        />
        <SensorCard
          label="Luminosidade"
          value={reading.light}
          unit="lux"
          icon={SunMedium}
          iconClassName="text-primary"
        />
      </div>
      {error && (
        <p role="alert" className="text-xs text-status-warning">
          Falha ao atualizar. Exibindo a última medição recebida.
        </p>
      )}
    </div>
  )
}
