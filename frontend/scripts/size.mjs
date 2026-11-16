// Бюджет веса (DECISIONS 2026-11-15 и 2026-11-16): то, что телефон качает при открытии приложения, —
// входной файл, его статические импорты и словарь одного языка (самый тяжёлый из трёх) — меньше
// 60 КБ в gzip. Экраны, которые грузятся при переходе, в первый экран не входят; их вес — для сведения.
import { readdirSync, readFileSync } from 'node:fs'
import { gzipSync } from 'node:zlib'
import { join } from 'node:path'

const BUDGET_JS = 60 * 1024
const dir = 'dist/assets'
const files = readdirSync(dir)
const gz = (f) => gzipSync(readFileSync(join(dir, f)), { level: 9 }).length

// Статические импорты Vite в сборке: `from"./X.js"` и `import"./X.js"`; динамические — `import("./X.js")`.
function staticClosure(entry) {
  const seen = new Set()
  const walk = (f) => {
    if (seen.has(f)) return
    seen.add(f)
    for (const m of readFileSync(join(dir, f), 'utf8').matchAll(/(?:from|import)\s*["']\.\/([\w.-]+\.js)["']/g)) walk(m[1])
  }
  walk(entry)
  return seen
}

const entry = files.find((f) => /^index-.*\.js$/.test(f))
const dicts = files.filter((f) => /^(uz|ru|en)-.*\.js$/.test(f))
if (!entry || dicts.length !== 3) {
  console.error('Не найден входной файл или три словаря в', dir)
  process.exit(1)
}

const first = [...staticClosure(entry)].reduce((n, f) => n + gz(f), 0) + Math.max(...dicts.map(gz))
const all = files.filter((f) => f.endsWith('.js')).reduce((n, f) => n + gz(f), 0)
const css = files.filter((f) => f.endsWith('.css')).reduce((n, f) => n + gz(f), 0)

const kb = (n) => (n / 1024).toFixed(1) + ' КБ'
console.log(`gzip: первый экран ${kb(first)} из ${kb(BUDGET_JS)}; весь JS ${kb(all)}, CSS ${kb(css)}`)
if (first > BUDGET_JS) {
  console.error('Первый экран превысил бюджет — найти, что прибавило вес, до коммита')
  process.exit(1)
}
