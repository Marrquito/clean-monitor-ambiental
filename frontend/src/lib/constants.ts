import type { HistoryBucket, SummaryPeriod } from '@/lib/api'

export const CHAT_MAX_LENGTH = 500

export const CHAT_SUGGESTIONS = [
  'Como está o ambiente agora?',
  'Qual foi a temperatura máxima hoje?',
  'Qual foi a umidade média nas últimas duas horas?',
  'A temperatura está aumentando?',
  'Compare a temperatura da última hora com a anterior.',
] as const

const HOUR_MS = 60 * 60 * 1000

export interface HistoryPeriodOption {
  id: '1h' | '6h' | '24h'
  label: string
  windowMs: number
  bucket: HistoryBucket
  bucketLabel: string
}

export const HISTORY_PERIODS: readonly HistoryPeriodOption[] = [
  { id: '1h', label: '1 h', windowMs: HOUR_MS, bucket: '1m', bucketLabel: '1 min' },
  { id: '6h', label: '6 h', windowMs: 6 * HOUR_MS, bucket: '5m', bucketLabel: '5 min' },
  { id: '24h', label: '24 h', windowMs: 24 * HOUR_MS, bucket: '15m', bucketLabel: '15 min' },
]

export const SUMMARY_PERIOD: SummaryPeriod = 'last_24_hours'
export const SUMMARY_PERIOD_LABEL = 'Últimas 24 horas'

export const POLL_INTERVAL_MS = {
  health: 10_000,
  latest: 5_000,
  history: 30_000,
  summary: 30_000,
} as const
