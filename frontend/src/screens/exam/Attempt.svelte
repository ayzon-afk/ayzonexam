<script lang="ts">
  import { onMount } from 'svelte'
  import Head from '../../components/Head.svelte'
  import Questions from '../../components/Questions.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { api, type Result } from '../../lib/api'
  import { formatBand } from '../../lib/exam'
  import { t } from '../../lib/i18n/index.svelte'
  import { back, go } from '../../lib/nav.svelte'

  // Итог задания: балл, верные ответы, ключ; для Listening — сценарий записи.
  let { r }: { r: { id: number } } = $props()

  let res = $state<Result | null>(null)
  let failed = $state(false)
  let showKey = $state(false)

  onMount(async () => {
    try {
      res = await api.attempt(r.id)
    } catch {
      failed = true
    }
  })
</script>

{#if failed}
  <Head title={t('result.title')} demo={false} />
  <p class="error">{t('app.error')}</p>
{:else if !res}
  <Skeleton rows={3} />
{:else}
  <Head title={res.kind === 'level' ? t('level.title') : res.title} sub={res.kind === 'level' ? '' : res.kind === 'reading' ? 'Reading' : 'Listening'} demo={false} />
  {#if res.level}
    {@const lv = res.level}
    <!-- Тест уровня, который подстраивается: балл, диапазон «скорее всего», CEFR и части (DECISIONS 2026-11-27). -->
    <section class="card result">
      <p class="hint">{t('level.result')}</p>
      <p class="num band">{formatBand(res.band)}</p>
      <p>{t('level.range', { low: formatBand(lv.low), high: formatBand(lv.high) })} · <b>{lv.cefr}</b></p>
      <p class="hint">{t('level.measured')}</p>
    </section>
    <ul class="card parts">
      <li><span class="grow">{t('level.core')}</span><span class="num">{lv.core.correct}/{lv.core.total}</span></li>
      <li><span class="grow">Reading · {lv.passage}</span><span class="num">{lv.reading.correct}/{lv.reading.total}</span></li>
    </ul>
  {:else}
    <section class="card result">
      <p class="hint">{res.kind === 'level' ? t('level.result') : t('result.band')}</p>
      <p class="num band">{formatBand(res.band)}</p>
      <p>{t('practice.score', { n: res.correct, total: res.total })}</p>
      <p class="hint">{res.kind === 'level' ? t('level.after') : res.ranked ? t('test.ranked') : t('test.unranked')}</p>
    </section>
  {/if}

  {#if res.mock_id}
    <button class="btn" onclick={back}>{t('mock.back')}</button>
  {:else if res.kind === 'level'}
    <!-- Тест уровня в рейтинг не идёт: сразу показать, как туда попасть. -->
    <button class="btn" onclick={() => go({ s: 'practice', part: 'reading' })}>{t('level.toRating')}</button>
    <button class="btn btn-plain" onclick={back}>{t('practice.done')}</button>
  {:else}
    <button class="btn" onclick={back}>{t('practice.done')}</button>
  {/if}

  {#if res.level}
    {#if res.level.mistakes.length}
      <button class="btn btn-plain" onclick={() => (showKey = !showKey)}>{showKey ? t('level.hideMistakes') : t('level.showMistakes', { n: res.level.mistakes.length })}</button>
    {/if}
    {#if showKey}
      {#each res.level.mistakes as m, i (i)}
        <section class="card mistake">
          <p><b>{m.q}</b></p>
          <p class="wrong">{m.given >= 0 ? t('level.yours', { a: m.options[m.given] ?? '' }) : t('level.noAnswer')}</p>
          <p class="right">{t('level.correct', { a: m.options[m.answer] ?? '' })}</p>
        </section>
      {/each}
    {/if}
  {:else}
    <button class="btn btn-plain" onclick={() => (showKey = !showKey)}>{showKey ? t('test.hideKey') : t('test.showKey')}</button>
  {/if}

  {#if showKey && !res.level}
    {#if res.transcript}
      <section class="card stack">
        <h2>{t('test.transcript')}</h2>
        {#each res.transcript.split('\n') as line, i (i)}<p>{line}</p>{/each}
      </section>
    {/if}
    <Questions test={res.test} answers={res.answers.map((a) => (a === -1 ? null : a))} key={res.key} marks={res.marks} />
  {/if}
  <button class="card row hist" onclick={() => go({ s: 'history' })}>
    <span class="grow">{t('history.title')}</span>
    <span class="chev" aria-hidden="true">›</span>
  </button>
{/if}

<style>
  .result {
    display: grid;
    gap: 4px;
    text-align: center;
  }

  .band {
    font-size: 48px;
    line-height: 1.1;
  }

  h2 {
    font-size: 16px;
  }

  .hist {
    padding: 0 16px;
  }

  .parts {
    display: grid;
    gap: 8px;
    margin: 0;
    padding: 12px 16px;
    list-style: none;
  }

  .parts li {
    display: flex;
    gap: 12px;
    align-items: baseline;
  }

  .parts .grow {
    min-width: 0;
    overflow-wrap: anywhere;
  }

  .mistake {
    display: grid;
    gap: 4px;
  }

  .wrong {
    color: var(--danger);
  }

  .right {
    color: var(--ok);
  }
</style>
