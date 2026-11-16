import { getLang } from './i18n/index.svelte'

// Дата на языке интерфейса по Ташкенту: «12 ноября», с time — «12 ноября, 10:00».
export function fmtDate(d: Date | string, time = false): string {
  return new Date(d).toLocaleString(getLang(), {
    day: 'numeric',
    month: 'long',
    timeZone: 'Asia/Tashkent',
    ...(time ? { hour: '2-digit', minute: '2-digit' } : {}),
  })
}

// «октябрь 2026» — для итогов турниров.
export function fmtMonth(d: string): string {
  return new Date(d).toLocaleString(getLang(), { month: 'long', year: 'numeric', timeZone: 'Asia/Tashkent' })
}

// Таймер «38:05»; меньше нуля не бывает.
export function clock(ms: number): string {
  const s = Math.max(0, Math.round(ms / 1000))
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`
}
