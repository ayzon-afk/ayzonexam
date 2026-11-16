<script lang="ts">
  import { onDestroy } from 'svelte'
  import Head from '../../components/Head.svelte'
  import { clock, fmtDate } from '../../lib/date'
  import { attempts } from '../../lib/demo'
  import { formatBand } from '../../lib/exam'
  import { t, type Key } from '../../lib/i18n/index.svelte'
  import { go } from '../../lib/nav.svelte'

  // Начатый mock: две части сданы, Writing идёт, Speaking впереди. Сдавать можно частями.
  const parts: { name: string; min: number; state: 'done' | 'now' | 'next'; band?: number }[] = [
    { name: 'Listening', min: 30, state: 'done', band: 6.5 },
    { name: 'Reading', min: 60, state: 'done', band: 7.0 },
    { name: 'Writing', min: 60, state: 'now' },
    { name: 'Speaking', min: 14, state: 'next' },
  ]
  const label: Record<'done' | 'now' | 'next', Key> = { done: 'mock.done', now: 'mock.now', next: 'mock.next' }

  const ends = Date.now() + (38 * 60 + 12) * 1000
  let now = $state(Date.now())
  const tick = setInterval(() => (now = Date.now()), 1000)
  onDestroy(() => clearInterval(tick))

  const past = attempts.filter((a) => a.kind === 'mock')
</script>

<Head title={t('mock.title')} sub={t('mock.lead')} />

<section class="card stack">
  <div class="top">
    <h2>{t('mock.current')}</h2>
    <span class="num">{clock(ends - now)}</span>
  </div>
  <ol class="rows parts">
    {#each parts as p (p.name)}
      <li class="row">
        <span class="dot {p.state}" aria-hidden="true"></span>
        <span class="grow">
          <strong>{p.name}</strong>
          <span class="tag">{t('time.min', { n: p.min })} · {t(label[p.state])}</span>
        </span>
        {#if p.band}<span class="num">{formatBand(p.band)}</span>{/if}
      </li>
    {/each}
  </ol>
  <button class="btn" onclick={() => go({ s: 'writingNew' })}>{t('mock.continue')}</button>
  <p class="hint">{t('mock.saved')}</p>
</section>

<div class="sect"><h2>{t('mock.past')}</h2></div>
<ul class="card rows">
  {#each past as a (a.date)}
    <li>
      <button class="row" onclick={() => go({ s: 'report' })}>
        <span class="grow">{t('mock.title')} · {fmtDate(a.date)}</span>
        <span class="num">{formatBand(a.band)}</span>
        <span class="chev" aria-hidden="true">›</span>
      </button>
    </li>
  {/each}
</ul>

<style>
  .top {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
  }

  h2 {
    font-size: 17px;
  }

  .top .num {
    font-size: 20px;
  }

  .parts {
    padding: 0;
  }

  strong {
    display: block;
    font-weight: 600;
  }

  .dot {
    width: 12px;
    height: 12px;
    flex: none;
    border-radius: 50%;
    border: 2px solid var(--hint);
  }

  .dot.done {
    background: #2e9d58;
    border-color: #2e9d58;
  }

  .dot.now {
    border-color: var(--accent);
    background: color-mix(in srgb, var(--accent) 35%, transparent);
  }
</style>
