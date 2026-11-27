// Проверка вёрстки в настоящем браузере (DECISIONS 2026-11-25): каждый экран на ширинах от 320 до 1440 —
// ничего не вылезает за край экрана и за свою карточку. Без зависимостей: Chrome или Edge, который уже стоит
// на компьютере (CHROME=путь — свой), управляется по протоколу DevTools через встроенный в Node WebSocket.
// Данные — из scripts/layout/fixtures.mjs, сервер на Go не нужен. Нужна сборка: npm run build.
//
//   node scripts/layout.mjs                 — все экраны
//   node scripts/layout.mjs cabinet         — только те, в имени которых есть «cabinet»
//   node scripts/layout.mjs --shots ../shots — плюс снимки экрана каждого случая
import { spawn } from 'node:child_process'
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, statSync, writeFileSync } from 'node:fs'
import { createServer } from 'node:http'
import { tmpdir } from 'node:os'
import { extname, join, resolve } from 'node:path'
import { routes } from './layout/fixtures.mjs'

const WIDTHS = [320, 360, 390, 768, 1024, 1180, 1440]
const args = process.argv.slice(2)
const shotsAt = args.includes('--shots') ? resolve(args[args.indexOf('--shots') + 1]) : null
const only = args.filter((a, i) => !a.startsWith('--') && args[i - 1] !== '--shots')

// Случаи: кто вошёл, в Telegram или на сайте, адрес и шаги (клики по тексту или селектору).
// persona.lang: язык интерфейса — узбекские строки длиннее, их тоже проверяем.
const T = (persona, path, steps = []) => ({ persona, mode: 'tg', path, steps })
const W = (persona, path, steps = []) => ({ persona, mode: 'web', path, steps })
const tab = (i) => `nav button:nth-of-type(${i})`
const CASES = {
  'land: вход': W('', '/'),
  'land: центрам': W('', '/centers'),
  'land: админ-панель': W('', '/admin'),
  'land: кабинет': W('', '/cabinet'),
  'tg: главная': T('student', '/'),
  'tg: главная uz': T('student.uz', '/'),
  'tg: экзамены': T('student', '/', [tab(2)]),
  'tg: рейтинг': T('student', '/', [tab(3)]),
  'tg: рейтинг центров': T('student', '/', [tab(3), '.seg button:last-child']),
  'tg: рейтинг без региона': T('nocity', '/', [tab(3)]),
  'tg: рейтинг без региона uz': T('nocity.uz', '/', [tab(3)]),
  'tg: онбординг регион': T('newbie', '/', ['text:7.5', 'text:Дальше', 'text:Пока не знаю']),
  'tg: сообщество': T('student', '/', [tab(4)]),
  'tg: профиль': T('student', '/', [tab(5)]),
  'tg: профиль uz': T('student.uz', '/', [tab(5)]),
  'tg: история': T('student', '/', [tab(2), '.history']),
  'tg: практика': T('student', '/', [tab(2), '.part']),
  'tg: тест канала': T('student', '/', [tab(2), '.part', '.row']),
  'tg: тест канала uz': T('student.uz', '/', [tab(2), '.part', '.row']),
  'tg: тест канала разбор': T('student', '/', [tab(2), '.part', '.row', '.finish', '.veil .btn', 'text:Показать ответы']),
  'tg: тест канала часть 2': T('student', '/', [tab(2), '.part', '.row', '.plabel:nth-of-type(1)', 'text:Part 2']),
  'tg: тест подтверждение': T('student', '/', [tab(2), '.part', '.row', '.finish']),
  'tg: reading текст': T('student', '/', [tab(2), '.part', 'li:nth-child(2) .row']),
  'tg: reading вопросы': T('student', '/', [tab(2), '.part', 'li:nth-child(2) .row', 'text:Вопросы']),
  'tg: reading uz': T('student.uz', '/', [tab(2), '.part', 'li:nth-child(2) .row', 'text:Savollar']),
  'tg: reading passage 3': T('student', '/', [tab(2), '.part', 'li:nth-child(2) .row', 'text:Passage 3']),
  'tg: тест уровня вопрос': T('student', '/', [tab(2), '.level']),
  'tg: тест уровня вопрос uz': T('student.uz', '/', [tab(2), '.level']),
  'tg: тест уровня текст': T('student', '/', [tab(2), '.level', 'text:Не знаю']),
  'tg: тест уровня итог': T('student', '/', [tab(2), '.level', 'text:Не знаю', 'text:Узнать уровень', 'text:Показать ошибки']),
  'tg: тест уровня итог uz': T('student.uz', '/', [tab(2), '.level', 'text:Bilmayman', 'text:Darajani bilish', 'text:Xatolarni']),
  'tg: writing новое': T('student', '/?open=', [tab(2), 'text:Новая проверка|Продолжить']),
  'tg: writing разбор': T('student', '/?check=101'),
  'tg: турнир': T('student', '/', ['.tour']),
  'tg: новости': T('student', '/', ['text:Все']),
  'tg: центр': T('student', '/', [tab(5), 'text:Учебный центр|Мой центр']),
  'tg: центры': T('student', '/', [tab(4), 'text:Мой учебный центр']),
  'tg: premium': T('student', '/?open=premium'),
  'tg: друзья': T('student', '/?open=friends'),
  'tg: поддержка': T('student', '/?open=support'),
  'tg: mock': T('student', '/?open=mock'),
  'tg: mock отчёт': T('student', '/?open=mock-5'),
  'tg: speaking': T('student', '/?open=speaking'),
  'tg: speaking разбор': T('student', '/?open=speaking-301'),
  'tg: дуэль': T('student', '/?open=duel'),
  'tg: кабинет обзор': T('owner', '/?open=center', ['text:Обзор']),
  'tg: кабинет ученики': T('owner', '/?open=center', ['text:Ученики']),
  'tg: кабинет заявки': T('owner', '/?open=center'),
  'tg: кабинет тариф': T('owner', '/?open=center', ['text:Центр и тариф']),
  'tg: кабинет uz': T('owner.uz', '/?open=center', ['text:O‘quvchilar']),
  'tg: админка': T('admin', '/', [tab(5), 'text:Админка']),
  'web: приложение': W('student', '/'),
  'web: профиль': W('student', '/', [tab(5)]),
  'web: кабинет обзор': W('owner', '/cabinet#overview'),
  'web: кабинет ученики': W('owner', '/cabinet#students'),
  'web: кабинет ученики uz': W('owner.uz', '/cabinet#students'),
  'web: кабинет заявки': W('owner', '/cabinet#requests'),
  'web: кабинет тариф': W('owner', '/cabinet#settings'),
  'web: кабинет тариф uz': W('owner.uz', '/cabinet#settings'),
  'web: кабинет оплата на проверке': W('waiting', '/cabinet#settings'),
  'tg: кабинет из бота об оплате': T('owner', '/?open=cabinet'),
  'tg: учитель без центра — вход': T('solo', '/'),
  'tg: заявка центра без онбординга': T('newbie', '/?open=apply-center'),
  'tg: заявка учителя uz': T('student.uz', '/?open=apply-teacher'),
  'tg: заявка на проверке': T('applied', '/?open=apply-center'),
  'web: учитель без центра — тариф': W('solo', '/cabinet#settings'),
  'web: кабинет учителя': W('teacher', '/cabinet#students'),
  'web: кабинет без доступа': W('student', '/cabinet'),
  'web: админ обзор': W('admin', '/admin#overview'),
  'web: админ ученики': W('admin', '/admin#users'),
  'web: админ центры': W('admin', '/admin#centers'),
  'web: админ задания': W('admin', '/admin#tests'),
  'web: админ предпросмотр теста': W('admin', '/admin#tests', ['text:Volume 10', 'text:Предпросмотр']),
}

// ——— сервер: сборка из dist и ответы API ———
const dist = resolve('dist')
if (!existsSync(join(dist, 'index.html'))) {
  console.error('Нет сборки frontend/dist — сначала npm run build')
  process.exit(1)
}
const types = { '.js': 'text/javascript', '.css': 'text/css', '.html': 'text/html', '.svg': 'image/svg+xml', '.png': 'image/png', '.ico': 'image/x-icon', '.json': 'application/json' }

function route(method, path) {
  const exact = routes[`${method} ${path}`]
  if (exact !== undefined) return exact
  for (const [k, v] of Object.entries(routes)) {
    const [m, p] = k.split(' ')
    if (m === method && p.includes(':') && new RegExp('^' + p.replace(/:\w+/g, '[^/]+') + '$').test(path)) return v
  }
  return method === 'GET' ? undefined : {}
}

const server = createServer((req, res) => {
  const url = new URL(req.url, 'http://x')
  if (url.pathname.startsWith('/api/')) {
    const [persona, lang] = (req.headers.authorization ?? '').replace(/^tma /, '').split('.')
    let body = route(req.method, url.pathname)
    if (typeof body === 'function') body = body(persona, url)
    if (body === null) return res.writeHead(204).end()
    if (body === undefined || typeof body === 'number' || (!persona && url.pathname !== '/api/info')) {
      const code = typeof body === 'number' ? body : persona ? 404 : 401
      return res.writeHead(code, { 'Content-Type': 'application/json' }).end(`{"error":"${code}"}`)
    }
    if (url.pathname === '/api/me' && lang) body = { ...body, lang }
    return res.writeHead(200, { 'Content-Type': 'application/json' }).end(JSON.stringify(body))
  }
  let file = join(dist, url.pathname)
  if (!file.startsWith(dist) || !existsSync(file) || statSync(file).isDirectory()) file = join(dist, 'index.html')
  res.writeHead(200, { 'Content-Type': types[extname(file)] ?? 'application/octet-stream' }).end(readFileSync(file))
})
await new Promise((ok) => server.listen(0, '127.0.0.1', ok))
const origin = `http://127.0.0.1:${server.address().port}`

// ——— браузер ———
function findChrome() {
  const local = process.env.LOCALAPPDATA ?? ''
  const list = [
    process.env.CHROME,
    'C:/Program Files/Google/Chrome/Application/chrome.exe',
    'C:/Program Files (x86)/Google/Chrome/Application/chrome.exe',
    join(local, 'Google/Chrome/Application/chrome.exe'),
    'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe',
    'C:/Program Files/Microsoft/Edge/Application/msedge.exe',
    '/usr/bin/google-chrome',
    '/usr/bin/google-chrome-stable',
    '/usr/bin/chromium',
    '/usr/bin/chromium-browser',
    '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
  ]
  return list.find((p) => p && existsSync(p))
}

const chromePath = findChrome()
if (!chromePath) {
  console.error('Не найден Chrome или Edge. Укажите путь: CHROME=/путь/к/chrome node scripts/layout.mjs')
  server.close()
  process.exit(1)
}
const profile = mkdtempSync(join(tmpdir(), 'ayzon-layout-'))
const chrome = spawn(
  chromePath,
  [
    '--headless=new',
    '--remote-debugging-port=0',
    `--user-data-dir=${profile}`,
    '--no-first-run',
    '--no-default-browser-check',
    '--disable-extensions',
    '--disable-gpu',
    '--mute-audio',
    '--hide-scrollbars',
    ...(process.platform === 'linux' ? ['--no-sandbox'] : []),
    'about:blank',
  ],
  { stdio: ['ignore', 'ignore', 'pipe'] },
)
const wsURL = await new Promise((ok, fail) => {
  let buf = ''
  const timer = setTimeout(() => fail(new Error('Chrome не ответил за 20 с')), 20000)
  chrome.stderr.on('data', (d) => {
    buf += d
    const m = buf.match(/DevTools listening on (ws:\/\/\S+)/)
    if (m) {
      clearTimeout(timer)
      ok(m[1])
    }
  })
  chrome.on('exit', (code) => fail(new Error(`Chrome закрылся (${code}): ${buf.slice(0, 500)}`)))
})

// Протокол DevTools: команда — {id, method, params, sessionId}, ответ — с тем же id.
const ws = new WebSocket(wsURL)
await new Promise((ok, fail) => ((ws.onopen = ok), (ws.onerror = fail)))
let seq = 0
const waiting = new Map()
ws.onmessage = (e) => {
  const msg = JSON.parse(e.data)
  const w = msg.id && waiting.get(msg.id)
  if (!w) return
  waiting.delete(msg.id)
  if (msg.error) w.fail(new Error(`${w.method}: ${msg.error.message}`))
  else w.ok(msg.result)
}
const send = (method, params = {}, sessionId) =>
  new Promise((ok, fail) => {
    const id = ++seq
    waiting.set(id, { ok, fail, method })
    ws.send(JSON.stringify({ id, method, params, sessionId }))
  })

async function evaluate(s, expression) {
  const r = await send('Runtime.evaluate', { expression, awaitPromise: true, returnByValue: true }, s)
  if (r.exceptionDetails) throw new Error(r.exceptionDetails.exception?.description ?? r.exceptionDetails.text)
  return r.result.value
}

const sleep = (ms) => new Promise((ok) => setTimeout(ok, ms))

// Страница успокоилась: нет запросов в пути, заготовок (aria-busy) и анимаций перехода.
const SETTLED = `new Promise((ok) => {
  const t0 = performance.now()
  const tick = () => {
    const busy = window.__inflight > 0 || document.querySelector('[aria-busy="true"]') || document.getAnimations().some((a) => a.playState === 'running')
    if (!busy || performance.now() - t0 > 6000) requestAnimationFrame(() => requestAnimationFrame(() => ok(!busy)))
    else setTimeout(tick, 50)
  }
  tick()
})`

// Клик: CSS-селектор или «text:A|B» — первая видимая кнопка или ссылка с таким текстом.
const CLICK = (step) => `(() => {
  const step = ${JSON.stringify(step)}
  let el = null
  if (step.startsWith('text:')) {
    const words = step.slice(5).split('|')
    el = [...document.querySelectorAll('button, a, [role="tab"]')].find((b) => b.offsetParent !== null && words.some((w) => b.textContent.trim().startsWith(w) || b.textContent.includes(w)))
  } else el = document.querySelector(step)
  if (!el) return false
  el.click()
  return true
})()`

// Что вылезло: за край экрана (не внутри прокрутки) и текст, который не влез в свой блок.
const CHECK = `(() => {
  const W = document.documentElement.clientWidth
  const name = (el) => el.tagName.toLowerCase() + (el.id ? '#' + el.id : '') + [...el.classList].filter((c) => !c.startsWith('s-') && !c.startsWith('svelte-')).slice(0, 3).map((c) => '.' + c).join('')
  const path = (el) => { const p = []; for (let e = el; e && e !== document.body && p.length < 4; e = e.parentElement) p.unshift(name(e)); return p.join(' > ') }
  const clipped = (el) => { for (let p = el.parentElement; p && p !== document.documentElement; p = p.parentElement) { if (/(auto|scroll|hidden|clip)/.test(getComputedStyle(p).overflowX)) return true } return false }
  const out = [], spill = []
  for (const el of document.body.querySelectorAll('*')) {
    const st = getComputedStyle(el)
    if (st.display === 'none' || st.visibility === 'hidden' || st.position === 'fixed') continue
    const r = el.getBoundingClientRect()
    // Нулевой размер и спрятанное за левым краем нарочно (поле-ловушка для ботов) — не в счёт.
    if (r.width < 1 || r.height < 1 || r.right <= 0) continue
    if ((r.right > W + 1 || r.left < -1) && !clipped(el)) out.push(path(el) + ' [' + Math.round(r.left) + '…' + Math.round(r.right) + ' из ' + W + ']')
    if (/^(block|flex|grid|list-item)$/.test(st.display) && st.overflowX === 'visible' && el.clientWidth > 0 && el.scrollWidth > el.clientWidth + 2 && !clipped(el)) {
      const kid = [...el.children].some((c) => c.getBoundingClientRect().right > r.right + 2)
      if (!kid) spill.push(path(el) + ' [' + el.scrollWidth + ' > ' + el.clientWidth + ']')
    }
  }
  return { W, scroll: document.documentElement.scrollWidth, out: out.slice(0, 6), spill: spill.slice(0, 6), text: document.body.innerText.length }
})()`

const telegram = (persona) => `window.Telegram = { WebApp: {
  initData: ${JSON.stringify(persona)}, initDataUnsafe: { user: { language_code: 'ru' } }, version: '8.0', platform: 'android', colorScheme: 'light',
  ready() {}, expand() {}, close() {}, isVersionAtLeast() { return true }, disableVerticalSwipes() {}, onEvent() {}, offEvent() {},
  isFullscreen: false, requestFullscreen() {}, exitFullscreen() {},
  setHeaderColor() {}, setBackgroundColor() {}, setBottomBarColor() {}, openTelegramLink() {}, openLink() {},
  BackButton: { show() {}, hide() {}, onClick() {}, offClick() {} },
  HapticFeedback: { impactOccurred() {}, notificationOccurred() {}, selectionChanged() {} },
} }`

const PRELUDE = (c) => `(() => {
  let n = 0
  Object.defineProperty(window, '__inflight', { get: () => n })
  const f = window.fetch
  window.fetch = (...a) => { n++; return f(...a).finally(() => n--) }
  try {
    localStorage.clear()
    localStorage.setItem('ayzon.theme', 'light')
    ${c.mode === 'web' && c.persona ? `localStorage.setItem('ayzon.web', ${JSON.stringify(c.persona)})` : ''}
  } catch {}
  ${c.mode === 'tg' ? telegram(c.persona) : ''}
})()`

// ——— прогон ———
const { targetId } = await send('Target.createTarget', { url: 'about:blank' })
const { sessionId: s } = await send('Target.attachToTarget', { targetId, flatten: true })
await send('Page.enable', {}, s)
await send('Runtime.enable', {}, s)
await send('Network.enable', {}, s)
// Скрипты Telegram (Mini App и виджет входа) не грузим: Telegram подставляется выше, сеть не нужна.
await send('Network.setBlockedURLs', { urls: ['*telegram.org*', '*t.me*'] }, s)
if (shotsAt) mkdirSync(shotsAt, { recursive: true })

const picked = Object.entries(CASES).filter(([name]) => only.length === 0 || only.some((o) => name.includes(o)))
const problems = []
let prelude = null
const started = Date.now()

for (const [name, c] of picked) {
  const mobile = (w) => w < 900
  await send('Emulation.setDeviceMetricsOverride', { width: WIDTHS[0], height: 740, deviceScaleFactor: 1, mobile: true }, s)
  if (prelude) await send('Page.removeScriptToEvaluateOnNewDocument', { identifier: prelude }, s)
  prelude = (await send('Page.addScriptToEvaluateOnNewDocument', { source: PRELUDE(c) }, s)).identifier
  await send('Page.navigate', { url: 'about:blank' }, s)
  await send('Page.navigate', { url: origin + c.path }, s)
  await sleep(150)
  await evaluate(s, SETTLED)
  let broken = ''
  for (const step of c.steps) {
    if (!(await evaluate(s, CLICK(step)))) {
      broken = `шаг «${step}» не нашёл, куда нажать`
      break
    }
    await sleep(60)
    await evaluate(s, SETTLED)
  }
  if (broken) {
    problems.push(`${name}: ${broken}`)
    continue
  }
  for (const w of WIDTHS) {
    await send('Emulation.setDeviceMetricsOverride', { width: w, height: mobile(w) ? 740 : 900, deviceScaleFactor: 1, mobile: mobile(w) }, s)
    await sleep(40)
    await evaluate(s, SETTLED)
    const r = await evaluate(s, CHECK)
    const where = `${name} @${w}`
    if (r.text < 20) problems.push(`${where}: пустой экран`)
    if (r.scroll > r.W + 1) problems.push(`${where}: страница шире экрана (${r.scroll} > ${r.W})`)
    for (const o of r.out) problems.push(`${where}: за краем — ${o}`)
    for (const o of r.spill) problems.push(`${where}: не влезло — ${o}`)
    if (shotsAt) {
      const { cssContentSize } = await send('Page.getLayoutMetrics', {}, s)
      const height = Math.min(Math.ceil(cssContentSize.height), 6000)
      const { data } = await send('Page.captureScreenshot', { format: 'png', captureBeyondViewport: true, clip: { x: 0, y: 0, width: w, height, scale: 1 } }, s)
      writeFileSync(join(shotsAt, `${name.replace(/[^\p{L}\p{N}]+/gu, '-')}-${w}.png`), Buffer.from(data, 'base64'))
    }
  }
}

ws.close()
chrome.kill()
server.close()
try {
  rmSync(profile, { recursive: true, force: true })
} catch {
  // Chrome ещё держит файлы — временная папка уберётся сама
}

const secs = ((Date.now() - started) / 1000).toFixed(1)
if (problems.length) {
  console.error(`Вёрстка: ${problems.length} проблем (${picked.length} экранов × ${WIDTHS.length} ширин, ${secs} с):`)
  for (const p of problems) console.error('  ' + p)
  process.exit(1)
}
console.log(`Вёрстка: ${picked.length} экранов × ${WIDTHS.length} ширин (${WIDTHS.join(', ')}) — ничего не вылезает (${secs} с)`)
