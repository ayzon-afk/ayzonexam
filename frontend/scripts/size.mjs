// Бюджет веса (DECISIONS 2026-11-15): весь JS приложения — меньше 60 КБ в gzip.
import { readdirSync, readFileSync } from 'node:fs'
import { gzipSync } from 'node:zlib'
import { join } from 'node:path'

const BUDGET_JS = 60 * 1024
const dir = 'dist/assets'
let js = 0
let css = 0
for (const f of readdirSync(dir)) {
  const gz = gzipSync(readFileSync(join(dir, f)), { level: 9 }).length
  if (f.endsWith('.js')) js += gz
  if (f.endsWith('.css')) css += gz
}
const kb = (n) => (n / 1024).toFixed(1) + ' КБ'
console.log(`gzip: JS ${kb(js)} из ${kb(BUDGET_JS)}, CSS ${kb(css)}`)
if (js > BUDGET_JS) {
  console.error('JS превысил бюджет — найти, что прибавило вес, до коммита')
  process.exit(1)
}
