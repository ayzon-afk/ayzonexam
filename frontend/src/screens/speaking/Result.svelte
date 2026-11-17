<script lang="ts">
  import { onDestroy, onMount } from 'svelte'
  import Head from '../../components/Head.svelte'
  import Loader from '../../components/Loader.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { api, type SpeakCriterion, type SpeakingCheck } from '../../lib/api'
  import { formatBand } from '../../lib/exam'
  import { t, type Key } from '../../lib/i18n/index.svelte'
  import { replace } from '../../lib/nav.svelte'
  import { haptic } from '../../lib/telegram'

  // Разбор Speaking: балл по 4 критериям, ошибки, что AI услышал, как звучал бы ответ на балл выше.
  let { r }: { r: { id: number } } = $props()

  let c = $state<SpeakingCheck | null>(null)
  let failed = $state(false)
  let timer: ReturnType<typeof setTimeout> | undefined
  let stopped = false

  const crits: { key: SpeakCriterion; name: Key }[] = [
    { key: 'fc', name: 'crit.fc' },
    { key: 'lr', name: 'crit.lr' },
    { key: 'gra', name: 'crit.gra' },
    { key: 'pr', name: 'crit.pr' },
  ]
  const pending = $derived(c?.status === 'queued' || c?.status === 'processing')

  async function load() {
    try {
      const was = c?.status
      c = await api.speakingCheck(r.id)
      failed = false
      if (was && was !== c.status && c.status === 'done') haptic.success()
    } catch {
      failed = true
    }
    if (!stopped && (pending || failed)) timer = setTimeout(load, 4000)
  }

  onMount(load)
  onDestroy(() => {
    stopped = true
    clearTimeout(timer)
  })
</script>

<Head title="Speaking" demo={false} />

{#if !c}
  {#if failed}<p class="error">{t('app.error')}</p>{:else}<Skeleton rows={3} />{/if}
{:else if pending}
  <section class="card wait">
    <h2>{t('speaking.checking')}</h2>
    <Loader />
    <p class="hint">{t('result.checkingHint')}</p>
  </section>
{:else if c.status === 'failed'}
  <section class="card stack">
    <p>{t('speaking.failed')}</p>
    <button class="btn" onclick={() => replace({ s: 'speaking' })}>{t('speaking.again')}</button>
  </section>
{:else if c.criteria && c.feedback && c.band !== null}
  {@const fb = c.feedback}
  <section class="card score">
    <p class="hint">{t('result.band')}</p>
    <p class="band">{formatBand(c.band)}</p>
    <p>{fb.summary}</p>
  </section>

  <section class="card crits">
    {#each crits as k (k.key)}
      <div>
        <div class="head">
          <h2>{t(k.name)}</h2>
          <span class="pill">{c.criteria[k.key]}</span>
        </div>
        <p class="hint">{fb.criteria[k.key]}</p>
      </div>
    {/each}
  </section>

  {#if fb.mistakes.length}
    <h2 class="section">{t('result.mistakes')} · {fb.mistakes.length}</h2>
    <ul class="card list">
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
    <ul class="card list tips">
      {#each fb.next_band as tip, i (i)}<li>{tip}</li>{/each}
    </ul>
  {/if}

  {#if fb.better}
    <h2 class="section">{t('speaking.better')}</h2>
    <p class="card text">{fb.better}</p>
  {/if}

  <h2 class="section">{t('speaking.transcript')}</h2>
  <section class="card stack">
    {#each c.questions as q, i (i)}
      {#if q.text}<p class="hint">{q.part ? `Part ${q.part}: ` : ''}{q.text}</p>{/if}
    {/each}
    <p class="text">{c.transcript}</p>
  </section>
  <p class="hint">{t('result.disclaimer')}</p>
  {#if !c.mock_id}<button class="btn" onclick={() => replace({ s: 'speaking' })}>{t('speaking.again')}</button>{/if}
{/if}

<style>
  .wait {
    display: grid;
    justify-items: center;
    gap: 8px;
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
    color: var(--score);
  }

  .crits {
    display: grid;
    gap: 14px;
  }

  .head {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 8px;
  }

  h2 {
    font-size: 16px;
  }

  .pill {
    min-width: 32px;
    padding: 2px 10px;
    border-radius: 999px;
    background: color-mix(in srgb, var(--accent) 15%, transparent);
    color: var(--accent);
    font-weight: 700;
    text-align: center;
  }

  .section {
    margin-top: 8px;
  }

  .list {
    margin: 0;
    padding: 12px 16px 12px 34px;
    display: grid;
    gap: 10px;
  }

  .text {
    white-space: pre-wrap;
    line-height: 1.55;
  }
</style>
