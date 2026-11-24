import { request } from './api'

// Пульс для учёта реального использования (DECISIONS 2026-11-24): раз в 30 с, только пока приложение на
// экране и ученик что-то делал за последние 5 минут (касание, клавиша, прокрутка) или звучит Listening.
// Сервер засчитывает не больше 30 с за пульс; первый пульс после открытия, фона или простоя — ничего.

const EVERY = 30_000
const IDLE = 5 * 60_000

const sid = Array.from(crypto.getRandomValues(new Uint8Array(12)), (b) => b.toString(16).padStart(2, '0')).join('')
let lastInput = Date.now()
let start = true
let timer: ReturnType<typeof setInterval> | undefined

function playing(): boolean {
  return Array.from(document.querySelectorAll('audio')).some((a) => !a.paused)
}

function active(): boolean {
  return document.visibilityState === 'visible' && (Date.now() - lastInput < IDLE || playing())
}

function send() {
  if (!active()) {
    start = true
    return
  }
  const first = start
  start = false
  request('POST', '/api/beat', { sid, start: first }).catch(() => {
    start = true // пульс потерян — следующий начнёт новый отрезок
  })
}

function input() {
  const wasIdle = !active()
  lastInput = Date.now()
  if (wasIdle) send()
}

export function startBeat() {
  if (timer) return
  for (const e of ['pointerdown', 'keydown', 'scroll', 'touchstart']) {
    addEventListener(e, input, { capture: true, passive: true })
  }
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'visible') input()
    else start = true
  })
  send()
  timer = setInterval(send, EVERY)
}
