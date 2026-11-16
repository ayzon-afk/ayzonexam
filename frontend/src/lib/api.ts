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

export type Task = 'task1' | 'task2'
export type CheckStatus = 'queued' | 'processing' | 'done' | 'failed'
export type Criterion = 'ta' | 'cc' | 'lr' | 'gra'

export type Prompt = { id: number; task: Task; text: string }

export type CheckSummary = {
  id: number
  task: Task
  status: CheckStatus
  band: number | null
  words: number
  prompt: string
  created_at: string
}

export type Mistake = { quote: string; fix: string; why: string; criterion: 'TA' | 'CC' | 'LR' | 'GRA' }

export type Check = CheckSummary & {
  essay: string
  source: 'text' | 'photo'
  criteria: Record<Criterion, number> | null
  feedback: {
    summary: string
    criteria: Record<Criterion, string>
    mistakes: Mistake[]
    corrected: string
    next_band: string[]
  } | null
  finished_at: string | null
}

export type History = { checks: CheckSummary[]; used_today: number; daily_limit: number }

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

// FormData уходит как multipart (заголовок с границей ставит браузер), остальное — JSON.
async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const form = body instanceof FormData
  const res = await fetch(path, {
    method,
    headers: {
      Authorization: `tma ${auth}`,
      ...(body === undefined || form ? {} : { 'Content-Type': 'application/json' }),
    },
    body: body === undefined ? undefined : form ? body : JSON.stringify(body),
  })
  const data = (await res.json().catch(() => ({}))) as { error?: string }
  if (!res.ok) throw new ApiError(res.status, data.error ?? 'internal')
  return data as T
}

export const api = {
  me: () => request<Me>('GET', '/api/me'),
  updateMe: (patch: MePatch) => request<Me>('PATCH', '/api/me', patch),
  prompts: (task: Task) => request<{ prompts: Prompt[] }>('GET', `/api/writing/prompts?task=${task}`),
  recognize: (pages: Blob[]) => {
    const f = new FormData()
    for (const p of pages) f.append('images', p, 'page.jpg')
    return request<{ text: string; words: number }>('POST', '/api/writing/recognize', f)
  },
  history: () => request<History>('GET', '/api/writing/checks'),
  check: (id: number) => request<Check>('GET', `/api/writing/checks/${id}`),
  createCheck: (c: { task: Task; source: 'text' | 'photo'; essay: string; promptId?: number; prompt?: string; image?: Blob }) => {
    const f = new FormData()
    f.append('task', c.task)
    f.append('source', c.source)
    f.append('essay', c.essay)
    if (c.promptId !== undefined) f.append('prompt_id', String(c.promptId))
    if (c.prompt) f.append('prompt', c.prompt)
    if (c.image) f.append('image', c.image, 'task.jpg')
    return request<{ id: number }>('POST', '/api/writing/checks', f)
  },
}
