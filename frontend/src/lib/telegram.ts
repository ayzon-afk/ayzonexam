// Тонкая обёртка над telegram-web-app.js: только то, чем приложение пользуется.
// Скрипт есть и вне Telegram, но тогда initData пустой.

type HapticStyle = 'light' | 'medium' | 'heavy' | 'rigid' | 'soft'

interface WebApp {
  initData: string
  initDataUnsafe: { user?: { language_code?: string } }
  version: string
  platform: string
  ready(): void
  expand(): void
  isVersionAtLeast(version: string): boolean
  disableVerticalSwipes?(): void
  HapticFeedback?: {
    impactOccurred(style: HapticStyle): void
    notificationOccurred(type: 'error' | 'success' | 'warning'): void
    selectionChanged(): void
  }
}

declare global {
  interface Window {
    Telegram?: { WebApp: WebApp }
  }
}

const tg: WebApp | undefined = typeof window !== 'undefined' ? window.Telegram?.WebApp : undefined

export function initTelegram() {
  if (!tg) return
  tg.ready()
  tg.expand()
  // Свайп вниз не закрывает приложение посреди экзамена.
  if (tg.isVersionAtLeast('7.7')) tg.disableVerticalSwipes?.()
}

export function initData(): string {
  return tg?.initData ?? ''
}

export function telegramLanguage(): string | undefined {
  return tg?.initDataUnsafe.user?.language_code
}

export const haptic = {
  select: () => tg?.HapticFeedback?.selectionChanged(),
  success: () => tg?.HapticFeedback?.notificationOccurred('success'),
  error: () => tg?.HapticFeedback?.notificationOccurred('error'),
}
