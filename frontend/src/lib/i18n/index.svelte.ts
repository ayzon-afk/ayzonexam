import { uz, type Key } from './uz'
import { ru } from './ru'
import { en } from './en'

export type Lang = 'uz' | 'ru' | 'en'
export type { Key }

export const LANGS: { code: Lang; name: string }[] = [
  { code: 'uz', name: 'O‘zbekcha' },
  { code: 'ru', name: 'Русский' },
  { code: 'en', name: 'English' },
]

const dicts: Record<Lang, Record<Key, string>> = { uz, ru, en }

let current = $state<Lang>('uz')

export function getLang(): Lang {
  return current
}

export function setLang(lang: Lang) {
  current = lang
  document.documentElement.lang = lang
}

// Язык Telegram до ответа сервера; по умолчанию узбекский — так же решает сервер (users.LangFromTelegram).
export function langFromTelegram(code: string | undefined): Lang {
  return code === 'ru' || code === 'en' ? code : 'uz'
}

export function format(template: string, params?: Record<string, string | number>): string {
  if (!params) return template
  return template.replace(/\{(\w+)\}/g, (m, k: string) => (k in params ? String(params[k]) : m))
}

export function t(key: Key, params?: Record<string, string | number>): string {
  return format(dicts[current][key], params)
}
