<script lang="ts">
  import { onMount } from 'svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { adm, when, type Row } from './admin'

  // Заявки учебных центров с открытой страницы (DECISIONS 2026-11-24); каждая приходит и в бот.
  let leads = $state<Row[] | null>(null)
  let failed = $state(false)

  onMount(async () => {
    try {
      leads = (await adm.get<{ leads: Row[] }>('leads')).leads
    } catch {
      failed = true
    }
  })
</script>

{#if failed}
  <p class="error">Не загрузилось.</p>
{:else if !leads}
  <Skeleton rows={3} />
{:else if leads.length === 0}
  <p class="hint">Заявок пока нет. Форма — на открытой странице сайта (вне Telegram).</p>
{:else}
  {#each leads as l (l.id)}
    <div class="card lead">
      <p><b>{l.center}</b> · {l.students ?? '—'} учеников</p>
      <p>{l.name}, <a href="tel:{l.phone}">{l.phone}</a></p>
      {#if l.message}<p class="hint">{l.message}</p>{/if}
      <p class="hint">№{l.id} · {when(l.created_at)} · {l.lang}</p>
    </div>
  {/each}
{/if}

<style>
  .lead {
    display: grid;
    gap: 4px;
    margin-bottom: 10px;
  }
</style>
