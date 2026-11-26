import { haptic } from './telegram'

// Честный mock (DECISIONS 2026-11-24): уход из приложения во время части mock — сигнал, вибрация и
// предупреждение; третий уход закрывает часть. Камера, полный экран, вставка — Proctor (DECISIONS 2026-11-25).

export const MAX_LEAVES = 3

let ctx: AudioContext | undefined

// arm готовит звук: телефоны разрешают его только после касания, поэтому контекст создаётся на первом
// касании экрана части, а сигнал звучит позже — при возврате в приложение.
export function arm() {
  const unlock = () => {
    try {
      ctx ??= new AudioContext()
      if (ctx.state === 'suspended') ctx.resume()
    } catch {
      // без звука — останутся вибрация и предупреждение
    }
  }
  addEventListener('pointerdown', unlock, { once: true, capture: true })
}

// warn — три коротких сигнала и вибрация; sound = false — только вибрация (Speaking: сигнал попал бы в запись).
export function warn(sound = true) {
  haptic.error()
  if (!ctx || !sound) return
  try {
    if (ctx.state === 'suspended') ctx.resume()
    const g = ctx.createGain()
    g.connect(ctx.destination)
    const t0 = ctx.currentTime
    for (let i = 0; i < 3; i++) {
      const o = ctx.createOscillator()
      o.type = 'square'
      o.frequency.value = 880
      o.connect(g)
      g.gain.setValueAtTime(0.12, t0 + i * 0.25)
      o.start(t0 + i * 0.25)
      o.stop(t0 + i * 0.25 + 0.15)
    }
  } catch {
    // звук недоступен — вибрации достаточно
  }
}

// Полный экран браузера на ноутбуке (DECISIONS 2026-11-25): часть mock идёт на весь экран, выход — нарушение.
// В Telegram Desktop браузерный полный экран недоступен (fullscreenEnabled = false) — там свой, Telegram.
export const canLock = () => !!document.fullscreenEnabled

export async function lockScreen(): Promise<boolean> {
  try {
    await document.documentElement.requestFullscreen({ navigationUI: 'hide' })
    return true
  } catch {
    return false // нужен клик: покажем кнопку
  }
}

export function unlockScreen() {
  if (document.fullscreenElement) document.exitFullscreen().catch(() => {})
}

// Второй монитор (Chrome и Edge знают о нём — screen.isExtended); другие браузеры — не проверяем.
export const secondScreen = () => (screen as Screen & { isExtended?: boolean }).isExtended === true
