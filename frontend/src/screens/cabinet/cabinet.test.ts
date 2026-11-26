import { describe, expect, it } from 'vitest'
import { centerPrice, IDLE_DAYS, idle, MAX_SEATS, sectionsFor, sum, toCSV, view, type Student } from './cabinet'

const now = Date.parse('2026-11-25T12:00:00Z')
const daysAgo = (n: number) => new Date(now - n * 24 * 60 * 60 * 1000).toISOString()

const st = (id: number, nickname: string, band: number | null, last: number | null, month = 0): Student => ({
  id,
  nickname,
  avatar: '',
  role: null,
  band,
  parts: band === null ? {} : { reading: band },
  month,
  last_at: last === null ? null : daysAgo(last),
})

const list = [st(1, 'Zafar', 6.5, 1, 4), st(2, 'aziza', 7.5, 20, 1), st(3, 'Bek', null, null), st(4, 'Dilnoza', 6.5, 3, 9), st(5, 'Ali', null, 2, 2)]
const ids = (l: Student[]) => l.map((s) => s.id)

describe('кабинет: idle', () => {
  it('ни разу не занимался — idle', () => expect(idle(st(1, 'a', null, null), now)).toBe(true))
  it(`граница — ${IDLE_DAYS} дней`, () => {
    expect(idle(st(1, 'a', null, IDLE_DAYS - 0.01), now)).toBe(false)
    expect(idle(st(1, 'a', null, IDLE_DAYS + 0.01), now)).toBe(true)
  })
})

describe('кабинет: view', () => {
  it('по баллу: выше — первым, равные — кто занимался позже, без балла — в конце', () => {
    expect(ids(view(list, '', 'all', 'band', now))).toEqual([2, 1, 4, 5, 3])
  })
  it('по активности: недавние первыми, ни разу — в конце', () => {
    expect(ids(view(list, '', 'all', 'active', now))).toEqual([1, 5, 4, 2, 3])
  })
  it('по нику — без учёта регистра', () => {
    expect(view(list, '', 'all', 'name', now).map((s) => s.nickname)).toEqual(['Ali', 'aziza', 'Bek', 'Dilnoza', 'Zafar'])
  })
  it('поиск — часть ника, регистр и пробелы не важны', () => {
    expect(ids(view(list, '  AZ ', 'all', 'band', now))).toEqual([2])
    expect(ids(view(list, 'nobody', 'all', 'band', now))).toEqual([])
  })
  it('отборы «без балла» и «не занимались»', () => {
    expect(ids(view(list, '', 'noBand', 'band', now))).toEqual([5, 3])
    expect(ids(view(list, '', 'idle', 'band', now))).toEqual([2, 3])
  })
  it('исходный список не меняется', () => {
    const before = ids(list)
    view(list, '', 'all', 'name', now)
    expect(ids(list)).toEqual(before)
  })
})

describe('кабинет: CSV', () => {
  const csv = toCSV([st(1, 'Ali; "the best"', 6.5, 0, 3), st(2, 'Bek', null, null)], ['Nick', 'Band', 'L', 'R', 'W', 'S', 'Tasks', 'Last'])
  const lines = csv.slice(1).trimEnd().split('\r\n')

  it('BOM для Excel, «;» и строки через CRLF', () => {
    expect(csv.startsWith('﻿')).toBe(true)
    expect(lines[0]).toBe('Nick;Band;L;R;W;S;Tasks;Last')
    expect(lines).toHaveLength(3)
  })
  it('кавычки и «;» в нике экранируются, пустые баллы — пустые ячейки', () => {
    expect(lines[1]).toBe(`"Ali; ""the best""";6.5;;6.5;;;3;2026-11-25`)
    expect(lines[2]).toBe('Bek;;;;;;0;')
  })
})

describe('кабинет: разделы и суммы', () => {
  it('учителю — без «Центр и тариф»', () => {
    expect(sectionsFor('teacher').map((s) => s.id)).toEqual(['overview', 'students', 'requests'])
    expect(sectionsFor('admin').map((s) => s.id)).toEqual(['overview', 'students', 'requests', 'settings'])
  })
  it('суммы — пробелами по тысячам', () => {
    expect(sum(400000)).toBe('400 000')
    expect(sum(1234567)).toBe('1 234 567')
    expect(sum(999)).toBe('999')
  })
})

describe('кабинет: тариф центра — как billing.CenterPrice', () => {
  it('за место в месяц, от 100 мест дешевле', () => {
    expect(centerPrice(20, 1)).toBe(400_000)
    expect(centerPrice(99, 1)).toBe(1_980_000)
    expect(centerPrice(100, 1)).toBe(1_500_000)
    expect(centerPrice(25, 12)).toBe(6_000_000)
  })
  it('меньше 20 мест, чужой срок, дробное — нельзя', () => {
    for (const [seats, months] of [[19, 1], [20, 2], [20, 0], [20.5, 1], [MAX_SEATS + 1, 1]] as const) expect(centerPrice(seats, months)).toBeNull()
  })
  it('учитель без центра — от 10 мест', () => {
    expect(centerPrice(10, 1, 'teacher')).toBe(200_000)
    expect(centerPrice(9, 1, 'teacher')).toBeNull()
  })
})
