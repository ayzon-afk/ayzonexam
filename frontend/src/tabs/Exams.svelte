<script lang="ts">
  import { onMount } from 'svelte'
  import { api, type History } from '../lib/api'
  import { formatBand } from '../lib/exam'
  import { getLang, t } from '../lib/i18n/index.svelte'
  import { loadDraft } from '../lib/writing'

  let { onnew, onopen }: { onnew: () => void; onopen: (id: number) => void } = $props()

  let history = $state<History | null>(null)
  let failed = $state(false)
  const hasDraft = loadDraft() !== null

  const left = $derived(history ? Math.max(history.daily_limit - history.used_today, 0) : null)

  async function load() {
    failed = false
    try {
      history = await api.history()
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

<section class="card writing">
  <h2>Writing</h2>
  <p class="hint">{t('writing.lead')}</p>
  {#if left !== null && history}
    <p class="left">
      {left > 0 ? t('writing.left', { n: left, limit: history.daily_limit }) : t('writing.limitReached')}
    </p>
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
{/if}

<p class="hint next">{t('exams.next')}</p>

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
    color: var(--accent);
  }

  .status {
    font-size: 13px;
    white-space: nowrap;
  }

  .next {
    text-align: center;
    margin-top: 8px;
  }
</style>
