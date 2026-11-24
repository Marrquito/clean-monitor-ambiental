import type { LucideIcon } from 'lucide-react'
import { Card, CardContent, CardTitle } from '@/components/ui/card'
import { formatNumber } from '@/lib/format'
import { cn } from '@/lib/utils'

interface SensorCardProps {
  label: string
  value: number
  unit: string
  icon: LucideIcon
  iconClassName: string
}

export function SensorCard({ label, value, unit, icon: Icon, iconClassName }: SensorCardProps) {
  return (
    <Card>
      <CardTitle className="flex items-center gap-2 px-4 text-sm font-medium text-muted-foreground">
        <Icon className={cn('size-4', iconClassName)} aria-hidden />
        {label}
      </CardTitle>
      <CardContent>
        <p className="text-4xl font-semibold tracking-tight">
          {formatNumber(value)}
          <span className="ml-1.5 text-lg font-normal text-muted-foreground">{unit}</span>
        </p>
      </CardContent>
    </Card>
  )
}
