import type { City } from './cities'
import { initData as telegramInitData } from './telegram'
import type { Lang } from './i18n/index.svelte'

export type Plan = {
  premium: boolean
  until: string | null
  source: '' | 'trial' | 'paid' | 'center'
  pending: boolean // оплата на проверке
  rejected: boolean // после отказа Premium — только после подтверждения
  discount_until: string | null
  mock_credits: number
  bonus: number
  prices: Record<'month' | 'month_discount' | 'quarter' | 'mock', number>
}

// kind: teacher — учитель без центра (тот же кабинет, ссылка и тариф).
export type CenterKind = 'center' | 'teacher'
export type MyCenter = { id: number; name: string; kind: CenterKind; status: 'pending' | 'member'; role: 'admin' | 'teacher' | null }

export type Me = {
  nickname: string
  first_name: string
  photo_url: string
  lang: Lang
  goal_band: number | null
  exam_date: string | null // ГГГГ-ММ-ДД
  onboarded: boolean
  city: '' | City
  center: MyCenter | null
  bot: string // @username бота: ссылки-приглашения
  admin: boolean
  verified_teacher: boolean
  plan: Plan | null
  invite: string
}

export type MePatch = Partial<Pick<Me, 'lang' | 'goal_band' | 'exam_date' | 'onboarded' | 'nickname'>> & { city?: City }

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

export type Mistake = { quote: string; fix: string; why: string; criterion: string }

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

export type Quota = { used: number; limit: number; period: 'day' | 'week'; bonus: number }
export type History = Quota & { checks: CheckSummary[] }

// Задания (этап 3).
export type Kind = 'reading' | 'listening' | 'level'
export type TestSummary = { id: number; kind: Kind; title: string; minutes: number; count: number; full_size: boolean; last_band: number | null }
// Виды вопросов (DECISIONS 2026-11-26): выбор, вписать, «выберите N» в группе; group — номер группы с 1.
export type QKind = 'choice' | 'text' | 'multi'
export type Question = { q: string; options: string[]; kind?: QKind; group?: number }
export type Group = { title: string; text?: string; options?: string[]; image?: string; pick?: number }
// Ответ: номер варианта (-1 — нет) или вписанный текст.
export type Answer = number | string
// Часть теста, как на компьютерном IELTS: Listening Part 1–4, Reading Passage 1–3; from и to — номера вопросов с 1.
export type Part = { title: string; intro?: string; passage?: string; from: number; to: number }
export type TestView = { id: number; kind: Kind; title: string; minutes: number; passage?: string; audio?: string; questions: Question[]; groups?: Group[]; parts?: Part[] }
export type Started = { attempt_id: number; ends_at: string; test: TestView }
export type Result = {
  id: number
  kind: Kind
  title: string
  correct: number
  total: number
  band: number
  ranked: boolean
  answers: Answer[]
  // По вопросу: номер варианта, верные буквы (multi) или допустимые ответы (text).
  key: (number | number[] | string[])[]
  marks: boolean[]
  transcript?: string
  finished_at: string
  test: TestView
  mock_id: number | null
}

// Рейтинг, центры, турнир (этап 8).
export type Scope = 'city' | 'center' | 'friends' | 'country'
export type RatingRow = { place: number; nickname: string; avatar: string; center: string; band: number; delta: number; me: boolean }
export type Board = { rows: RatingRow[]; mine: RatingRow | null; total: number }
export type CenterRow = { id: number; place: number; name: string; city: string; avg: number; scored: number; students: number; delta: number }
// Заявка «Добавить свой центр / Я учитель» (backend/internal/api/apply.go).
export type Application = { id: number; kind: CenterKind; name: string; city: string; students: number; status: 'pending' | 'approved' | 'rejected'; created_at: string }
export type ApplyBody = { kind: CenterKind; name: string; city: string; students: number; phone: string; about: string }
export type CenterInfo = { id: number; name: string; kind: CenterKind; city: string; about: string; students: number; avg: number | null; place: number | null; top: RatingRow[] | null }
export type Pending = { user_id: number; nickname: string; avatar: string; since: string }
export type TourState = {
  id: number
  starts_at: string
  ends_at: string
  players: number
  joined: boolean
  parts: Partial<Record<'reading' | 'listening', number>>
  band: number | null
  place: number | null
  finished: boolean
  live: boolean
  results: RatingRow[] | null
}
export type Winner = { month: string; nickname: string; avatar: string; center: string; center_id: number | null; band: number }
export type Stats = {
  current: number | null
  parts: number
  progress: number[]
  badges: Record<string, boolean>
  friends: number
  invited: number
  place: number | null
}

// Полный mock (этап 18).
export type PartState = {
  state: 'todo' | 'running' | 'checking' | 'done'
  band: number | null
  attempt_id?: number
  ends_at?: string
  violations: number // нарушения честного режима; с 3 часть не в рейтинге
}
export type MockPart = 'listening' | 'reading' | 'writing' | 'speaking'
export type MockCheck = { id: number; task: string; status: CheckStatus; band: number | null }
export type MockView = {
  id: number
  status: 'active' | 'done'
  started_at: string
  finished_at: string | null
  overall: number | null
  parts: Record<MockPart, PartState>
  writing_checks: MockCheck[]
  speaking_check: MockCheck | null
  weak: { part: MockPart; type: string; pct: number }[]
  plan: MockPart[]
  prev: Partial<Record<MockPart, number>>
  // L → R → W подряд, баллы частей — только в конце (DECISIONS 2026-11-26).
  strict: boolean
}
export type MockSummary = { id: number; status: string; overall: number | null; started_at: string; finished_at: string | null }
export type MockQuota = { premium: boolean; used: number; limit: number; credits: number }
export type MockPrompt = { id: number; text: string; image?: string }

// Друзья и дуэли (этап 9).
export type Friend = { id: number; nickname: string; avatar: string; band: number | null }
export type DuelPlayer = { nickname: string; avatar: string; score: number; answered: number; started_at: string | null; finished: boolean }
export type Duel = {
  id: number
  code: string
  state: 'open' | 'playing' | 'done' | 'expired'
  challenger: DuelPlayer
  opponent: DuelPlayer | null
  mine: boolean
  expires_at: string
  ends_at: string | null
}
export type DuelQ = { q: string; options: string[] }
export type Answered = { correct: boolean; answer: number; score: number; finished: boolean }

// Новости и поддержка (этапы 14, 17).
export type Post = { id: number; tag: 'tip' | 'result' | 'tournament' | 'news'; title: string; body: string; hidden: boolean; created_at: string }
export type TicketMessage = { id: number; from_admin: boolean; text: string; photo: number | null; created_at: string }
export type Ticket = { id: number; status: 'open' | 'answered' | 'closed'; created_at: string; updated_at: string; messages: TicketMessage[] | null }

// Speaking (этап 4).
export type SpeakingQ = { id: number; part: number; text: string }
export type SpeakCriterion = 'fc' | 'lr' | 'gra' | 'pr'
export type SpeakingCheck = {
  id: number
  mock_id: number | null
  source: 'app' | 'bot'
  questions: { part: number; text: string }[]
  seconds: number
  status: CheckStatus
  band: number | null
  criteria: Record<SpeakCriterion, number> | null
  transcript: string
  feedback: {
    summary: string
    criteria: Record<SpeakCriterion, string>
    mistakes: Mistake[]
    next_band: string[]
    better: string
  } | null
  created_at: string
}

export type HistoryItem = {
  kind: Kind | 'writing' | 'speaking' | 'mock'
  id: number
  title: string
  band: number | null
  status: 'done' | 'pending' | 'failed' | 'active'
  ranked: boolean
  tournament: boolean
  mock: boolean
  at: string
}

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
  ) {
    super(code)
  }
}

let auth = ''

// Вход на сайте (DECISIONS 2026-11-24): виджет «Войти через Telegram» → сервер выдаёт подписанный initData,
// он живёт в браузере сутки и ходит в заголовке, как у Mini App.
const WEB = 'ayzon.web'

export function webSession(): string {
  try {
    return localStorage.getItem(WEB) ?? ''
  } catch {
    return ''
  }
}

export function dropWebSession() {
  try {
    localStorage.removeItem(WEB)
  } catch {
    // хранилище недоступно — сессии и не было
  }
}

export async function webLogin(widget: Record<string, unknown>, lang: string): Promise<void> {
  const res = await fetch('/api/web/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ auth: widget, lang }),
  })
  if (!res.ok) throw new ApiError(res.status, 'login')
  const { init_data } = (await res.json()) as { init_data: string }
  try {
    localStorage.setItem(WEB, init_data)
  } catch {
    // без хранилища вход живёт до перезагрузки
  }
  auth = init_data
}

// Возвращает false, если приложение открыто не в Telegram и войти нечем.
export async function initAuth(): Promise<boolean> {
  auth = telegramInitData() || webSession()
  // В обычном браузере при `npm run dev` сервер с DEV_AUTH=1 выдаёт тестовый вход.
  // В боевую сборку эта ветка не попадает: import.meta.env.DEV там false.
  if (!auth && import.meta.env.DEV) {
    const params = new URLSearchParams(location.search)
    const res = await fetch(`/api/dev/init-data?id=${params.get('uid') ?? ''}&lang=${params.get('lang') ?? ''}`)
    if (res.ok) auth = ((await res.json()) as { init_data: string }).init_data
  }
  return auth !== ''
}

// Заголовок входа — для загрузок, которые идут мимо request (аудио в админке).
export function authHeader(): string {
  return `tma ${auth}`
}

// FormData и Blob уходят как есть (заголовок multipart ставит браузер), остальное — JSON.
export async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const raw = body instanceof FormData || body instanceof Blob
  const res = await fetch(path, {
    method,
    headers: {
      Authorization: authHeader(),
      ...(body === undefined || raw ? {} : { 'Content-Type': 'application/json' }),
    },
    body: body === undefined ? undefined : raw ? (body as BodyInit) : JSON.stringify(body),
  })
  const data = (await res.json().catch(() => ({}))) as { error?: string }
  if (!res.ok) throw new ApiError(res.status, data.error ?? 'internal')
  return data as T
}

const get = <T>(path: string) => request<T>('GET', path)
const post = <T>(path: string, body?: unknown) => request<T>('POST', path, body ?? {})

export const api = {
  me: () => get<Me>('/api/me'),
  updateMe: (patch: MePatch) => request<Me>('PATCH', '/api/me', patch),
  stats: () => get<Stats>('/api/me/stats'),
  history: () => get<{ items: HistoryItem[] }>('/api/history'),

  prompts: (task: Task) => get<{ prompts: Prompt[] }>(`/api/writing/prompts?task=${task}`),
  recognize: (pages: Blob[]) => {
    const f = new FormData()
    for (const p of pages) f.append('images', p, 'page.jpg')
    return post<{ text: string; words: number }>('/api/writing/recognize', f)
  },
  checks: () => get<History>('/api/writing/checks'),
  check: (id: number) => get<Check>(`/api/writing/checks/${id}`),
  createCheck: (c: { task: Task; source: 'text' | 'photo'; essay: string; promptId?: number; prompt?: string; image?: Blob }) => {
    const f = new FormData()
    f.append('task', c.task)
    f.append('source', c.source)
    f.append('essay', c.essay)
    if (c.promptId !== undefined) f.append('prompt_id', String(c.promptId))
    if (c.prompt) f.append('prompt', c.prompt)
    if (c.image) f.append('image', c.image, 'task.jpg')
    return post<{ id: number }>('/api/writing/checks', f)
  },

  tests: (kind: Kind) => get<{ tests: TestSummary[] }>(`/api/tests?kind=${kind}`),
  startTest: (id: number) => post<Started>(`/api/tests/${id}/start`),
  // В строгом mock ответ без балла и ключа: { id, mock_id, hidden: true }.
  submit: (attempt: number, answers: Answer[], leaves: number) => post<Result | { id: number; mock_id: number; hidden: true }>(`/api/attempts/${attempt}/submit`, { answers, leaves }),
  attempt: (id: number) => get<Result>(`/api/attempts/${id}`),

  rating: (scope: Scope) => get<Board>(`/api/rating?scope=${scope}`),
  centerRating: () => get<{ centers: CenterRow[]; mine: number | null }>('/api/rating/centers'),
  centers: () => get<{ centers: CenterInfo[] }>('/api/centers'),
  myApplication: () => get<{ application: Application | null }>('/api/apply'),
  apply: (body: ApplyBody) => post<{ application: Application }>('/api/apply', body),
  center: (id: number) => get<{ center: CenterInfo; status: 'pending' | 'member' | null }>(`/api/centers/${id}`),
  joinCenter: (code: string) => post<Me>('/api/centers/join', { code }),
  requestCenter: (id: number) => post<Me>(`/api/centers/${id}/request`),
  leaveCenter: () => request<Me>('DELETE', '/api/centers/me'),
  centerRequests: () => get<{ requests: Pending[]; invite_code: string }>('/api/center/requests'),
  decideRequest: (userId: number, accept: boolean) => post<void>(`/api/center/requests/${userId}`, { accept }),
  tournament: () => get<{ current: TourState; last: TourState | null; winners: Winner[] }>('/api/tournament'),
  joinTournament: () => post<void>('/api/tournament/join'),
  startTournament: (kind: 'reading' | 'listening') => post<Started>('/api/tournament/start', { kind }),

  mocks: () => get<{ active: MockView | null; past: MockSummary[]; quota: MockQuota }>('/api/mock'),
  newMock: () => post<MockView>('/api/mock'),
  mock: (id: number) => get<MockView>(`/api/mock/${id}`),
  mockPart: (id: number, kind: 'listening' | 'reading') => post<Started>(`/api/mock/${id}/part`, { kind }),
  mockWritingStart: (id: number) => post<{ ends_at: string; task1: MockPrompt | null; task2: MockPrompt }>(`/api/mock/${id}/writing/start`),
  mockWriting: (id: number, task1: string, task2: string, violations: number) =>
    post<MockView>(`/api/mock/${id}/writing`, { task1, task2, violations }),
  mockSpeakingStart: (id: number) => post<{ ends_at: string; questions: { part: number; text: string }[] }>(`/api/mock/${id}/speaking/start`),
  mockSpeaking: (id: number, audio: Blob[], seconds: number, violations: number) => {
    const f = new FormData()
    audio.forEach((a, i) => f.append(`audio${i}`, a, `part${i + 1}`))
    f.append('seconds', String(Math.round(seconds)))
    f.append('violations', String(violations))
    return post<MockView>(`/api/mock/${id}/speaking`, f)
  },

  refer: (code: string) => post<{ friend: string; referred: boolean }>('/api/referral', { code }),
  friends: () => get<{ friends: Friend[] }>('/api/friends'),
  unfriend: (id: number) => request<void>('DELETE', `/api/friends/${id}`),
  duels: () => get<{ duels: Duel[] }>('/api/duels'),
  newDuel: (friendId?: number) => post<Duel>('/api/duels', friendId ? { friend_id: friendId } : {}),
  duelByCode: (code: string) => get<Duel>(`/api/duels/code/${encodeURIComponent(code)}`),
  duel: (id: number) => get<Duel>(`/api/duels/${id}`),
  playDuel: (id: number) => post<{ duel: Duel; questions: DuelQ[]; answered: number }>(`/api/duels/${id}/play`),
  answerDuel: (id: number, i: number, choice: number) => post<Answered>(`/api/duels/${id}/answer`, { i, choice }),
  finishDuel: (id: number) => post<Duel>(`/api/duels/${id}/finish`),

  news: (limit = 20) => get<{ posts: Post[] }>(`/api/news?limit=${limit}`),
  post: (id: number) => get<Post>(`/api/news/${id}`),
  support: () => get<{ tickets: Ticket[] }>('/api/support'),
  writeSupport: (text: string, photo?: Blob) => {
    const f = new FormData()
    f.append('text', text)
    if (photo) f.append('photo', photo, 'photo.jpg')
    return post<{ id: number }>('/api/support', f)
  },

  speaking: () => get<{ questions: Record<string, SpeakingQ>; history: SpeakingCheck[]; quota: Quota }>('/api/speaking'),
  speak: (questionId: number, audio: Blob, seconds: number) => {
    const f = new FormData()
    f.append('question_id', String(questionId))
    f.append('seconds', String(Math.round(seconds)))
    f.append('audio', audio, 'answer')
    return post<{ id: number }>('/api/speaking', f)
  },
  speakingCheck: (id: number) => get<SpeakingCheck>(`/api/speaking/${id}`),
  speakViaBot: (questionId: number) => post<void>('/api/speaking/bot', { question_id: questionId }),

  payLink: () => get<{ path: string }>('/api/pay/link'),
}
