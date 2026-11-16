<script lang="ts">
  import { onDestroy } from 'svelte'
  import Head from '../../components/Head.svelte'
  import { clock } from '../../lib/date'
  import { speaking } from '../../lib/demo-exam'
  import { formatBand } from '../../lib/exam'
  import { t } from '../../lib/i18n/index.svelte'
  import { haptic } from '../../lib/telegram'

  // Демо: микрофон не включается, запись и оценка только показывают будущий порядок экранов.
  type Phase = 'ready' | 'rec' | 'wait' | 'done'
  let phase = $state<Phase>('ready')
  let part = $state(0)
  let secs = $state(0)
  let timer: ReturnType<typeof setInterval> | undefined
  let wait: ReturnType<typeof setTimeout> | undefined

  function start() {
    haptic.select()
    phase = 'rec'
    secs = 0
    timer = setInterval(() => (secs += 1), 1000)
  }

  function stop() {
    clearInterval(timer)
    phase = 'wait'
    wait = setTimeout(() => {
      haptic.success()
      phase = 'done'
    }, 1500)
  }

  onDestroy(() => {
    clearInterval(timer)
    clearTimeout(wait)
  })
</script>

<Head title="Speaking" sub={t('speaking.lead')} />

{#if phase === 'done'}
  <section class="card result">
    <p class="hint">{t('result.band')}</p>
    <p class="num band">{formatBand(speaking.band)}</p>
  </section>
  <ul class="card rows">
    {#each speaking.crit as c (c.name)}
      <li class="row">
        <span class="grow">{c.name}</span>
        <span class="num">{formatBand(c.band)}</span>
      </li>
    {/each}
  </ul>
  <section class="card stack">
    <h2>{t('speaking.notes')}</h2>
    {#each speaking.notes as n (n)}
      <p>{t(n)}</p>
    {/each}
  </section>
  <section class="card stack">
    <h2>{t('speaking.transcript')}</h2>
    <p class="hint">{speaking.transcript}</p>
  </section>
  <p class="hint">{t('result.disclaimer')}</p>
  <button class="btn" onclick={() => (phase = 'ready')}>{t('speaking.again')}</button>
{:else}
  <div class="seg" role="tablist">
    {#each speaking.parts as _, i (i)}
      <button role="tab" aria-selected={part === i} class:on={part === i} disabled={phase !== 'ready'} onclick={() => (part = i)}
        >Part {i + 1}</button
      >
    {/each}
  </div>

  <section class="card question">
    <p class="hint">{t(part === 1 ? 'speaking.part2' : 'speaking.part13')}</p>
    <p class="q">{speaking.parts[part]}</p>
  </section>

  <div class="mic">
    {#if phase === 'wait'}
      <p class="hint">{t('speaking.checking')}</p>
    {:else}
      <button class="rec" class:on={phase === 'rec'} onclick={phase === 'rec' ? stop : start}>
        {phase === 'rec' ? '■' : '●'}
      </button>
      <p class="num">{clock(secs * 1000)}</p>
      <p class="hint">{phase === 'rec' ? t('speaking.stop') : t('speaking.start')}</p>
    {/if}
  </div>

  <p class="hint">{t('speaking.fallback')}</p>
{/if}

<style>
  h2 {
    font-size: 16px;
  }

  .question {
    display: grid;
    gap: 6px;
  }

  .q {
    font-size: 18px;
    line-height: 1.45;
  }

  .mic {
    display: grid;
    justify-items: center;
    gap: 6px;
    padding: 24px 0;
  }

  .rec {
    width: 84px;
    height: 84px;
    border-radius: 50%;
    background: var(--accent);
    color: var(--accent-text);
    font-size: 28px;
  }

  .rec.on {
    background: var(--danger);
    animation: pulse 1.4s ease-in-out infinite;
  }

  @keyframes pulse {
    50% {
      box-shadow: 0 0 0 14px color-mix(in srgb, var(--danger) 18%, transparent);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .rec.on {
      animation: none;
    }
  }

  .mic .num {
    font-size: 20px;
  }

  .result {
    text-align: center;
  }

  .band {
    font-size: 48px;
    line-height: 1.1;
  }
</style>
