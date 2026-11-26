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
  isFullscreen?: boolean
  requestFullscreen?(): void
  exitFullscreen?(): void
  close(): void
  colorScheme?: 'light' | 'dark'
  onEvent?(event: string, cb: () => void): void
  offEvent?(event: string, cb: () => void): void
  openTelegramLink?(url: string): void
  openLink?(url: string): void
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
  // На компьютере — сразу во весь экран (DECISIONS 2026-11-25): кабинет и админка — во всю ширину, как
  // на сайте. Кто вышел из полного экрана, тому в следующий раз не навязываем.
  if (DESKTOP.includes(tg.platform) && canFullscreen() && remembered() !== '0') tg.requestFullscreen?.()
  tg.onEvent?.('fullscreenChanged', () => remember(tg.isFullscreen ? '1' : '0'))
}

// Платформы Telegram на компьютере: телефон открывает Mini App на весь экран и так.
const DESKTOP = ['tdesktop', 'macos', 'weba', 'webk', 'unigram']
const FULL_KEY = 'ayzon.fullscreen'

function remembered(): string | null {
  try {
    return localStorage.getItem(FULL_KEY)
  } catch {
    return null
  }
}

function remember(v: string) {
  try {
    localStorage.setItem(FULL_KEY, v)
  } catch {
    // хранилище недоступно — в следующий раз снова во весь экран
  }
}

// Весь экран (Bot API 8.0): шапка Telegram прячется, его кнопки ложатся поверх — отступ под них даёт
// --safe-top (app.css). false — старый клиент или не Telegram.
export function canFullscreen(): boolean {
  return inTelegram() && !!tg?.requestFullscreen && tg.isVersionAtLeast('8.0')
}

export function isFullscreen(): boolean {
  return !!tg?.isFullscreen
}

export function setFullscreen(on: boolean) {
  if (!canFullscreen()) return
  if (on) tg?.requestFullscreen?.()
  else tg?.exitFullscreen?.()
}

// onFullscreen — вызов при входе и выходе из полного экрана; возвращает отписку (для $effect).
export function onFullscreen(cb: () => void): () => void {
  tg?.onEvent?.('fullscreenChanged', cb)
  return () => tg?.offEvent?.('fullscreenChanged', cb)
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

// Внешняя страница (оплата на сайте) — в браузере телефона, приложение остаётся открытым.
export function openExternal(url: string) {
  if (inTelegram() && tg?.openLink) tg.openLink(url)
  else window.open(url, '_blank', 'noopener')
}

// Параметр запуска из ссылки t.me/<бот>/app?startapp=… (на будущее: прямые ссылки на Mini App).
export function startParam(): string {
  return (tg?.initDataUnsafe as { start_param?: string } | undefined)?.start_param ?? ''
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
