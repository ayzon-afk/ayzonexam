<script lang="ts">
  import { onMount } from 'svelte'
  import Icon, { type IconName } from '../components/Icon.svelte'
  import Skeleton from '../components/Skeleton.svelte'
  import { api, type History } from '../lib/api'
  import { formatBand } from '../lib/exam'
  import { getLang, t, type Key } from '../lib/i18n/index.svelte'
  import { go, type Route } from '../lib/nav.svelte'
  import { loadDraft } from '../lib/writing'

  const onnew = () => go({ s: 'writingNew' })
  const onopen = (id: number) => go({ s: 'writingCheck', id })

  // Практика по частям: короткие сеты с таймером; Writing ниже — со своей историей.
  const parts: { name: string; hint: Key; icon: IconName; r: Route }[] = [
    { name: 'Reading', hint: 'quick.reading', icon: 'book', r: { s: 'practice', part: 'reading' } },
    { name: 'Listening', hint: 'quick.listening', icon: 'headphones', r: { s: 'practice', part: 'listening' } },
    { name: 'Speaking', hint: 'quick.speaking', icon: 'mic', r: { s: 'speaking' } },
  ]

  let history = $state<History | null>(null)
  let failed = $state(false)
  const hasDraft = loadDraft() !== null

  const left = $derived(history ? Math.max(history.limit - history.used, 0) : null)

  async function load() {
    failed = false
    try {
      history = await api.checks()
    } catch {
      failed = true
    }
  }

  function day(iso: string): string {
    return new Date(iso).toLocaleDateString(getLang(), { day: 'numeric', month: 'short' })
  }

  onMount(load)
</script>

<h1>{t('tab.exams')}</h1>

<button class="card level" onclick={() => go({ s: 'practice', part: 'level' })}>
  <span class="ico"><Icon name="target" /></span>
  <span class="grow">
    <strong>{t('level.title')}</strong>
    <span class="tag">{t('level.lead')}</span>
  </span>
  <span class="chev" aria-hidden="true">›</span>
</button>

<section class="card mock">
  <h2><span class="ico inline"><Icon name="timer" size={20} /></span> {t('mock.title')}</h2>
  <p class="hint">{t('mock.lead')}</p>
  <button class="btn" onclick={() => go({ s: 'mock' })}>{t('mock.open')}</button>
</section>

<div class="sect"><h2>{t('exams.parts')}</h2></div>
<div class="parts">
  {#each parts as p (p.name)}
    <button class="card part" onclick={() => go(p.r)}>
      <span class="ico"><Icon name={p.icon} /></span>
      <strong>{p.name}</strong>
      <span class="tag">{t(p.hint)}</span>
    </button>
  {/each}
</div>

<section class="card writing">
  <h2><span class="ico inline"><Icon name="pen" size={20} /></span> Writing</h2>
  <p class="hint">{t('writing.lead')}</p>
  {#if left !== null && history}
    <p class="left">
      {left > 0 ? t(history.period === 'day' ? 'quota.day' : 'quota.week', { n: left, limit: history.limit }) : t('writing.limitReached')}
      {#if history.bonus > 0}· {t('quota.bonus', { n: history.bonus })}{/if}
    </p>
    {#if left === 0 && history.period === 'week'}
      <button class="btn btn-plain" onclick={() => go({ s: 'premium' })}>{t('premium.go')}</button>
    {/if}
  {/if}
  <button class="btn" onclick={onnew}>{hasDraft ? t('writing.continue') : t('writing.new')}</button>
</section>

{#if failed}
  <div class="card">
    <p class="error">{t('app.error')}</p>
    <button class="btn btn-plain" onclick={load}>{t('app.retry')}</button>
  </div>
{:else if history}
  <h2 class="section">{t('writing.history')}</h2>
  {#if history.checks.length === 0}
    <p class="hint">{t('writing.empty')}</p>
  {:else}
    <ul class="card list">
      {#each history.checks as c (c.id)}
        <li>
          <button onclick={() => onopen(c.id)}>
            <span class="what">
              <span class="task">{t(c.task === 'task1' ? 'task.task1' : 'task.task2')} · {day(c.created_at)}</span>
              {#if c.prompt}<span class="hint prompt">{c.prompt}</span>{/if}
            </span>
            {#if c.status === 'done' && c.band !== null}
              <span class="band">{formatBand(c.band)}</span>
            {:else if c.status === 'failed'}
              <span class="status error">{t('writing.status.failed')}</span>
            {:else}
              <span class="status hint">{t('writing.status.pending')}</span>
            {/if}
          </button>
        </li>
      {/each}
    </ul>
  {/if}
{:else}
  <Skeleton rows={2} />
{/if}

<button class="card row history" onclick={() => go({ s: 'history' })}>
  <Icon name="history" />
  <span class="grow">{t('history.title')}</span>
  <span class="chev" aria-hidden="true">›</span>
</button>

<style>
  h1 {
    font-size: 26px;
  }

  h2 {
    font-size: 18px;
  }

  .writing {
    display: grid;
    gap: 10px;
  }

  .left {
    font-weight: 500;
  }

  .section {
    margin-top: 8px;
  }

  .list {
    list-style: none;
    margin: 0;
    padding: 0 16px;
  }

  li + li {
    border-top: 1px solid var(--separator);
  }

  li button {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    min-height: 60px;
    padding: 10px 0;
    text-align: left;
  }

  .what {
    flex: 1;
    min-width: 0;
    display: grid;
    gap: 2px;
  }

  .task {
    font-weight: 500;
  }

  .prompt {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .band {
    font-size: 22px;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
    color: var(--score);
  }

  .status {
    font-size: 13px;
    white-space: nowrap;
  }

  .ico {
    display: grid;
    place-items: center;
    width: 38px;
    height: 38px;
    flex: none;
    border-radius: 11px;
    background: color-mix(in srgb, var(--accent) 14%, transparent);
    color: var(--accent);
  }

  .ico.inline {
    display: inline-grid;
    width: 30px;
    height: 30px;
    border-radius: 9px;
    vertical-align: middle;
  }

  .part .ico {
    margin-bottom: 6px;
  }

  .level {
    display: flex;
    align-items: center;
    gap: 12px;
    text-align: left;
  }

  .level strong {
    display: block;
  }

  .mock {
    display: grid;
    gap: 10px;
  }

  .parts {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 10px;
  }

  .part {
    display: grid;
    gap: 2px;
    align-content: start;
    padding: 14px 12px;
    text-align: left;
  }

  .history {
    padding: 0 16px;
    margin-top: 4px;
  }
</style>
