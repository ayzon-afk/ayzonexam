import { onTelegramTheme, setChrome, telegramScheme } from './telegram'

// Оформление: auto — светлая или тёмная, как в Telegram; на сайте — светлая, как бот у большинства
// (решение владельца 2026-11-24); light и dark — всегда своя. Цвета обеих — фирменные (app.css). Выбор
// помнит устройство: это удобство, не данные.
export type Theme = 'auto' | 'light' | 'dark'

const KEY = 'ayzon.theme'

export function loadTheme(): Theme {
  try {
    const v = localStorage.getItem(KEY)
    return v === 'light' || v === 'dark' ? v : 'auto'
  } catch {
    return 'auto'
  }
}

export function applyTheme(theme: Theme) {
  const root = document.documentElement
  const resolved = theme === 'auto' ? (telegramScheme() ?? 'light') : theme
  if (resolved) root.dataset.theme = resolved
  else delete root.dataset.theme
  try {
    if (theme === 'auto') localStorage.removeItem(KEY)
    else localStorage.setItem(KEY, theme)
  } catch {
    // приватный режим: выбор забудется при закрытии — не беда
  }
  setChrome(getComputedStyle(root).getPropertyValue('--bg').trim())
}

// Тема до первого кадра и слежение за сменой темы в Telegram, пока выбрано «как в Telegram».
export function initTheme() {
  applyTheme(loadTheme())
  onTelegramTheme(() => {
    if (loadTheme() === 'auto') applyTheme('auto')
  })
}
