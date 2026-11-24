import { Button } from '@/components/ui/button'
import type { HistoryPeriodOption } from '@/lib/constants'

interface PeriodSelectorProps {
  value: HistoryPeriodOption
  options: readonly HistoryPeriodOption[]
  onChange: (option: HistoryPeriodOption) => void
}

export function PeriodSelector({ value, options, onChange }: PeriodSelectorProps) {
  return (
    <div role="group" aria-label="Período dos gráficos" className="inline-flex rounded-lg bg-muted/60 p-1">
      {options.map(option => {
        const active = option.id === value.id
        return (
          <Button
            key={option.id}
            type="button"
            size="sm"
            variant={active ? 'default' : 'ghost'}
            aria-pressed={active}
            onClick={() => onChange(option)}
            className="min-w-12"
          >
            {option.label}
          </Button>
        )
      })}
    </div>
  )
}
