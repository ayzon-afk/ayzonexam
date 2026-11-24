<script lang="ts">
  import type { Component } from 'svelte'
  import BackBar from '../../components/BackBar.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { back } from '../../lib/nav.svelte'

  // Админка владельца (этап 17, SPEC «Админка»). Грузится отдельно — ученики её не качают; вход проверяет
  // сервер по ADMIN_IDS. Тексты — по-русски, только для владельца и помощников (DECISIONS).
  type Section = { id: string; name: string; load: () => Promise<{ default: Component<any> }> }
  const sections: Section[] = [
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

  let current = $state('overview')
  const section = $derived(sections.find((s) => s.id === current)!)
</script>

<BackBar onback={back} />
<h1>Админка</h1>

<div class="seg" role="tablist">
  {#each sections as s (s.id)}
    <button role="tab" aria-selected={current === s.id} class:on={current === s.id} onclick={() => (current = s.id)}>{s.name}</button>
  {/each}
</div>

{#key current}
  {#await section.load()}
    <Skeleton rows={4} />
  {:then m}
    <m.default goto={(id: string) => (current = id)} />
  {:catch}
    <p class="error">Раздел не загрузился. Проверьте интернет.</p>
  {/await}
{/key}

<style>
  h1 {
    font-size: 24px;
  }
</style>
