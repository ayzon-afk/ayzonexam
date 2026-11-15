// Запуск Go-команд в backend/ из npm-скриптов корня.
// Go ищется в PATH, иначе в ~/sdk/go (туда он поставлен на компьютере владельца).
// `fmt-check` падает, если какой-то файл не отформатирован gofmt.
import { spawnSync } from 'node:child_process'
import { existsSync } from 'node:fs'
import { homedir } from 'node:os'
import { join } from 'node:path'

const exe = process.platform === 'win32' ? '.exe' : ''
function find(name) {
  // ENOENT — программы нет в PATH; любой код выхода значит, что она есть.
  if (!spawnSync(name, ['help'], { stdio: 'ignore' }).error) return name
  const local = join(homedir(), 'sdk', 'go', 'bin', name + exe)
  return existsSync(local) ? local : name
}

const backend = join(import.meta.dirname, '..', 'backend')
const args = process.argv.slice(2)

if (args[0] === 'fmt-check') {
  const r = spawnSync(find('gofmt'), ['-l', '.'], { cwd: backend, encoding: 'utf8' })
  if (r.status !== 0 || r.error) {
    console.error(r.stderr || r.error)
    process.exit(1)
  }
  const files = r.stdout.trim()
  if (files) {
    console.error('Не отформатированы (gofmt -w .):\n' + files)
    process.exit(1)
  }
  console.log('gofmt: всё отформатировано')
  process.exit(0)
}

const r = spawnSync(find('go'), args, { cwd: backend, stdio: 'inherit' })
if (r.error) console.error(r.error.message)
process.exit(r.status ?? 1)
