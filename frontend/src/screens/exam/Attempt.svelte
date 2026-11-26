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
  <section class="card result">
    <p class="hint">{res.kind === 'level' ? t('level.result') : t('result.band')}</p>
    <p class="num band">{formatBand(res.band)}</p>
    <p>{t('practice.score', { n: res.correct, total: res.total })}</p>
    <p class="hint">{res.kind === 'level' ? t('level.after') : res.ranked ? t('test.ranked') : t('test.unranked')}</p>
  </section>

  {#if res.mock_id}
    <button class="btn" onclick={back}>{t('mock.back')}</button>
  {:else if res.kind === 'level'}
    <!-- Тест уровня в рейтинг не идёт: сразу показать, как туда попасть. -->
    <button class="btn" onclick={() => go({ s: 'practice', part: 'reading' })}>{t('level.toRating')}</button>
    <button class="btn btn-plain" onclick={back}>{t('practice.done')}</button>
  {:else}
    <button class="btn" onclick={back}>{t('practice.done')}</button>
  {/if}

  <button class="btn btn-plain" onclick={() => (showKey = !showKey)}>{showKey ? t('test.hideKey') : t('test.showKey')}</button>

  {#if showKey}
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
</style>
