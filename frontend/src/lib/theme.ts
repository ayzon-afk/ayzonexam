import { setChrome } from './telegram'

// Оформление: auto — цвета Telegram (в браузере — тёмная или светлая, как в системе),
// light и dark — свои цвета поверх любых настроек. Выбор помнит устройство: это удобство, не данные.
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
  if (theme === 'auto') delete root.dataset.theme
  else root.dataset.theme = theme
  try {
    if (theme === 'auto') localStorage.removeItem(KEY)
    else localStorage.setItem(KEY, theme)
  } catch {
    // приватный режим: выбор забудется при закрытии — не беда
  }
  // Шапка и фон Telegram — под цвет приложения, иначе при своей теме сверху остаётся полоса чужого цвета.
  setChrome(theme === 'auto' ? null : getComputedStyle(root).getPropertyValue('--bg').trim())
}
