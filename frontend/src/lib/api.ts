export class ApiError extends Error {
  readonly status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

export type SensorSource = 'simulated' | (string & {})
export type HistoryBucket = '1m' | '5m' | '15m' | '1h'
export type SummaryPeriod = 'last_hour' | 'last_2_hours' | 'last_24_hours' | 'today'

export interface HealthResponse {
  status: string
  database: 'ok' | 'error'
  ollama: 'ok' | 'unavailable'
  model: string
  model_available: boolean
  time: string
}

export interface SensorReading {
  measured_at: string
  temperature: number
  humidity: number
  light: number
  source: SensorSource
}

export interface HistoryPoint {
  time: string
  temperature: number
  humidity: number
  light: number
}

export interface HistoryResponse {
  bucket: HistoryBucket
  points: HistoryPoint[]
}

export interface MetricRange {
  min: number
  max: number
  avg: number
}

export interface DashboardSummary {
  period: { name: SummaryPeriod; from: string; to: string }
  count: number
  current: Omit<SensorReading, 'source'> | null
  temperature: MetricRange | null
  humidity: MetricRange | null
  source: { kind: SensorSource; connected: boolean; last_measured_at: string | null }
  generated_at: string
}

export interface ChatResponse {
  answer: string
  model: string
  intent: string
  period: { name: string; from: string; to: string } | null
}

export interface HistoryQuery {
  from: Date
  to: Date
  bucket: HistoryBucket
}

interface RequestOptions {
  method?: 'GET' | 'POST'
  body?: unknown
  signal?: AbortSignal
}

const NETWORK_ERROR_MESSAGE =
  'Não foi possível conectar ao servidor. Verifique se o backend está em execução.'
const UNEXPECTED_RESPONSE_MESSAGE = 'Resposta inesperada do servidor.'

function isAbortError(error: unknown): boolean {
  return error instanceof DOMException && error.name === 'AbortError'
}

function readErrorMessage(body: unknown): string | null {
  if (typeof body === 'object' && body !== null && 'error' in body && typeof body.error === 'string') {
    return body.error
  }
  return null
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, signal } = options
  const headers: Record<string, string> = { Accept: 'application/json' }
  if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
  }

  let response: Response
  try {
    response = await fetch(`/api${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      signal,
    })
  } catch (error) {
    if (isAbortError(error)) {
      throw error
    }
    throw new ApiError(NETWORK_ERROR_MESSAGE, 0)
  }

  const payload: unknown = await response.json().catch(() => null)

  if (!response.ok) {
    throw new ApiError(readErrorMessage(payload) ?? UNEXPECTED_RESPONSE_MESSAGE, response.status)
  }
  if (payload === null) {
    throw new ApiError(UNEXPECTED_RESPONSE_MESSAGE, response.status)
  }
  return payload as T
}

export function getHealth(signal?: AbortSignal): Promise<HealthResponse> {
  return request<HealthResponse>('/health', { signal })
}

export function getLatestReading(signal?: AbortSignal): Promise<SensorReading> {
  return request<SensorReading>('/sensors/latest', { signal })
}

export function getHistory({ from, to, bucket }: HistoryQuery, signal?: AbortSignal): Promise<HistoryResponse> {
  const params = new URLSearchParams({
    from: from.toISOString(),
    to: to.toISOString(),
    bucket,
  })
  return request<HistoryResponse>(`/sensors/history?${params.toString()}`, { signal })
}

export function getDashboardSummary(period: SummaryPeriod, signal?: AbortSignal): Promise<DashboardSummary> {
  const params = new URLSearchParams({ period })
  return request<DashboardSummary>(`/dashboard/summary?${params.toString()}`, { signal })
}

export function sendChatMessage(message: string, signal?: AbortSignal): Promise<ChatResponse> {
  return request<ChatResponse>('/chat', { method: 'POST', body: { message }, signal })
}
