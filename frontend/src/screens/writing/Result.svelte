<script lang="ts">
  import { onDestroy, onMount } from 'svelte'
  import BackBar from '../../components/BackBar.svelte'
  import Loader from '../../components/Loader.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { api, ApiError, type Check, type Criterion } from '../../lib/api'
  import { formatBand } from '../../lib/exam'
  import { getLang, t, type Key } from '../../lib/i18n/index.svelte'
  import { haptic } from '../../lib/telegram'
  import { saveDraft } from '../../lib/writing'

  let { id, onback, onagain }: { id: number; onback: () => void; onagain: () => void } = $props()

  let check = $state<Check | null>(null)
  let failed = $state<'' | 'error' | 'missing'>('')
  let timer: ReturnType<typeof setTimeout> | undefined
  let stopped = false
  const opened = Date.now()

  const pending = $derived(check?.status === 'queued' || check?.status === 'processing')
  const criteria = $derived<{ key: Criterion; name: Key }[]>([
    { key: 'ta', name: check?.task === 'task1' ? 'crit.ta.task1' : 'crit.ta.task2' },
    { key: 'cc', name: 'crit.cc' },
    { key: 'lr', name: 'crit.lr' },
    { key: 'gra', name: 'crit.gra' },
  ])

  // Пока проверка идёт — спрашиваем каждые 3 с, после двух минут — реже.
  async function load() {
    try {
      const was = check?.status
      check = await api.check(id)
      failed = ''
      if (was && was !== check.status && check.status === 'done') haptic.success()
    } catch (e) {
      failed = e instanceof ApiError && e.status === 404 ? 'missing' : 'error'
    }
    if (!stopped && (pending || failed === 'error')) {
      timer = setTimeout(load, Date.now() - opened < 120_000 ? 3000 : 10_000)
    }
  }

  // Повторная отправка: текст и тема уходят в черновик, ученик снова видит экран отправки.
  function again() {
    if (!check) return
    saveDraft({ task: check.task, promptId: null, prompt: check.task === 'task2' ? check.prompt : '', essay: check.essay, source: check.source })
    onagain()
  }

  function day(iso: string): string {
    return new Date(iso).toLocaleDateString(getLang(), { day: 'numeric', month: 'long' })
  }

  onMount(load)
  onDestroy(() => {
    stopped = true
    clearTimeout(timer)
  })
</script>

<BackBar {onback} />

{#if failed === 'missing'}
  <p class="error">{t('app.error')}</p>
{:else if !check}
  {#if failed}<p class="error">{t('app.error')}</p>{:else}<Skeleton rows={2} />{/if}
{:else}
  <header>
    <h1>{t(check.task === 'task1' ? 'task.task1' : 'task.task2')}</h1>
    <p class="hint">{day(check.created_at)} · {t('new.words', { n: check.words, min: check.task === 'task1' ? 150 : 250 })}</p>
  </header>

  {#if pending}
    <section class="card wait">
      <h2>{t('result.checking')}</h2>
      <Loader />
      <p class="hint">{t('result.checkingHint')}</p>
    </section>
  {:else if check.status === 'failed'}
    <section class="card stack">
      <p>{t('result.failed')}</p>
      <button class="btn" onclick={again}>{t('result.again')}</button>
    </section>
  {:else if check.criteria && check.feedback && check.band !== null}
    {@const fb = check.feedback}
    <section class="card score">
      <p class="hint">{t('result.band')}</p>
      <p class="band">{formatBand(check.band)}</p>
      <p>{fb.summary}</p>
    </section>

    <section class="card crits">
      {#each criteria as c (c.key)}
        <div class="crit">
          <div class="crit-head">
            <h2>{t(c.name)}</h2>
            <span class="pill">{check.criteria[c.key]}</span>
          </div>
          <p class="hint">{fb.criteria[c.key]}</p>
        </div>
      {/each}
    </section>

    {#if fb.mistakes.length}
      <h2 class="section">{t('result.mistakes')} · {fb.mistakes.length}</h2>
      <ul class="card mistakes">
        {#each fb.mistakes as m, i (i)}
          <li>
            <p><s>{m.quote}</s> → <b>{m.fix}</b></p>
            <p class="hint">{m.why}</p>
          </li>
        {/each}
      </ul>
    {/if}

    {#if fb.next_band.length}
      <h2 class="section">{t('result.nextBand')}</h2>
      <ul class="card tips">
        {#each fb.next_band as tip, i (i)}<li>{tip}</li>{/each}
      </ul>
    {/if}

    {#if fb.corrected}
      <details class="card">
        <summary>{t('result.corrected')}</summary>
        <p class="text">{fb.corrected}</p>
      </details>
    {/if}
  {/if}

  <details class="card">
    <summary>{t('result.yourText')}</summary>
    <p class="text">{check.essay}</p>
  </details>
  <details class="card">
    <summary>{t('result.task')}</summary>
    <p class="text">{check.prompt || t('result.chartTask')}</p>
    <!-- Разбор — в первом экране (Shell): график грузится, только когда он есть. -->
    {#if check.chart}
      {#await import('../../components/TaskChart.svelte') then m}<m.default chart={check.chart} />{/await}
    {/if}
  </details>

  {#if check.status === 'done'}<p class="hint center">{t('result.disclaimer')}</p>{/if}
{/if}

<style>
  h1 {
    font-size: 24px;
  }

  h2 {
    font-size: 16px;
    font-weight: 600;
  }

  header {
    display: grid;
    gap: 2px;
  }

  .stack {
    display: grid;
    gap: 12px;
  }

  .wait {
    display: grid;
    justify-items: center;
    gap: 8px;
    padding: 28px 16px;
    text-align: center;
  }

  .score {
    display: grid;
    gap: 4px;
    text-align: center;
  }

  .band {
    font-size: 48px;
    font-weight: 700;
    line-height: 1.1;
    font-variant-numeric: tabular-nums;
    color: var(--score);
  }

  .crits {
    display: grid;
    gap: 14px;
  }

  .crit {
    display: grid;
    gap: 4px;
  }

  .crit-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .pill {
    min-width: 36px;
    padding: 2px 10px;
    border-radius: 999px;
    background: var(--accent);
    color: var(--accent-text);
    font-weight: 700;
    text-align: center;
    font-variant-numeric: tabular-nums;
  }

  .section {
    margin-top: 4px;
  }

  ul {
    margin: 0;
  }

  .mistakes {
    list-style: none;
    padding: 0 16px;
  }

  .mistakes li {
    display: grid;
    gap: 4px;
    padding: 12px 0;
  }

  .mistakes li + li {
    border-top: 1px solid var(--separator);
  }

  s {
    color: var(--danger);
  }

  .tips {
    display: grid;
    gap: 8px;
    padding: 16px 16px 16px 34px;
  }

  summary {
    font-weight: 500;
    cursor: pointer;
  }

  .text {
    margin-top: 10px;
    white-space: pre-wrap;
    line-height: 1.5;
  }

  .center {
    text-align: center;
  }
</style>
