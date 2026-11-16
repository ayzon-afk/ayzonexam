import { describe, expect, it } from 'vitest'
import { nextTournament } from './demo'
import { setBand } from './demo-exam'

describe('nextTournament', () => {
  it('последнее воскресенье месяца, 10:00 по Ташкенту', () => {
    expect(nextTournament(new Date('2026-11-16T12:00:00Z')).toISOString()).toBe('2026-11-29T05:00:00.000Z')
  })
  it('после начала — уже следующий месяц, через границу года тоже', () => {
    expect(nextTournament(new Date('2026-11-29T05:00:00Z')).toISOString()).toBe('2026-12-27T05:00:00.000Z')
    expect(nextTournament(new Date('2026-12-28T00:00:00Z')).toISOString()).toBe('2027-01-31T05:00:00.000Z')
  })
})

describe('setBand', () => {
  it('от 4.0 до 9.0 с шагом 0.5', () => {
    expect(setBand(0, 8)).toBe(4)
    expect(setBand(5, 10)).toBe(6.5)
    expect(setBand(10, 10)).toBe(9)
  })
})
