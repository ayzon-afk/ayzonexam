import { openTelegramLink } from './telegram'

export type Shared = 'shared' | 'copied' | 'failed'

// Ссылка-приглашение: через бота (t.me/<бот>?start=…) — бот откроет приложение на нужном экране
// (?open=…, bot.Greet). Бота нет (локально) — адрес приложения с тем же параметром.
export function inviteLink(bot: string, param: string): string {
  return bot ? `https://t.me/${bot}?start=${encodeURIComponent(param)}` : `${location.origin}/?open=${encodeURIComponent(param)}`
}

// Поделиться: в Telegram — выбор чата, в браузере — системное «Поделиться», иначе — копия в буфер.
export async function share(text: string, url = location.origin): Promise<Shared> {
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
