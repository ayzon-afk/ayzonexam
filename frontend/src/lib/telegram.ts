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
  close(): void
  colorScheme?: 'light' | 'dark'
  onEvent?(event: string, cb: () => void): void
  openTelegramLink?(url: string): void
  setHeaderColor?(color: string): void
  setBackgroundColor?(color: string): void
  setBottomBarColor?(color: string): void
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

// Приложение открыто внутри Telegram: только там есть подписанный initData.
export function inTelegram(): boolean {
  return !!tg?.initData
}

// Закрыть Mini App — в Telegram «выйти» значит это: аккаунт и есть Telegram.
export function closeApp() {
  tg?.close()
}

// Шапка, фон и нижняя полоса Telegram — в цвет приложения: цвета свои, не Telegram.
export function setChrome(color: string) {
  if (!tg || !tg.isVersionAtLeast('6.9')) return
  tg.setHeaderColor?.(color)
  tg.setBackgroundColor?.(color)
  if (tg.isVersionAtLeast('7.10')) tg.setBottomBarColor?.(color)
}

// Светлая или тёмная тема Telegram; null — вне Telegram (тогда решает тема телефона через CSS).
export function telegramScheme(): 'light' | 'dark' | null {
  return inTelegram() ? (tg?.colorScheme ?? null) : null
}

export function onTelegramTheme(cb: () => void) {
  tg?.onEvent?.('themeChanged', cb)
}

// Ссылка t.me внутри Telegram — без выхода из приложения. false — Telegram нет.
export function openTelegramLink(url: string): boolean {
  if (!inTelegram() || !tg?.openTelegramLink) return false
  tg.openTelegramLink(url)
  return true
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
