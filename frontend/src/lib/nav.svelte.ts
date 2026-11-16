import { flushSync } from 'svelte'

// Навигация без роутера: вкладка и стопка экранов поверх неё. «Назад» снимает верхний экран,
// нажатие на вкладку очищает стопку.

export type Tab = 'home' | 'exams' | 'rating' | 'community' | 'profile'
export type Part = 'reading' | 'listening' | 'level'

export type Route =
  | { s: 'writingNew' }
  | { s: 'writingCheck'; id: number }
  | { s: 'news' }
  | { s: 'post'; id: number }
  | { s: 'tournament' }
  | { s: 'center'; id: number }
  | { s: 'practice'; part: Part }
  | { s: 'speaking' }
  | { s: 'mock' }
  | { s: 'report' }
  | { s: 'duel' }
  | { s: 'teacher' }
  | { s: 'review' }
  | { s: 'group'; id: number }
  | { s: 'premium' }
  | { s: 'history' }

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
