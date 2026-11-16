import { openTelegramLink } from './telegram'

export type Shared = 'shared' | 'copied' | 'failed'

// Поделиться результатом: в Telegram — выбор чата, в браузере — системное «Поделиться», иначе — копия
// в буфер. Ссылка пока на адрес приложения; с этапа 7 — на бота (t.me/…?startapp), DECISIONS.
export async function share(text: string): Promise<Shared> {
  const url = location.origin
  if (openTelegramLink(`https://t.me/share/url?url=${encodeURIComponent(url)}&text=${encodeURIComponent(text)}`)) return 'shared'
  if (navigator.share) {
    try {
      await navigator.share({ text, url })
      return 'shared'
    } catch {
      return 'failed'
    }
  }
  try {
    await navigator.clipboard.writeText(`${text} ${url}`)
    return 'copied'
  } catch {
    return 'failed'
  }
}
