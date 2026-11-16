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
  BackButton?: {
    show(): void
    hide(): void
    onClick(cb: () => void): void
    offClick(cb: () => void): void
  }
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

// Системная кнопка «Назад» в шапке Telegram. false — её нет (браузер, старый клиент), экран рисует свою.
export function hasBackButton(): boolean {
  return !!tg?.BackButton && tg.isVersionAtLeast('6.1')
}

// showBack показывает кнопку «Назад» и возвращает функцию, которая её убирает (для $effect).
export function showBack(cb: () => void): () => void {
  const b = tg?.BackButton
  if (!b || !hasBackButton()) return () => {}
  b.onClick(cb)
  b.show()
  return () => {
    b.offClick(cb)
    b.hide()
  }
}

export const haptic = {
  select: () => tg?.HapticFeedback?.selectionChanged(),
  success: () => tg?.HapticFeedback?.notificationOccurred('success'),
  error: () => tg?.HapticFeedback?.notificationOccurred('error'),
}
