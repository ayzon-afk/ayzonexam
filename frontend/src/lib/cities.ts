import type { Key } from './i18n/index.svelte'

// Регионы Узбекистана — как rating.Regions на сервере (меняется одно — меняется и другое);
// other — другая страна: рейтинга региона у такого ученика нет.
export const REGIONS = [
  'tashkent',
  'tashkent_region',
  'andijan',
  'bukhara',
  'fergana',
  'jizzakh',
  'kashkadarya',
  'khorezm',
  'namangan',
  'navoi',
  'samarkand',
  'surkhandarya',
  'syrdarya',
  'karakalpakstan',
] as const

export type Region = (typeof REGIONS)[number]
export type City = Region | 'other'

export const CITIES: City[] = [...REGIONS, 'other']

export const cityKey = (c: City): Key => `city.${c}`

export const isRegion = (c: string): c is Region => (REGIONS as readonly string[]).includes(c)
