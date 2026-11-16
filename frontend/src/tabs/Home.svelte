<script lang="ts">
  import Avatar from '../components/Avatar.svelte'
  import Countdown from '../components/Countdown.svelte'
  import Demo from '../components/Demo.svelte'
  import NewsList from '../components/NewsList.svelte'
  import type { Me } from '../lib/api'
  import { fmtDate } from '../lib/date'
  import { centerById, news, nextTournament, winners } from '../lib/demo'
  import { daysUntil, formatBand } from '../lib/exam'
  import { t, type Key } from '../lib/i18n/index.svelte'
  import { go, type Route } from '../lib/nav.svelte'

  let { me }: { me: Me } = $props()

  const days = $derived(me.exam_date ? daysUntil(me.exam_date, new Date()) : null)
  const tour = nextTournament(new Date())
  const champ = winners[0]!

  // Быстрый старт: Writing настоящий, остальное — демо-экраны этапа 12.
  const quick: { name: string; hint: Key; r: Route }[] = [
    { name: 'Writing', hint: 'quick.writing', r: { s: 'writingNew' } },
    { name: 'Reading', hint: 'quick.reading', r: { s: 'practice', part: 'reading' } },
    { name: 'Listening', hint: 'quick.listening', r: { s: 'practice', part: 'listening' } },
    { name: 'Speaking', hint: 'quick.speaking', r: { s: 'speaking' } },
  ]
</script>

<h1>{t('home.greeting', { name: me.first_name || me.nickname })}</h1>

<div class="card goal">
  <div>
    <p class="hint">{t('home.goal')}</p>
    <p class="band">{me.goal_band === null ? '—' : formatBand(me.goal_band)}</p>
  </div>
  <p class="days">
    {#if days === null || days < 0}
      {t('home.noDate')}
    {:else if days === 0}
      {t('home.examToday')}
    {:else}
      {t('home.daysLeft', { days })}
    {/if}
  </p>
</div>

<button class="card tour" onclick={() => go({ s: 'tournament' })}>
  <span class="kicker">{t('tour.title')} <Demo /></span>
  <strong>{fmtDate(tour, true)}</strong>
  <span class="left"><Countdown to={tour} /></span>
  <span class="btn btn-small">{t('tour.join')}</span>
</button>

<button class="card champ" onclick={() => go({ s: 'center', id: champ.center })}>
  <Avatar name={champ.nick} size={48} />
  <span class="grow">
    <span class="tag">{t('champ.title')}</span>
    <strong>{champ.nick}</strong>
    <span class="tag">{centerById(champ.center).name}</span>
  </span>
  <span class="num big">{formatBand(champ.band)}</span>
</button>

<div class="sect"><h2>{t('quick.title')}</h2></div>
<div class="tiles">
  {#each quick as q (q.name)}
    <button class="card tile" onclick={() => go(q.r)}>
      <strong>{q.name}</strong>
      <span class="tag">{t(q.hint)}</span>
    </button>
  {/each}
  <button class="card tile" onclick={() => go({ s: 'mock' })}>
    <strong>{t('mock.title')}</strong>
    <span class="tag">{t('quick.mock')}</span>
  </button>
  <button class="card tile" onclick={() => go({ s: 'practice', part: 'level' })}>
    <strong>{t('level.title')}</strong>
    <span class="tag">{t('quick.level')}</span>
  </button>
</div>

<div class="sect">
  <h2>{t('news.title')} <Demo /></h2>
  <button class="more" onclick={() => go({ s: 'news' })}>{t('more')}</button>
</div>
<NewsList items={news.slice(0, 3)} />

<style>
  h1 {
    font-size: 26px;
  }

  .goal {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .band {
    font-size: 40px;
    font-weight: 700;
    line-height: 1.1;
    font-variant-numeric: tabular-nums;
    color: var(--accent);
  }

  .days {
    text-align: right;
    font-weight: 500;
  }

  .tour {
    display: grid;
    justify-items: start;
    gap: 6px;
    text-align: left;
    background: linear-gradient(135deg, var(--accent), color-mix(in srgb, var(--accent) 60%, #7b2ff7));
    color: var(--accent-text);
  }

  .tour .kicker {
    font-size: 13px;
    font-weight: 600;
    opacity: 0.9;
  }

  .tour strong {
    font-size: 20px;
  }

  .tour .left {
    font-variant-numeric: tabular-nums;
  }

  .tour .btn {
    margin-top: 4px;
    background: var(--accent-text);
    color: var(--accent);
  }

  .tour :global(.demo) {
    border-color: currentColor;
    color: inherit;
  }

  .champ {
    display: flex;
    align-items: center;
    gap: 12px;
    text-align: left;
  }

  .champ strong {
    display: block;
    font-size: 17px;
  }

  .big {
    font-size: 28px;
  }

  .tiles {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 10px;
  }

  .tile {
    display: grid;
    gap: 2px;
    align-content: start;
    min-height: 76px;
    text-align: left;
  }
</style>
