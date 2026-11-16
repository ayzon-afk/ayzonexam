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

export function go(r: Route) {
  nav.stack.push(r)
  window.scrollTo(0, 0)
}

export function back() {
  nav.stack.pop()
  window.scrollTo(0, 0)
}

// replace меняет верхний экран: после «Отправить» — разбор, и «Назад» не возвращает на форму.
export function replace(r: Route) {
  nav.stack[nav.stack.length - 1] = r
  window.scrollTo(0, 0)
}

export function openTab(tab: Tab) {
  nav.tab = tab
  nav.stack = []
  window.scrollTo(0, 0)
}
