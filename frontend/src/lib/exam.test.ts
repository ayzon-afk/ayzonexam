import { describe, expect, it } from 'vitest'
import { BANDS, daysUntil, examDateRange, formatBand, isoDate } from './exam'

describe('BANDS', () => {
  it('4.0–9.0 с шагом 0.5', () => {
    expect(BANDS.map(formatBand)).toEqual([
      '4.0', '4.5', '5.0', '5.5', '6.0', '6.5', '7.0', '7.5', '8.0', '8.5', '9.0',
    ])
  })
})

describe('даты', () => {
  const today = new Date(2026, 10, 15, 23, 30) // поздний вечер — граница суток

  it('isoDate по местному календарю', () => {
    expect(isoDate(today)).toBe('2026-11-15')
  })

  it('диапазон: с сегодня на два года', () => {
    expect(examDateRange(today)).toEqual({ min: '2026-11-15', max: '2028-11-15' })
  })

  it('daysUntil', () => {
    expect(daysUntil('2026-11-15', today)).toBe(0)
    expect(daysUntil('2026-11-16', today)).toBe(1)
    expect(daysUntil('2026-12-20', today)).toBe(35)
    expect(daysUntil('2026-11-14', today)).toBe(-1)
  })

  it('daysUntil через переход на летнее время не теряет день', () => {
    expect(daysUntil('2027-04-01', new Date(2027, 2, 1))).toBe(31)
  })
})
