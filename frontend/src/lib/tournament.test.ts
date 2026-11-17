import { describe, expect, it } from 'vitest'
import { tourStart } from './tournament'
import { routeFromOpen } from './nav.svelte'

// Те же примеры — в backend/internal/rating/tournament_test.go: расписание одно.
describe('tourStart', () => {
  it('последнее воскресенье месяца, 10:00 по Ташкенту', () => {
    expect(tourStart(new Date('2026-11-16T12:00:00Z')).toISOString()).toBe('2026-11-29T05:00:00.000Z')
  })
  it('пока идёт — он же; кончился — следующий месяц, через границу года тоже', () => {
    expect(tourStart(new Date('2026-11-29T06:00:00Z')).toISOString()).toBe('2026-11-29T05:00:00.000Z')
    expect(tourStart(new Date('2026-11-29T08:00:00Z')).toISOString()).toBe('2026-12-27T05:00:00.000Z')
    expect(tourStart(new Date('2026-12-28T00:00:00Z')).toISOString()).toBe('2027-01-31T05:00:00.000Z')
  })
})

describe('routeFromOpen', () => {
  it('кнопки бота открывают нужный экран', () => {
    expect(routeFromOpen('mock-12')).toEqual({ tab: 'exams', route: { s: 'report', id: 12 } })
    expect(routeFromOpen('duel-ab3k')).toEqual({ tab: 'rating', route: { s: 'duel', code: 'ab3k' } })
    expect(routeFromOpen('d-ab3k')).toEqual({ tab: 'rating', route: { s: 'duel', code: 'ab3k' } })
    expect(routeFromOpen('center-requests')).toEqual({ tab: 'profile', route: { s: 'centerRequests' } })
    expect(routeFromOpen('speaking-5')).toEqual({ tab: 'exams', route: { s: 'speakingCheck', id: 5 } })
    expect(routeFromOpen('premium')).toEqual({ tab: 'profile', route: { s: 'premium' } })
    expect(routeFromOpen('что-то')).toBeNull()
  })
})
