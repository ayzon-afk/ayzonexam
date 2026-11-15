// Баллы и даты экзамена. Правила совпадают с сервером (api.validBand, проверка exam_date).

// Цели 4.0–9.0 с шагом 0.5.
export const BANDS: number[] = Array.from({ length: 11 }, (_, i) => 4 + i * 0.5)

export function formatBand(band: number): string {
  return band.toFixed(1)
}

// Дата «ГГГГ-ММ-ДД» по местному календарю телефона.
export function isoDate(d: Date): string {
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
}

// Допустимые даты экзамена: с сегодня и на два года вперёд.
export function examDateRange(today: Date): { min: string; max: string } {
  const max = new Date(today)
  max.setFullYear(max.getFullYear() + 2)
  return { min: isoDate(today), max: isoDate(max) }
}

// Сколько дней до экзамена; 0 — сегодня, отрицательное — уже прошёл.
export function daysUntil(examDate: string, today: Date): number {
  const [y, m, d] = examDate.split('-').map(Number)
  const exam = Date.UTC(y!, m! - 1, d!)
  const now = Date.UTC(today.getFullYear(), today.getMonth(), today.getDate())
  return Math.round((exam - now) / 86_400_000)
}
