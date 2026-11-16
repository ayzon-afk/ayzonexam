// Демо-данные этапа 12 (ROADMAP): весь интерфейс виден до того, как его подключат к серверу.
// Каждый следующий этап заменяет свою часть файла настоящими данными. Центры и ученики придуманы
// (DECISIONS 2026-11-16): настоящие названия — только с согласия центра.
// Задания — в lib/demo-exam.ts; здесь только то, что видно на вкладках.
import type { Key } from './i18n/index.svelte'

export type Center = { id: number; name: string; avg: number; students: number }

export const centers: Center[] = [
  { id: 1, name: 'Nova Academy', avg: 6.9, students: 412 },
  { id: 2, name: 'Atlas English', avg: 6.7, students: 356 },
  { id: 3, name: 'Zenit Study', avg: 6.6, students: 298 },
  { id: 4, name: 'Orion School', avg: 6.4, students: 241 },
  { id: 5, name: 'Lingua Park', avg: 6.2, students: 187 },
]

export const MY_CENTER = 2

export function centerById(id: number): Center {
  return centers.find((c) => c.id === id) ?? centers[0]!
}

export type Player = { nick: string; band: number; center: number }

export const players: Player[] = [
  { nick: 'aziz_8', band: 8.5, center: 1 },
  { nick: 'malika.ielts', band: 8.0, center: 2 },
  { nick: 'sardor07', band: 8.0, center: 3 },
  { nick: 'nilufar_s', band: 7.5, center: 1 },
  { nick: 'jasur_b', band: 7.5, center: 2 },
  { nick: 'dilnoza.t', band: 7.5, center: 4 },
  { nick: 'timur_uz', band: 7.0, center: 2 },
  { nick: 'kamila_7', band: 7.0, center: 5 },
  { nick: 'bekzod_m', band: 7.0, center: 3 },
  { nick: 'shahzoda', band: 6.5, center: 2 },
]

export const friends = ['malika.ielts', 'jasur_b', 'kamila_7']

export type Scope = 'city' | 'center' | 'friends' | 'country'

// Текущий ученик в демо: его балл по mock, место и число участников в каждом рейтинге.
export const mine = {
  band: 6.0,
  place: { city: 148, center: 12, friends: 4, country: 612 } as Record<Scope, number>,
  total: { city: 3214, center: 356, friends: 4, country: 9870 } as Record<Scope, number>,
  progress: [5.0, 5.5, 5.5, 6.0, 5.5, 6.0, 6.5, 6.0],
  checked: 3,
  bonus: 1,
  invited: 2,
}

// Победители турниров, свежий — первый: он стоит на главной весь месяц.
export const winners: { month: string; nick: string; band: number; center: number }[] = [
  { month: '2026-10-15', nick: 'aziz_8', band: 8.5, center: 1 },
  { month: '2026-09-15', nick: 'malika.ielts', band: 8.0, center: 2 },
  { month: '2026-08-15', nick: 'sardor07', band: 8.0, center: 3 },
]

export const TOURNAMENT_PLAYERS = 1248

// Турнир — последнее воскресенье месяца, 10:00 по Ташкенту (05:00 UTC).
export function nextTournament(now: Date): Date {
  let start = 0
  for (let add = 0; add < 2; add++) {
    const last = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth() + add + 1, 0))
    last.setUTCDate(last.getUTCDate() - last.getUTCDay())
    start = Date.UTC(last.getUTCFullYear(), last.getUTCMonth(), last.getUTCDate(), 5)
    if (start > now.getTime()) break
  }
  return new Date(start)
}

export type News = { id: number; tag: Key; title: Key; body: Key; date: string }

export const news: News[] = [
  { id: 1, tag: 'demo.tag.result', title: 'demo.n1.title', body: 'demo.n1.body', date: '2026-11-12' },
  { id: 2, tag: 'demo.tag.tip', title: 'demo.n2.title', body: 'demo.n2.body', date: '2026-11-10' },
  { id: 3, tag: 'demo.tag.tip', title: 'demo.n3.title', body: 'demo.n3.body', date: '2026-11-07' },
  { id: 4, tag: 'demo.tag.tournament', title: 'demo.n4.title', body: 'demo.n4.body', date: '2026-10-26' },
]

export type Post = { id: number; nick: string; teacher?: boolean; text: Key; hours: number; likes: number; comments: number }

export const feed: Post[] = [
  { id: 1, nick: 'malika.ielts', text: 'demo.f1', hours: 2, likes: 24, comments: 5 },
  { id: 2, nick: 'teacher_dilshod', teacher: true, text: 'demo.f2', hours: 5, likes: 41, comments: 9 },
  { id: 3, nick: 'sardor07', text: 'demo.f3', hours: 9, likes: 12, comments: 2 },
  { id: 4, nick: 'jasur_b', text: 'demo.f4', hours: 20, likes: 8, comments: 1 },
]

export type Badge = { title: Key; got: boolean }

export const badges: Badge[] = [
  { title: 'badge.first6', got: true },
  { title: 'badge.duel10', got: true },
  { title: 'badge.tour', got: true },
  { title: 'badge.first7', got: false },
  { title: 'badge.mock10', got: false },
  { title: 'badge.essay100', got: false },
]

export type Group = { id: number; name: string; teacher: string; students: number }

export const groups: Group[] = [
  { id: 1, name: 'IELTS 7+ · Evening', teacher: 'teacher_dilshod', students: 14 },
  { id: 2, name: 'Writing Intensive', teacher: 'ms_aziza', students: 9 },
]

export type Homework = { id: number; group: number; kind: 'writing' | 'speaking' | 'mock'; title: string; due: string; band?: number }

export const homework: Homework[] = [
  { id: 1, group: 1, kind: 'writing', title: 'Some people think university education should be free for everyone. Discuss both views.', due: '2026-11-19' },
  { id: 2, group: 1, kind: 'speaking', title: 'Describe a book you enjoyed reading.', due: '2026-11-18' },
  { id: 3, group: 1, kind: 'mock', title: 'Reading — Test 4', due: '2026-11-14', band: 6.5 },
  { id: 4, group: 2, kind: 'writing', title: 'The chart shows car ownership in three cities, 2000–2020.', due: '2026-11-20' },
]

export type Submission = { id: number; nick: string; task: 'task1' | 'task2'; ai: number; summary: Key }

export const submissions: Submission[] = [
  { id: 1, nick: 'jasur_b', task: 'task2', ai: 6.5, summary: 'demo.s1' },
  { id: 2, nick: 'shahzoda', task: 'task2', ai: 6.0, summary: 'demo.s2' },
  { id: 3, nick: 'timur_uz', task: 'task1', ai: 7.0, summary: 'demo.s3' },
]

export const groupStats = {
  crit: { ta: 6.4, cc: 6.1, lr: 6.0, gra: 5.8 },
  errors: ['demo.e1', 'demo.e2', 'demo.e3', 'demo.e4', 'demo.e5'] as Key[],
  hours: 31,
}

export type Attempt = { kind: 'mock' | 'tournament' | 'reading' | 'listening' | 'speaking' | 'writing'; band: number; date: string }

export const attempts: Attempt[] = [
  { kind: 'mock', band: 6.5, date: '2026-11-15' },
  { kind: 'reading', band: 7.0, date: '2026-11-13' },
  { kind: 'listening', band: 6.5, date: '2026-11-12' },
  { kind: 'speaking', band: 6.5, date: '2026-11-10' },
  { kind: 'writing', band: 6.0, date: '2026-11-08' },
  { kind: 'tournament', band: 6.0, date: '2026-10-25' },
]

export const report = {
  overall: 6.5,
  parts: [
    { name: 'Listening', band: 6.5 },
    { name: 'Reading', band: 7.0 },
    { name: 'Writing', band: 6.0 },
    { name: 'Speaking', band: 6.5 },
  ],
  weak: ['demo.w1', 'demo.w2', 'demo.w3'] as Key[],
  plan: ['demo.p1', 'demo.p2', 'demo.p3', 'demo.p4'] as Key[],
}
