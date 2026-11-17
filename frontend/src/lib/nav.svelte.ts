import { flushSync } from 'svelte'
import type { Kind } from './api'

// Навигация без роутера: вкладка и стопка экранов поверх неё. «Назад» снимает верхний экран,
// нажатие на вкладку очищает стопку.

export type Tab = 'home' | 'exams' | 'rating' | 'community' | 'profile'
export type Part = Kind

// Откуда начинается задание: практика, часть mock или часть турнира.
export type TestSource = { test: number } | { mock: number; kind: 'listening' | 'reading' } | { tour: 'listening' | 'reading' }

export type Route =
  | { s: 'writingNew' }
  | { s: 'writingCheck'; id: number }
  | { s: 'news' }
  | { s: 'post'; id: number }
  | { s: 'tournament' }
  | { s: 'center'; id: number }
  | { s: 'centers' }
  | { s: 'centerRequests' }
  | { s: 'practice'; part: Part }
  | { s: 'test'; from: TestSource }
  | { s: 'attempt'; id: number }
  | { s: 'speaking' }
  | { s: 'speakingCheck'; id: number }
  | { s: 'mock' }
  | { s: 'mockWriting'; id: number }
  | { s: 'mockSpeaking'; id: number }
  | { s: 'report'; id: number }
  | { s: 'duel'; code?: string }
  | { s: 'friends' }
  | { s: 'teacher' }
  | { s: 'review' }
  | { s: 'group'; id: number }
  | { s: 'premium' }
  | { s: 'history' }
  | { s: 'support' }
  | { s: 'admin' }

export const nav = $state<{ tab: Tab; stack: Route[] }>({ tab: 'home', stack: [] })

export function current(): Route | undefined {
  return nav.stack[nav.stack.length - 1]
}

// Смена экрана с плавным переходом (app.css, «Плавные переходы»): браузер снимает старый кадр, Svelte
// рисует новый, анимация идёт между ними. Нет View Transitions или включено «уменьшить движение» — сразу.
type Dir = 'forward' | 'back' | 'tab'

function move(dir: Dir, change: () => void) {
  const done = () => {
    change()
    window.scrollTo(0, 0)
  }
  const doc = document as Document & { startViewTransition?: (cb: () => void) => unknown }
  if (!doc.startViewTransition || matchMedia('(prefers-reduced-motion: reduce)').matches) return done()
  document.documentElement.dataset.nav = dir
  doc.startViewTransition(() => flushSync(done))
}

export function go(r: Route) {
  move('forward', () => nav.stack.push(r))
}

export function back() {
  move('back', () => nav.stack.pop())
}

// replace меняет верхний экран: после «Отправить» — разбор, и «Назад» не возвращает на форму.
export function replace(r: Route) {
  move('forward', () => (nav.stack[nav.stack.length - 1] = r))
}

export function openTab(tab: Tab) {
  if (nav.tab === tab && nav.stack.length === 0) return
  move(nav.stack.length ? 'back' : 'tab', () => {
    nav.tab = tab
    nav.stack = []
  })
}

// Экран по параметру ?open=… из кнопки бота или ссылки-приглашения (bot.AppURL).
export function routeFromOpen(open: string): { tab: Tab; route?: Route } | null {
  const dash = open.indexOf('-')
  const kind = dash < 0 ? open : open.slice(0, dash)
  const rest = dash < 0 ? '' : open.slice(dash + 1)
  const id = Number(rest)
  const num = Number.isInteger(id) && id > 0
  switch (kind) {
    case 'premium':
      return { tab: 'profile', route: { s: 'premium' } }
    case 'friends':
      return { tab: 'rating', route: { s: 'friends' } }
    case 'support':
      return { tab: 'profile', route: { s: 'support' } }
    case 'center':
      return { tab: 'profile', route: { s: 'centerRequests' } }
    case 'mock':
      return num ? { tab: 'exams', route: { s: 'report', id } } : { tab: 'exams', route: { s: 'mock' } }
    case 'speaking':
      return num ? { tab: 'exams', route: { s: 'speakingCheck', id } } : { tab: 'exams', route: { s: 'speaking' } }
    case 'duel':
    case 'd':
      return rest ? { tab: 'rating', route: { s: 'duel', code: rest } } : { tab: 'rating', route: { s: 'duel' } }
  }
  return null
}
