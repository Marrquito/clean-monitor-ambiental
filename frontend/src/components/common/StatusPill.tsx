import type { LucideIcon } from 'lucide-react'
import { cn } from '@/lib/utils'

export type StatusTone = 'good' | 'warning' | 'critical' | 'neutral'

const TONE_CLASSES: Record<StatusTone, string> = {
  good: 'border-status-good/40 bg-status-good/15 text-status-good',
  warning: 'border-status-warning/40 bg-status-warning/15 text-status-warning',
  critical: 'border-status-critical/40 bg-status-critical/15 text-status-critical',
  neutral: 'border-border bg-muted text-muted-foreground',
}

interface StatusPillProps {
  tone: StatusTone
  icon: LucideIcon
  children: string
}

export function StatusPill({ tone, icon: Icon, children }: StatusPillProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs font-medium whitespace-nowrap',
        TONE_CLASSES[tone],
      )}
    >
      <Icon className="size-3.5 shrink-0" aria-hidden />
      {children}
    </span>
  )
}
