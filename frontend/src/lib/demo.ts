// Демо-данные этапа 12 (ROADMAP) для того, что ещё не подключено к серверу: лента и взаимная проверка
// (этап 19), группы, ДЗ и режим учителя (этап 10). Этап, который подключает экран, убирает его данные
// отсюда и пометку <Demo />. Ученики и учителя придуманы (DECISIONS 2026-11-16).
import type { Key } from './i18n/index.svelte'

// Взаимная проверка в демо: сколько эссе ученик проверил и сколько бонусов получил.
export const mine = { checked: 3, bonus: 1 }

export type Post = { id: number; nick: string; teacher?: boolean; text: Key; hours: number; likes: number; comments: number }

export const feed: Post[] = [
  { id: 1, nick: 'malika.ielts', text: 'demo.f1', hours: 2, likes: 24, comments: 5 },
  { id: 2, nick: 'teacher_dilshod', teacher: true, text: 'demo.f2', hours: 5, likes: 41, comments: 9 },
  { id: 3, nick: 'sardor07', text: 'demo.f3', hours: 9, likes: 12, comments: 2 },
  { id: 4, nick: 'jasur_b', text: 'demo.f4', hours: 20, likes: 8, comments: 1 },
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

// Эссе для взаимной проверки в демо.
export const peerEssay =
  'Nowadays many people believe that technology makes our life easier. In my opinion, this is true, but it also has some disadvantages. Firstly, phones and computers help us to communicate with friends who live far away. Secondly, we can find any information in internet very fast…'
