<script lang="ts">
  import { onMount } from 'svelte'
  import Choice from '../../components/Choice.svelte'
  import Head from '../../components/Head.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { api, type LevelState } from '../../lib/api'
  import { t } from '../../lib/i18n/index.svelte'
  import { replace } from '../../lib/nav.svelte'

  // Тест уровня (DECISIONS 2026-11-27): сервер выдаёт по одному вопросу и по ответам выбирает следующий —
  // сильнее отвечает, труднее вопрос. После 12 вопросов — короткий текст Reading под найденный уровень.
  // Закрыл приложение — тест продолжится с того же места (сервер помнит 3 часа).

  let cur = $state<LevelState | null>(null)
  let failed = $state(false)
  let busy = $state(false)
  let choice = $state<number | null>(null)
  let reading = $state<(number | null)[]>([])

  async function run(f: () => Promise<LevelState>) {
    busy = true
    failed = false
    try {
      cur = await f()
      choice = null
      if (cur.passage && reading.length !== cur.passage.questions.length) reading = cur.passage.questions.map(() => null)
      window.scrollTo(0, 0)
    } catch {
      failed = true
    } finally {
      busy = false
    }
  }

  const start = () => run(api.levelStart)

  // answer — номер варианта; -1 — «не знаю»: честнее угадывания, и балл точнее.
  function answer(a: number) {
    if (!cur?.item || busy) return
    const { id, item } = cur
    run(() => api.levelAnswer(id, item.id, a))
  }

  async function finish() {
    if (!cur || busy) return
    busy = true
    failed = false
    try {
      const { id } = await api.levelFinish(cur.id, reading.map((a) => a ?? -1))
      replace({ s: 'attempt', id })
    } catch {
      failed = true
      busy = false
    }
  }

  onMount(start)
</script>

<Head title={t('level.title')} sub={cur ? t('level.step', { n: Math.min(cur.step, cur.steps), total: cur.steps }) : ''} demo={false} />

{#if cur}
  <div class="bar" role="progressbar" aria-valuemin={0} aria-valuemax={cur.steps} aria-valuenow={cur.step - 1}>
    <span style:width="{((cur.step - 1) / cur.steps) * 100}%"></span>
  </div>
{/if}

{#if failed && !cur}
  <p class="error">{t('app.error')}</p>
  <button class="btn btn-plain" onclick={start}>{t('app.retry')}</button>
{:else if !cur}
  <Skeleton rows={3} />
{:else if cur.item}
  <p class="hint">{t('level.honest')}</p>
  {#key cur.item.id}
    <Choice n={cur.step} q={cur.item} bind:value={choice} />
  {/key}
  {#if failed}<p class="error">{t('app.error')}</p>{/if}
  <button class="btn" disabled={choice === null || busy} onclick={() => answer(choice ?? -1)}>{t('level.next')}</button>
  <button class="btn btn-plain" disabled={busy} onclick={() => answer(-1)}>{t('level.skip')}</button>
{:else if cur.passage}
  <p class="hint">{t('level.reading')}</p>
  <article class="card passage">
    <h2>{cur.passage.title}</h2>
    {#each cur.passage.text.split('\n\n') as p, i (i)}<p>{p}</p>{/each}
  </article>
  {#each cur.passage.questions as q, i (i)}
    <Choice n={cur.step + i} {q} bind:value={reading[i]} />
  {/each}
  {#if failed}<p class="error">{t('app.error')}</p>{/if}
  <button class="btn" disabled={busy} onclick={finish}>{t('level.finish')}</button>
{/if}

<style>
  .bar {
    height: 6px;
    border-radius: 3px;
    background: var(--separator);
    overflow: hidden;
  }

  .bar span {
    display: block;
    height: 100%;
    background: var(--accent);
    transition: width 0.25s ease;
  }

  .passage {
    display: grid;
    gap: 10px;
  }

  .passage h2 {
    font-size: 18px;
  }

  .passage p {
    line-height: 1.55;
  }
</style>
