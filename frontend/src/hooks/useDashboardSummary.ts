import { getDashboardSummary } from '@/lib/api'
import { POLL_INTERVAL_MS, SUMMARY_PERIOD } from '@/lib/constants'
import { usePolling } from '@/hooks/usePolling'

export function useDashboardSummary() {
  return usePolling(
    signal => getDashboardSummary(SUMMARY_PERIOD, signal),
    POLL_INTERVAL_MS.summary,
    SUMMARY_PERIOD,
  )
}
