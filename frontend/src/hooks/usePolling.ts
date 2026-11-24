import { useCallback, useEffect, useRef, useState } from 'react'

export interface PollingState<T> {
  data: T | null
  error: Error | null
  isLoading: boolean
  updatedAt: Date | null
  refresh: () => void
}

type Fetcher<T> = (signal: AbortSignal) => Promise<T>

interface Snapshot<T> {
  key: string
  data: T | null
  error: Error | null
  updatedAt: Date | null
}

function toError(error: unknown): Error {
  return error instanceof Error ? error : new Error(String(error))
}

export function usePolling<T>(fetcher: Fetcher<T>, intervalMs: number, key: string): PollingState<T> {
  const [snapshot, setSnapshot] = useState<Snapshot<T> | null>(null)
  const [reloadToken, setReloadToken] = useState(0)
  const fetcherRef = useRef(fetcher)

  useEffect(() => {
    fetcherRef.current = fetcher
  })

  useEffect(() => {
    let active = true
    const controller = new AbortController()

    const load = async () => {
      try {
        const data = await fetcherRef.current(controller.signal)
        if (active) {
          setSnapshot({ key, data, error: null, updatedAt: new Date() })
        }
      } catch (error) {
        if (active) {
          setSnapshot(previous => {
            const sameKey = previous?.key === key
            return {
              key,
              data: sameKey ? previous.data : null,
              error: toError(error),
              updatedAt: sameKey ? previous.updatedAt : null,
            }
          })
        }
      }
    }

    void load()
    const timer = window.setInterval(() => void load(), intervalMs)

    return () => {
      active = false
      controller.abort()
      window.clearInterval(timer)
    }
  }, [key, intervalMs, reloadToken])

  const refresh = useCallback(() => setReloadToken(token => token + 1), [])

  const current = snapshot?.key === key ? snapshot : null

  return {
    data: current?.data ?? null,
    error: current?.error ?? null,
    isLoading: current === null,
    updatedAt: current?.updatedAt ?? null,
    refresh,
  }
}
