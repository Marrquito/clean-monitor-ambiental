import { ApiError, getLatestReading } from '@/lib/api'
import { POLL_INTERVAL_MS } from '@/lib/constants'
import { usePolling } from '@/hooks/usePolling'

export function useLatestReading() {
  const polling = usePolling(signal => getLatestReading(signal), POLL_INTERVAL_MS.latest, 'latest')
  const isEmpty = polling.error instanceof ApiError && polling.error.status === 404

  return {
    ...polling,
    error: isEmpty ? null : polling.error,
    isEmpty,
  }
}

export type LatestReadingState = ReturnType<typeof useLatestReading>
