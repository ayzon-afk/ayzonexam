// Запросы админки (backend/internal/api/admin.go) и мелочи для её экранов.
import { authHeader, request } from '../../lib/api'

export type Row = Record<string, any>

export const adm = {
  get: <T = Row>(path: string) => request<T>('GET', `/api/admin/${path}`),
  post: <T = Row>(path: string, body: unknown) => request<T>('POST', `/api/admin/${path}`, body),
  put: <T = Row>(path: string, body: unknown) => request<T>('PUT', `/api/admin/${path}`, body),
  patch: <T = Row>(path: string, body: unknown) => request<T>('PATCH', `/api/admin/${path}`, body),
}

// Закрытый файл (скрин оплаты, фото обращения): картинка грузится с заголовком входа и показывается
// из памяти — адрес без входа не откроется.
export async function privateImage(id: number): Promise<string> {
  const res = await fetch(`/api/admin/media/${id}`, { headers: { Authorization: authHeader() } })
  if (!res.ok) throw new Error('media')
  return URL.createObjectURL(await res.blob())
}

export function when(iso: string | null | undefined): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleString('ru-RU', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit', timeZone: 'Asia/Tashkent' })
}

export function sum(n: number): string {
  return Number(n).toLocaleString('ru-RU').replace(/ /g, ' ')
}
