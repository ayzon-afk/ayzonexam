// Кабинет центра (backend/internal/api/cabinet.go): запросы, разделы и расчёты для его экранов.
// Отдельный модуль, а не lib/api.ts: ученики его не качают.
import type { Component } from 'svelte'
import { request, type Me } from '../../lib/api'
import type { Key } from '../../lib/i18n/index.svelte'

export type Role = 'admin' | 'teacher'
export type PartKind = 'listening' | 'reading' | 'writing' | 'speaking'

export type CabCenter = {
  id: number
  name: string
  city: string
  about: string
  invite_code: string
  seats: number
  paid_until: string | null
  active: boolean
  students: number
  pending: number
  active_week: number
  avg: number | null
  scored: number
  place: number | null
  centers: number
  delta: number
}
export type CabPayment = { amount: number; seats: number; paid_until: string; at: string }
export type Cabinet = { center: CabCenter; role: Role; payments?: CabPayment[] }

export type Student = {
  id: number
  nickname: string
  avatar: string
  role: Role | null
  band: number | null
  parts: Partial<Record<PartKind, number>>
  month: number
  last_at: string | null
}

export const cab = {
  get: () => request<Cabinet>('GET', '/api/cabinet'),
  about: (about: string) => request<Cabinet>('PATCH', '/api/cabinet', { about }),
  newCode: () => request<Cabinet>('POST', '/api/cabinet/code', {}),
  students: () => request<{ students: Student[]; me: number }>('GET', '/api/cabinet/students'),
  remove: (id: number) => request<void>('DELETE', `/api/cabinet/students/${id}`),
  teacher: (id: number, teacher: boolean) => request<void>('PUT', `/api/cabinet/students/${id}/role`, { teacher }),
}

export const PARTS: { kind: PartKind; short: string; name: string }[] = [
  { kind: 'listening', short: 'L', name: 'Listening' },
  { kind: 'reading', short: 'R', name: 'Reading' },
  { kind: 'writing', short: 'W', name: 'Writing' },
  { kind: 'speaking', short: 'S', name: 'Speaking' },
]

// Разделы: учитель не видит «Центр и тариф» (деньги и настройки — у аккаунта центра).
export type SectionProps = { me: Me; data: Cabinet; onchange: (c: Cabinet) => void; goto: (id: string) => void }
export type Section = { id: string; name: Key; admin?: boolean; load: () => Promise<{ default: Component<SectionProps> }> }

export const sections: Section[] = [
  { id: 'overview', name: 'cab.sOverview', load: () => import('./Overview.svelte') },
  { id: 'students', name: 'cab.sStudents', load: () => import('./Students.svelte') },
  { id: 'requests', name: 'cab.sRequests', load: () => import('./Requests.svelte') },
  { id: 'settings', name: 'cab.sSettings', admin: true, load: () => import('./Settings.svelte') },
]

export const sectionsFor = (role: Role) => sections.filter((s) => !s.admin || role === 'admin')

export const IDLE_DAYS = 14
const DAY = 24 * 60 * 60 * 1000

export type Sort = 'band' | 'active' | 'name'
export type Filter = 'all' | 'noBand' | 'idle'

// idle — не занимался IDLE_DAYS дней (или ни разу).
export function idle(s: Student, now = Date.now()): boolean {
  return !s.last_at || now - new Date(s.last_at).getTime() > IDLE_DAYS * DAY
}

// view — список учеников после поиска, отбора и сортировки; исходный список не меняется.
export function view(list: Student[], q: string, filter: Filter, sort: Sort, now = Date.now()): Student[] {
  const needle = q.trim().toLowerCase()
  const out = list.filter(
    (s) =>
      (!needle || s.nickname.toLowerCase().includes(needle)) &&
      (filter === 'all' || (filter === 'noBand' ? s.band === null : idle(s, now))),
  )
  const time = (s: Student) => (s.last_at ? new Date(s.last_at).getTime() : 0)
  const byName = (a: Student, b: Student) => a.nickname.localeCompare(b.nickname)
  const cmp: Record<Sort, (a: Student, b: Student) => number> = {
    band: (a, b) => (b.band ?? -1) - (a.band ?? -1) || time(b) - time(a) || byName(a, b),
    active: (a, b) => time(b) - time(a) || b.month - a.month || byName(a, b),
    name: byName,
  }
  return out.sort(cmp[sort])
}

// CSV для Excel: «;» и BOM — русская и узбекская Windows открывают его без мастера импорта.
export function toCSV(list: Student[], head: string[]): string {
  const cell = (v: unknown) => {
    const s = v === null || v === undefined ? '' : String(v)
    return /[";\n\r]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s
  }
  const rows = list.map((s) => [s.nickname, s.band, ...PARTS.map((p) => s.parts[p.kind]), s.month, s.last_at?.slice(0, 10)])
  return '﻿' + [head, ...rows].map((r) => r.map(cell).join(';')).join('\r\n') + '\r\n'
}

export function download(name: string, text: string) {
  const url = URL.createObjectURL(new Blob([text], { type: 'text/csv;charset=utf-8' }))
  const a = Object.assign(document.createElement('a'), { href: url, download: name })
  a.click()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

// Ссылка на кабинет на сайте: тот же адрес, что у приложения, + /cabinet.
export function siteURL(): string {
  return `${location.origin}/cabinet`
}

// 1 234 567 — пробелами, как пишут суммы в Узбекистане.
export function sum(n: number): string {
  return String(Math.round(n)).replace(/\B(?=(\d{3})+(?!\d))/g, ' ')
}
