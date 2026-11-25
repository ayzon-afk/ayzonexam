import { describe, expect, it, vi } from 'vitest'
import { clock } from './date'
import { routeFromOpen } from './nav.svelte'
import { inviteLink } from './share'

// Ссылки-приглашения и кнопки бота: ученик, учитель и центр попадают на нужный экран.

describe('inviteLink', () => {
  it('через бота — t.me/<бот>?start=…', () => {
    expect(inviteLink('humostartup_bot', 'c-abc123')).toBe('https://t.me/humostartup_bot?start=c-abc123')
  })
  it('параметр кодируется', () => {
    expect(inviteLink('b', 'r-a b&c')).toBe('https://t.me/b?start=r-a%20b%26c')
  })
  it('без бота (локально) — адрес приложения с ?open=', () => {
    vi.stubGlobal('location', { origin: 'http://localhost:5173' })
    expect(inviteLink('', 'd-xy')).toBe('http://localhost:5173/?open=d-xy')
    vi.unstubAllGlobals()
  })
})

describe('routeFromOpen: все кнопки бота', () => {
  const cases: [string, ReturnType<typeof routeFromOpen>][] = [
    ['premium', { tab: 'profile', route: { s: 'premium' } }],
    ['friends', { tab: 'rating', route: { s: 'friends' } }],
    ['support', { tab: 'profile', route: { s: 'support' } }],
    ['center', { tab: 'profile', route: { s: 'centerRequests' } }],
    ['mock', { tab: 'exams', route: { s: 'mock' } }],
    ['mock-7', { tab: 'exams', route: { s: 'report', id: 7 } }],
    ['speaking', { tab: 'exams', route: { s: 'speaking' } }],
    ['duel', { tab: 'rating', route: { s: 'duel' } }],
  ]
  for (const [open, want] of cases) it(open, () => expect(routeFromOpen(open)).toEqual(want))

  it('мусор в id — не открывает чужой разбор', () => {
    expect(routeFromOpen('mock-abc')).toEqual({ tab: 'exams', route: { s: 'mock' } })
    expect(routeFromOpen('mock-0')).toEqual({ tab: 'exams', route: { s: 'mock' } })
    expect(routeFromOpen('mock--3')).toEqual({ tab: 'exams', route: { s: 'mock' } })
    expect(routeFromOpen('speaking-1.5')).toEqual({ tab: 'exams', route: { s: 'speaking' } })
  })
  it('ссылки друга и центра обрабатывает App, не навигация', () => {
    expect(routeFromOpen('r-abc')).toBeNull()
    expect(routeFromOpen('c-abc')).toBeNull()
    expect(routeFromOpen('')).toBeNull()
  })
})

describe('clock', () => {
  it('минуты и секунды', () => {
    expect(clock(0)).toBe('0:00')
    expect(clock(65_000)).toBe('1:05')
    expect(clock(60 * 60_000)).toBe('60:00')
  })
  it('меньше нуля не бывает, доли секунды округляются', () => {
    expect(clock(-5000)).toBe('0:00')
    expect(clock(1499)).toBe('0:01')
    expect(clock(1500)).toBe('0:02')
  })
})
