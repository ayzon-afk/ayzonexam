import type { Component } from 'svelte'

// Разделы админки — общие для админки в Mini App (Admin.svelte) и админ-панели на сайте (AdminSite.svelte).
// Каждый грузится отдельно: ученики их не качают.
export type Section = { id: string; name: string; load: () => Promise<{ default: Component<any> }> }

export const sections: Section[] = [
  { id: 'overview', name: 'Обзор', load: () => import('./Overview.svelte') },
  { id: 'usage', name: 'Использование', load: () => import('./Usage.svelte') },
  { id: 'users', name: 'Ученики', load: () => import('./Users.svelte') },
  { id: 'leads', name: 'Заявки', load: () => import('./Leads.svelte') },
  { id: 'centers', name: 'Центры', load: () => import('./Centers.svelte') },
  { id: 'teachers', name: 'Учителя', load: () => import('./Teachers.svelte') },
  { id: 'payments', name: 'Оплаты', load: () => import('./Payments.svelte') },
  { id: 'tickets', name: 'Поддержка', load: () => import('./Tickets.svelte') },
  { id: 'news', name: 'Новости', load: () => import('./News.svelte') },
  { id: 'tours', name: 'Турнир', load: () => import('./Tours.svelte') },
  { id: 'tests', name: 'Задания', load: () => import('./Tests.svelte') },
  { id: 'log', name: 'Журнал', load: () => import('./Log.svelte') },
]
