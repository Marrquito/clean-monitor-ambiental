import { getHistory } from '@/lib/api'
import { POLL_INTERVAL_MS, type HistoryPeriodOption } from '@/lib/constants'
import { usePolling } from '@/hooks/usePolling'

export function useHistory(period: HistoryPeriodOption) {
  return usePolling(
    signal => {
      const to = new Date()
      const from = new Date(to.getTime() - period.windowMs)
      return getHistory({ from, to, bucket: period.bucket }, signal)
    },
    POLL_INTERVAL_MS.history,
    period.id,
  )
}
