import { initData as telegramInitData } from './telegram'
import type { Lang } from './i18n/index.svelte'

export type Me = {
  nickname: string
  first_name: string
  photo_url: string
  lang: Lang
  goal_band: number | null
  exam_date: string | null // ГГГГ-ММ-ДД
  onboarded: boolean
}

export type MePatch = Partial<Pick<Me, 'lang' | 'goal_band' | 'exam_date' | 'onboarded'>>

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
  ) {
    super(code)
  }
}

let auth = ''

// Возвращает false, если приложение открыто не в Telegram и войти нечем.
export async function initAuth(): Promise<boolean> {
  auth = telegramInitData()
  // В обычном браузере при `npm run dev` сервер с DEV_AUTH=1 выдаёт тестовый вход.
  // В боевую сборку эта ветка не попадает: import.meta.env.DEV там false.
  if (!auth && import.meta.env.DEV) {
    const params = new URLSearchParams(location.search)
    const res = await fetch(`/api/dev/init-data?id=${params.get('uid') ?? ''}&lang=${params.get('lang') ?? ''}`)
    if (res.ok) auth = ((await res.json()) as { init_data: string }).init_data
  }
  return auth !== ''
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method,
    headers: {
      Authorization: `tma ${auth}`,
      ...(body === undefined ? {} : { 'Content-Type': 'application/json' }),
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  const data = (await res.json().catch(() => ({}))) as { error?: string }
  if (!res.ok) throw new ApiError(res.status, data.error ?? 'internal')
  return data as T
}

export const api = {
  me: () => request<Me>('GET', '/api/me'),
  updateMe: (patch: MePatch) => request<Me>('PATCH', '/api/me', patch),
}
