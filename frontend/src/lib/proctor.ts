import { haptic } from './telegram'

// Честный mock (DECISIONS 2026-11-24): уход из приложения во время части mock — сигнал, вибрация и
// предупреждение; третий уход закрывает часть. Камера — позже.

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

// warn — три коротких сигнала и вибрация.
export function warn() {
  haptic.error()
  if (!ctx) return
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
