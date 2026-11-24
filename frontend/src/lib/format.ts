export const DISPLAY_TIME_ZONE = 'America/Sao_Paulo'

const numberFormatter = new Intl.NumberFormat('pt-BR', { maximumFractionDigits: 1, minimumFractionDigits: 0 })

const dateTimeFormatter = new Intl.DateTimeFormat('pt-BR', {
  timeZone: DISPLAY_TIME_ZONE,
  day: '2-digit',
  month: '2-digit',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
})

const timeFormatter = new Intl.DateTimeFormat('pt-BR', {
  timeZone: DISPLAY_TIME_ZONE,
  hour: '2-digit',
  minute: '2-digit',
})

const dayTimeFormatter = new Intl.DateTimeFormat('pt-BR', {
  timeZone: DISPLAY_TIME_ZONE,
  day: '2-digit',
  month: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
})

export function formatNumber(value: number): string {
  return numberFormatter.format(value)
}

export function formatMeasure(value: number, unit: string): string {
  return `${formatNumber(value)} ${unit}`
}

export function formatDateTime(isoDate: string): string {
  return dateTimeFormatter.format(new Date(isoDate))
}

export function formatTime(isoDate: string): string {
  return timeFormatter.format(new Date(isoDate))
}

export function formatChartTick(isoDate: string, showDay: boolean): string {
  return showDay ? dayTimeFormatter.format(new Date(isoDate)) : timeFormatter.format(new Date(isoDate))
}
