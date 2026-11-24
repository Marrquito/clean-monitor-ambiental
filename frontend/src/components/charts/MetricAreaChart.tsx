import { Area, AreaChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import { formatChartTick, formatDateTime, formatMeasure, formatNumber } from '@/lib/format'

export interface ChartPoint {
  time: string
  value: number
}

interface MetricAreaChartProps {
  data: ChartPoint[]
  label: string
  unit: string
  color: string
  showDay: boolean
}

const AXIS_TICK = { fill: 'var(--muted-foreground)', fontSize: 11 }

export function MetricAreaChart({ data, label, unit, color, showDay }: MetricAreaChartProps) {
  return (
    <ResponsiveContainer width="100%" height="100%">
      <AreaChart data={data} margin={{ top: 8, right: 8, bottom: 0, left: 0 }}>
        <CartesianGrid vertical={false} stroke="var(--border)" />
        <XAxis
          dataKey="time"
          tickFormatter={(value: string) => formatChartTick(value, showDay)}
          tick={AXIS_TICK}
          tickLine={false}
          axisLine={false}
          minTickGap={32}
        />
        <YAxis
          domain={['dataMin - 1', 'dataMax + 1']}
          tickFormatter={(value: number) => formatNumber(value)}
          tick={AXIS_TICK}
          tickLine={false}
          axisLine={false}
          width={40}
        />
        <Tooltip
          cursor={{ stroke: 'var(--muted-foreground)', strokeWidth: 1 }}
          labelFormatter={(time: unknown) => formatDateTime(String(time))}
          formatter={(value: unknown) => [formatMeasure(Number(value), unit), label]}
          contentStyle={{
            backgroundColor: 'var(--popover)',
            borderColor: 'var(--border)',
            borderRadius: 8,
            fontSize: 12,
          }}
          labelStyle={{ color: 'var(--muted-foreground)' }}
          itemStyle={{ color: 'var(--foreground)' }}
        />
        <Area
          type="monotone"
          dataKey="value"
          name={label}
          stroke={color}
          strokeWidth={2}
          fill={color}
          fillOpacity={0.1}
          dot={false}
          activeDot={{ r: 4, fill: color, stroke: 'var(--card)', strokeWidth: 2 }}
          isAnimationActive={false}
        />
      </AreaChart>
    </ResponsiveContainer>
  )
}
