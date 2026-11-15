import { describe, expect, it } from 'vitest'
import { uz } from './uz'
import { ru } from './ru'
import { en } from './en'
import { format, langFromTelegram } from './index.svelte'

const placeholders = (s: string) => [...s.matchAll(/\{(\w+)\}/g)].map((m) => m[1]).sort()

describe('словари', () => {
  for (const [name, dict] of Object.entries({ ru, en })) {
    it(`${name}: те же ключи, что в uz`, () => {
      expect(Object.keys(dict).sort()).toEqual(Object.keys(uz).sort())
    })
    it(`${name}: те же подстановки {…}, что в uz`, () => {
      for (const [k, v] of Object.entries(uz)) {
        expect(placeholders(dict[k as keyof typeof uz]), k).toEqual(placeholders(v))
      }
    })
    it(`${name}: нет пустых строк`, () => {
      for (const [k, v] of Object.entries(dict)) expect(v.trim(), k).not.toBe('')
    })
  }
})

describe('format', () => {
  it('подставляет параметры и не трогает неизвестные', () => {
    expect(format('{a} и {b}', { a: 1 })).toBe('1 и {b}')
  })
})

describe('langFromTelegram', () => {
  it('ru и en — как есть, остальное — узбекский', () => {
    expect(langFromTelegram('ru')).toBe('ru')
    expect(langFromTelegram('en')).toBe('en')
    expect(langFromTelegram('uz')).toBe('uz')
    expect(langFromTelegram('de')).toBe('uz')
    expect(langFromTelegram(undefined)).toBe('uz')
  })
})
