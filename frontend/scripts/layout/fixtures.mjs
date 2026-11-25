// Ответы API для проверки вёрстки (scripts/layout.mjs). Формы — как в src/lib/api.ts и
// screens/cabinet/cabinet.ts. Ники и названия — длинные нарочно: на них и ломается вёрстка.
// Кто вошёл, видно по заголовку Authorization: «tma student», «tma owner», «tma teacher», «tma admin».

const now = Date.parse('2026-11-25T09:00:00Z')
const ago = (days) => new Date(now - days * 86400000).toISOString()
const inDays = (days) => new Date(now + days * 86400000).toISOString()

const CENTER = 'Cambridge Learning Centre — Chilonzor filiali'
const nicks = ['Shahzoda_Abdurakhmonova', 'Javohir', 'Muhammadali.Toshpulatov', 'Dilnoza', 'Bek', 'Sevinch_2009', 'Azizbek Karimov', 'Madina', 'Otabek_IELTS_8', 'Nilufar']

const plan = {
  premium: true,
  until: inDays(12),
  source: 'center',
  pending: false,
  rejected: false,
  discount_until: null,
  mock_credits: 1,
  bonus: 2,
  prices: { month: 49000, month_discount: 34000, quarter: 129000, mock: 25000 },
}

function me(persona) {
  const role = persona === 'owner' ? 'admin' : persona === 'teacher' ? 'teacher' : null
  return {
    nickname: persona === 'owner' ? 'Cambridge_Chilonzor_Admin' : nicks[0],
    first_name: 'Shahzoda',
    photo_url: '',
    lang: 'ru',
    goal_band: 7.5,
    exam_date: '2027-03-14',
    onboarded: true,
    city: 'tashkent',
    center: { id: 7, name: CENTER, status: 'member', role },
    bot: 'humostartup_bot',
    admin: persona === 'admin',
    verified_teacher: persona === 'teacher',
    plan,
    invite: 'k3m9xq2p7a',
  }
}

const row = (i, extra = {}) => ({ place: i + 1, nickname: nicks[i % nicks.length], avatar: '', center: CENTER, band: 8.5 - i * 0.5, delta: [2, 0, -1, 3, 0][i % 5], me: i === 3, ...extra })
const board = { rows: Array.from({ length: 12 }, (_, i) => row(i)), mine: row(3), total: 248 }

const tour = (live) => ({
  id: 11,
  starts_at: inDays(4),
  ends_at: inDays(4.125),
  players: 1324,
  joined: true,
  parts: { reading: 7 },
  band: 7,
  place: 41,
  finished: false,
  live,
  results: board.rows.slice(0, 5),
})

const checks = Array.from({ length: 4 }, (_, i) => ({
  id: 100 + i,
  task: i % 2 ? 'task1' : 'task2',
  status: i === 0 ? 'processing' : 'done',
  band: i === 0 ? null : 6 + i * 0.5,
  words: 287 + i,
  prompt: 'Some people believe that university education should be free for every student, while others think students should pay. Discuss both views and give your own opinion.',
  created_at: ago(i * 2),
}))

const mistake = { quote: 'In nowadays the technology are developing very fast', fix: 'Nowadays, technology is developing very fast', why: 'Subject–verb agreement and an incorrect preposition.', criterion: 'gra' }

const check = {
  ...checks[1],
  essay: 'In nowadays the technology are developing very fast. '.repeat(30),
  source: 'text',
  criteria: { ta: 6.5, cc: 6, lr: 6.5, gra: 6 },
  feedback: {
    summary: 'A clear position with relevant ideas, but paragraphs need stronger topic sentences and some grammar errors reduce clarity.',
    criteria: { ta: 'Addresses both views.', cc: 'Linking words are overused.', lr: 'Good range, some collocation errors.', gra: 'Frequent agreement errors.' },
    mistakes: [mistake, mistake, mistake],
    corrected: 'Nowadays, technology is developing very fast. '.repeat(20),
    next_band: ['Use one clear topic sentence per paragraph.', 'Check subject–verb agreement in every sentence.'],
  },
  finished_at: ago(2),
}

const part = (state, band) => ({ state, band })
const mock = {
  id: 5,
  status: 'done',
  started_at: ago(3),
  finished_at: ago(3),
  overall: 6.5,
  parts: { listening: part('done', 7), reading: part('done', 6.5), writing: part('done', 6), speaking: part('done', 6.5) },
  writing_checks: [{ id: 101, task: 'task2', status: 'done', band: 6 }],
  speaking_check: { id: 301, task: 'speaking', status: 'done', band: 6.5 },
  weak: [{ part: 'reading', type: 'tfng', pct: 48 }, { part: 'listening', type: 'map', pct: 55 }],
  plan: ['writing', 'reading', 'speaking'],
  prev: { listening: 6.5, reading: 6.5, writing: 5.5, speaking: 6 },
}

const speakingCheck = {
  id: 301,
  mock_id: null,
  source: 'app',
  questions: [{ part: 1, text: 'Do you work or are you a student?' }, { part: 2, text: 'Describe a place in your city that you like to visit. You should say where it is, when you go there and explain why you like it.' }],
  seconds: 184,
  status: 'done',
  band: 6.5,
  criteria: { fc: 6.5, lr: 6, gra: 6.5, pr: 7 },
  transcript: 'Well, I am a student at the moment and I study at a lyceum in Tashkent. '.repeat(8),
  feedback: { summary: 'Fluent with some hesitation.', criteria: { fc: 'Good flow.', lr: 'Some repetition.', gra: 'Mostly accurate.', pr: 'Clear.' }, mistakes: [mistake], next_band: ['Extend answers with examples.'], better: 'Honestly, I am currently a student…' },
  created_at: ago(1),
}

const post = (i) => ({ id: i, tag: ['tip', 'result', 'tournament', 'news'][i % 4], title: 'Турнир месяца: итоги ноября и как готовиться к декабрьскому этапу', body: 'Длинный текст новости. '.repeat(40), hidden: false, created_at: ago(i) })

const students = nicks.map((n, i) => ({
  id: 500 + i,
  nickname: i === 0 ? 'Cambridge_Chilonzor_Admin' : n,
  avatar: '',
  role: i === 0 ? 'admin' : i === 1 ? 'teacher' : null,
  band: i > 7 ? null : 8 - i * 0.5,
  parts: i > 7 ? {} : { listening: 8 - i * 0.5, reading: 7.5 - i * 0.5, writing: 6.5, ...(i % 2 ? {} : { speaking: 7 }) },
  month: 30 - i * 3,
  last_at: i === 9 ? null : ago(i * 3),
}))

const cabinet = (persona) => ({
  center: {
    id: 7,
    name: CENTER,
    city: 'tashkent',
    about: 'IELTS 7+ за 6 месяцев. Адрес: Чиланзар, 9-й квартал. Телефон +998 90 123 45 67.',
    invite_code: 'k3m9xq2p7a',
    seats: persona === 'owner' ? 120 : 0,
    paid_until: persona === 'owner' ? '2026-12-31' : null,
    active: true,
    students: 128,
    pending: 3,
    active_week: 96,
    avg: 6.72,
    scored: 87,
    place: 2,
    centers: 14,
    delta: 1,
  },
  role: persona === 'owner' ? 'admin' : 'teacher',
  ...(persona === 'owner' ? { payments: [{ amount: 1800000, seats: 120, paid_until: '2026-12-31', at: ago(25) }, { amount: 1650000, seats: 110, paid_until: '2026-11-30', at: ago(55) }] } : {}),
})

const centerRow = (i) => ({ id: i + 1, place: i + 1, name: i === 1 ? CENTER : `IELTS Academy ${['Yunusobod', 'Mirzo Ulug‘bek', 'Sergeli', 'Chilonzor'][i % 4]}`, city: 'tashkent', avg: 7.1 - i * 0.2, scored: 40 - i, students: 120 - i * 5, delta: i % 3 - 1 })

const adminRow = (i) => ({ id: i + 1, telegram_id: 100000000 + i, nickname: nicks[i % nicks.length], first_name: 'Abdurakhmonova Shahzoda Rustamovna', username: 'shahzoda_abdurakhmonova', lang: 'ru', created_at: ago(i), seen_at: ago(0), premium_until: inDays(3), blocked_at: null, center_role: null, verified_teacher: false, center_status: 'member', center: CENTER })

// routes: путь без ?query → ответ (функция от persona и адреса или значение). null — 204.
export const routes = {
  'GET /api/info': { bot: 'humostartup_bot' },
  'GET /api/me': (p) => me(p),
  'PATCH /api/me': (p) => me(p),
  'POST /api/beat': null,
  'GET /api/me/stats': { current: 6.5, parts: 4, progress: [5.5, 6, 6, 6.5, 6.5, 7, 6.5], badges: { first: true, streak7: true, mock: true }, friends: 6, invited: 2, place: 41 },
  'GET /api/history': { items: checks.map((c, i) => ({ kind: ['writing', 'reading', 'listening', 'mock'][i], id: c.id, title: c.prompt, band: c.band, status: c.status === 'done' ? 'done' : 'pending', ranked: i % 2 === 0, tournament: i === 2, mock: i === 3, at: c.created_at })) },
  'GET /api/writing/checks': { used: 2, limit: 3, period: 'day', bonus: 1, checks },
  'GET /api/writing/prompts': { prompts: [{ id: 1, task: 'task2', text: checks[0].prompt }, { id: 2, task: 'task2', text: 'Some people think that children should start school at a very early age. To what extent do you agree?' }] },
  'GET /api/writing/checks/:id': check,
  'GET /api/tests': (_, url) => ({ tests: Array.from({ length: 6 }, (_, i) => ({ id: i + 1, kind: url.searchParams.get('kind') ?? 'reading', title: `Cambridge IELTS 18 · Academic Test ${i + 1} · Passage ${i + 1}`, minutes: 20, count: 13, full_size: i === 0, last_band: i < 3 ? 6.5 + i * 0.5 : null })) }),
  'GET /api/rating': board,
  'GET /api/rating/centers': { centers: Array.from({ length: 8 }, (_, i) => centerRow(i)), mine: 2 },
  'GET /api/centers': { centers: Array.from({ length: 6 }, (_, i) => ({ ...centerRow(i), about: 'IELTS и General English', avg: null, place: null, top: null })) },
  'GET /api/centers/:id': { center: { id: 7, name: CENTER, city: 'tashkent', about: 'IELTS 7+ за 6 месяцев. '.repeat(4), students: 128, avg: 6.7, place: 2, top: board.rows.slice(0, 5) }, status: 'member' },
  'GET /api/center/requests': { requests: nicks.slice(4, 7).map((n, i) => ({ user_id: 900 + i, nickname: n, avatar: '', since: ago(i) })), invite_code: 'k3m9xq2p7a' },
  'GET /api/tournament': { current: tour(false), last: { ...tour(false), finished: true }, winners: [{ month: '2026-10-01', nickname: nicks[2], avatar: '', center: CENTER, center_id: 7, band: 8.5 }] },
  'GET /api/mock': { active: null, past: [{ id: 5, status: 'done', overall: 6.5, started_at: ago(3), finished_at: ago(3) }], quota: { premium: true, used: 1, limit: 2, credits: 1 } },
  'GET /api/mock/:id': mock,
  'GET /api/friends': { friends: nicks.slice(1, 7).map((n, i) => ({ id: i + 1, nickname: n, avatar: '', band: i % 3 ? 6.5 : null })) },
  'GET /api/duels': { duels: [{ id: 1, code: 'ab3k', state: 'done', challenger: { nickname: nicks[0], avatar: '', score: 8, answered: 10, started_at: ago(1), finished: true }, opponent: { nickname: nicks[2], avatar: '', score: 7, answered: 10, started_at: ago(1), finished: true }, mine: true, expires_at: inDays(1), ends_at: ago(1) }] },
  'GET /api/news': { posts: Array.from({ length: 5 }, (_, i) => post(i + 1)) },
  'GET /api/news/:id': post(1),
  'GET /api/support': { tickets: [{ id: 1, status: 'answered', created_at: ago(2), updated_at: ago(1), messages: [{ id: 1, from_admin: false, text: 'Не могу найти, где мой результат за mock. '.repeat(3), photo: null, created_at: ago(2) }, { id: 2, from_admin: true, text: 'Результат — во вкладке «Экзамены» → «История».', photo: null, created_at: ago(1) }] }] },
  'GET /api/speaking': { questions: { '1': { id: 1, part: 1, text: speakingCheck.questions[0].text }, '2': { id: 2, part: 2, text: speakingCheck.questions[1].text }, '3': { id: 3, part: 3, text: 'Why do some people prefer to live in big cities?' } }, history: [speakingCheck], quota: { used: 1, limit: 2, period: 'week', bonus: 0 } },
  'GET /api/speaking/:id': speakingCheck,
  'GET /api/pay/link': { path: '/pay?t=x' },
  'GET /api/cabinet': (p) => (p === 'owner' || p === 'teacher' ? cabinet(p) : 403),
  'GET /api/cabinet/students': (p) => (p === 'owner' || p === 'teacher' ? { students, me: 500 } : 403),
  'GET /api/admin/overview': { users: 12840, active_day: 1320, active_week: 5410, new_day: 210, new_week: 1350, paying: 412, money_month: 18450000, money_total: 96400000, centers: 14, writing_day: 640, writing_month: 15400, speaking_day: 230, speaking_month: 5100, ai_cost_month: 84.37, pending_payments: 5, open_tickets: 3 },
  'GET /api/admin/users': { users: Array.from({ length: 12 }, (_, i) => adminRow(i)) },
  'GET /api/admin/centers': { centers: Array.from({ length: 5 }, (_, i) => ({ ...centerRow(i), about: 'О центре', invite_code: 'k3m9xq2p7a', seats: 120, paid_until: '2026-12-31', active: true, created_at: ago(60), pending: 2, staff: [{ id: 1, nickname: 'Cambridge_Chilonzor_Admin', name: 'Rustam', role: 'admin' }] })) },
}
