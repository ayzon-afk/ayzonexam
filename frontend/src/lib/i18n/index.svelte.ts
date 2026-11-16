import type { Key } from './uz'

export type Lang = 'uz' | 'ru' | 'en'
export type { Key }

export const LANGS: { code: Lang; name: string }[] = [
  { code: 'uz', name: 'O‘zbekcha' },
  { code: 'ru', name: 'Русский' },
  { code: 'en', name: 'English' },
]

// Словари грузятся по одному: телефон качает только язык ученика, а не все три (DECISIONS).
const loaders: Record<Lang, () => Promise<Record<Key, string>>> = {
  uz: () => import('./uz').then((m) => m.uz),
  ru: () => import('./ru').then((m) => m.ru),
  en: () => import('./en').then((m) => m.en),
}

let current = $state<Lang>('uz')
let dict = $state<Record<Key, string> | null>(null)

export function getLang(): Lang {
  return current
}

// setLang переключает язык, когда его словарь загружен; при быстрой смене побеждает последний вызов.
export async function setLang(lang: Lang): Promise<void> {
  current = lang
  document.documentElement.lang = lang
  const next = await loaders[lang]()
  if (current === lang) dict = next
}

// Язык Telegram до ответа сервера; по умолчанию узбекский — так же решает сервер (users.LangFromTelegram).
export function langFromTelegram(code: string | undefined): Lang {
  return code === 'ru' || code === 'en' ? code : 'uz'
}

export function format(template: string, params?: Record<string, string | number>): string {
  if (!params) return template
  return template.replace(/\{(\w+)\}/g, (m, k: string) => (k in params ? String(params[k]) : m))
}

// До загрузки словаря — пустая строка: App не показывает экраны, пока словаря нет.
export function t(key: Key, params?: Record<string, string | number>): string {
  return format(dict?.[key] ?? '', params)
}

export function ready(): boolean {
  return dict !== null
}
