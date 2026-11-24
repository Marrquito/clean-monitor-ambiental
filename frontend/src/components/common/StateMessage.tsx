import type { LucideIcon } from 'lucide-react'
import { RotateCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

interface StateMessageProps {
  icon: LucideIcon
  title: string
  description?: string
  tone?: 'neutral' | 'error'
  onRetry?: () => void
}

export function StateMessage({ icon: Icon, title, description, tone = 'neutral', onRetry }: StateMessageProps) {
  return (
    <div
      role={tone === 'error' ? 'alert' : 'status'}
      className="flex h-full min-h-24 flex-col items-center justify-center gap-2 px-4 py-6 text-center"
    >
      <Icon
        className={cn('size-5', tone === 'error' ? 'text-status-critical' : 'text-muted-foreground')}
        aria-hidden
      />
      <p className="text-sm font-medium">{title}</p>
      {description && <p className="max-w-sm text-sm text-muted-foreground">{description}</p>}
      {onRetry && (
        <Button variant="outline" size="sm" onClick={onRetry} className="mt-1">
          <RotateCw aria-hidden />
          Tentar novamente
        </Button>
      )}
    </div>
  )
}
