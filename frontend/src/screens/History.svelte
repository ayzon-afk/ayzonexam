<script lang="ts">
  import { onMount } from 'svelte'
  import Head from '../components/Head.svelte'
  import Skeleton from '../components/Skeleton.svelte'
  import { api, type HistoryItem } from '../lib/api'
  import { fmtDate } from '../lib/date'
  import { formatBand } from '../lib/exam'
  import { t } from '../lib/i18n/index.svelte'
  import { go, type Route } from '../lib/nav.svelte'

  // Все попытки: задания, Writing, Speaking, mock. Нажатие открывает разбор.
  let items = $state<HistoryItem[] | null>(null)
  let failed = $state(false)

  function name(h: HistoryItem): string {
    switch (h.kind) {
      case 'mock':
        return t('mock.title')
      case 'level':
        return t('level.title')
      case 'writing':
        return 'Writing · ' + (h.title === 'task1' ? 'Task 1' : 'Task 2')
      case 'speaking':
        return 'Speaking' + (h.title === 'bot' ? ` · ${t('history.voice')}` : '')
    }
    return h.kind === 'reading' ? 'Reading' : 'Listening'
  }

  function route(h: HistoryItem): Route {
    switch (h.kind) {
      case 'mock':
        return h.status === 'done' ? { s: 'report', id: h.id } : { s: 'mock' }
      case 'writing':
        return { s: 'writingCheck', id: h.id }
      case 'speaking':
        return { s: 'speakingCheck', id: h.id }
    }
    return { s: 'attempt', id: h.id }
  }

  onMount(async () => {
    try {
      items = (await api.history()).items
    } catch {
      failed = true
    }
  })
</script>

<Head title={t('history.title')} demo={false} />

{#if failed}
  <p class="error">{t('app.error')}</p>
{:else if !items}
  <Skeleton rows={4} />
{:else if items.length === 0}
  <p class="card hint">{t('history.empty')}</p>
{:else}
  <ul class="card rows">
    {#each items as h (h.kind + h.id)}
      <li>
        <button class="row" onclick={() => go(route(h))}>
          <span class="grow">
            <strong>{name(h)}</strong>
            <span class="tag">
              {fmtDate(h.at)}{h.tournament ? ` · ${t('tour.title')}` : h.mock && h.kind !== 'mock' ? ` · ${t('mock.title')}` : ''}{h.ranked ? ` · ${t('history.ranked')}` : ''}
            </span>
          </span>
          {#if h.band !== null}
            <span class="num">{formatBand(h.band)}</span>
          {:else if h.status === 'failed'}
            <span class="tag error">{t('writing.status.failed')}</span>
          {:else}
            <span class="tag">{t('writing.status.pending')}</span>
          {/if}
        </button>
      </li>
    {/each}
  </ul>
{/if}

<style>
  strong {
    display: block;
    font-weight: 600;
  }

  .num {
    font-size: 20px;
  }
</style>
