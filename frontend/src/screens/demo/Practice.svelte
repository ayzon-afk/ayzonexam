<script lang="ts">
  import { onDestroy } from 'svelte'
  import Choice from '../../components/Choice.svelte'
  import Head from '../../components/Head.svelte'
  import { clock } from '../../lib/date'
  import { level, listening, reading, setBand, type Q } from '../../lib/demo-exam'
  import { formatBand } from '../../lib/exam'
  import { t } from '../../lib/i18n/index.svelte'
  import type { Part } from '../../lib/nav.svelte'
  import { back } from '../../lib/nav.svelte'
  import { haptic } from '../../lib/telegram'

  let { r }: { r: { part: Part } } = $props()

  // svelte-ignore state_referenced_locally
  const part = r.part
  const qs: Q[] = part === 'reading' ? reading.questions : part === 'listening' ? listening : level
  const minutes = part === 'reading' ? 20 : part === 'listening' ? 10 : 5
  const title = part === 'reading' ? 'Reading' : part === 'listening' ? 'Listening' : t('level.title')

  let answers = $state<(number | null)[]>(qs.map(() => null))
  let done = $state(false)

  // Таймер сета; по нулю ответы сдаются сами, как на экзамене.
  const ends = Date.now() + minutes * 60_000
  let now = $state(Date.now())
  const tick = setInterval(() => {
    now = Date.now()
    if (now >= ends && !done) finish()
  }, 1000)

  // Аудио Listening в демо нет: плеер только показывает, как будет выглядеть.
  const AUDIO = 200
  let playing = $state(false)
  let pos = $state(0)
  const play = setInterval(() => {
    if (playing) pos = Math.min(pos + 1, AUDIO)
    if (pos === AUDIO) playing = false
  }, 1000)

  onDestroy(() => {
    clearInterval(tick)
    clearInterval(play)
  })

  const correct = $derived(answers.filter((a, i) => a === qs[i]!.answer).length)
  const answered = $derived(answers.filter((a) => a !== null).length)

  function finish() {
    done = true
    playing = false
    haptic.success()
    window.scrollTo(0, 0)
  }
</script>

<Head {title} sub={t('practice.sub', { n: qs.length, min: minutes })} />

{#if done}
  <section class="card result">
    <p class="hint">{part === 'level' ? t('level.result') : t('result.band')}</p>
    <p class="num band">{formatBand(setBand(correct, qs.length))}</p>
    <p>{t('practice.score', { n: correct, total: qs.length })}</p>
    <p class="hint">{t('practice.estimate')}</p>
  </section>
  {#each qs as q, i (i)}
    <Choice n={i + 1} {q} value={answers[i]} checked />
  {/each}
  <button class="btn" onclick={back}>{t('practice.done')}</button>
{:else}
  <div class="card bar">
    <span class="num">{clock(ends - now)}</span>
    <span class="hint">{t('practice.answered', { n: answered, total: qs.length })}</span>
  </div>

  {#if part === 'reading'}
    <article class="card passage">
      <h2>{reading.title}</h2>
      {#each reading.text.split('\n\n') as p, i (i)}
        <p>{p}</p>
      {/each}
    </article>
  {:else if part === 'listening'}
    <div class="card player">
      <button class="play" aria-label={playing ? t('listening.pause') : t('listening.play')} onclick={() => (playing = !playing)}>
        {playing ? '❚❚' : '▶'}
      </button>
      <span class="track"><span style:width="{(pos / AUDIO) * 100}%"></span></span>
      <span class="hint">{clock(pos * 1000)} / {clock(AUDIO * 1000)}</span>
    </div>
    <p class="hint">{t('listening.once')}</p>
  {/if}

  {#each qs as q, i (i)}
    <Choice n={i + 1} {q} bind:value={answers[i]} />
  {/each}

  <button class="btn" disabled={answered === 0} onclick={finish}>{t('practice.finish')}</button>
{/if}

<style>
  .bar {
    position: sticky;
    top: calc(var(--safe-top) + 8px);
    z-index: 1;
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 16px;
    box-shadow: 0 2px 12px rgb(0 0 0 / 0.08);
  }

  .bar .num {
    font-size: 18px;
  }

  .passage {
    display: grid;
    gap: 10px;
    line-height: 1.55;
  }

  .passage h2 {
    font-size: 18px;
  }

  .player {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .play {
    width: 44px;
    height: 44px;
    flex: none;
    border-radius: 50%;
    background: var(--accent);
    color: var(--accent-text);
    font-size: 14px;
  }

  .track {
    flex: 1;
    height: 6px;
    border-radius: 3px;
    background: var(--surface);
    overflow: hidden;
  }

  .track span {
    display: block;
    height: 100%;
    background: var(--accent);
  }

  .result {
    display: grid;
    gap: 4px;
    text-align: center;
  }

  .band {
    font-size: 48px;
    line-height: 1.1;
  }
</style>
