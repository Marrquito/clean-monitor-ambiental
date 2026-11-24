import { getHealth, type HealthResponse } from '@/lib/api'
import { POLL_INTERVAL_MS } from '@/lib/constants'
import { usePolling } from '@/hooks/usePolling'

export type BackendStatus = 'checking' | 'online' | 'offline'

export interface HealthState {
  health: HealthResponse | null
  backend: BackendStatus
}

export function useHealth(): HealthState {
  const polling = usePolling(signal => getHealth(signal), POLL_INTERVAL_MS.health, 'health')

  let backend: BackendStatus = 'checking'
  if (polling.error) {
    backend = 'offline'
  } else if (polling.data) {
    backend = 'online'
  }

  return { health: polling.data, backend }
}
