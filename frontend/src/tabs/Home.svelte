<script lang="ts">
  import Avatar from '../components/Avatar.svelte'
  import Countdown from '../components/Countdown.svelte'
  import Demo from '../components/Demo.svelte'
  import Icon, { type IconName } from '../components/Icon.svelte'
  import NewsList from '../components/NewsList.svelte'
  import type { Me } from '../lib/api'
  import { fmtDate } from '../lib/date'
  import { centerById, mine, news, nextTournament, winners } from '../lib/demo'
  import { daysUntil, formatBand } from '../lib/exam'
  import { t, type Key } from '../lib/i18n/index.svelte'
  import { go, type Route } from '../lib/nav.svelte'

  let { me }: { me: Me } = $props()

  const days = $derived(me.exam_date ? daysUntil(me.exam_date, new Date()) : null)
  const tour = nextTournament(new Date())
  const champ = winners[0]!

  // Путь к цели: от 4.0 до цели, где сейчас текущий балл (пока демо — настоящий даст mock).
  const left = $derived(me.goal_band === null ? null : me.goal_band - mine.band)
  const progress = $derived(me.goal_band === null ? 0 : Math.min(1, Math.max(0.04, (mine.band - 4) / (me.goal_band - 4))))

  // Быстрый старт: Writing настоящий, остальное — демо-экраны этапа 12.
  const quick: { name: string; hint: Key; icon: IconName; r: Route }[] = [
    { name: 'Writing', hint: 'quick.writing', icon: 'pen', r: { s: 'writingNew' } },
    { name: 'Reading', hint: 'quick.reading', icon: 'book', r: { s: 'practice', part: 'reading' } },
    { name: 'Listening', hint: 'quick.listening', icon: 'headphones', r: { s: 'practice', part: 'listening' } },
    { name: 'Speaking', hint: 'quick.speaking', icon: 'mic', r: { s: 'speaking' } },
  ]
</script>

<h1>{t('home.greeting', { name: me.first_name || me.nickname })}</h1>

<div class="card goal">
  <div class="ends">
    <div>
      <p class="hint">{t('profile.current')} <Demo /></p>
      <p class="band">{formatBand(mine.band)}</p>
    </div>
    <div class="right">
      <p class="hint">{t('home.goal')}</p>
      <p class="band">{me.goal_band === null ? '—' : formatBand(me.goal_band)}</p>
    </div>
  </div>
  <div class="track" role="progressbar" aria-valuenow={Math.round(progress * 100)} aria-valuemin="0" aria-valuemax="100">
    <span style:width="{progress * 100}%"></span>
  </div>
  <p class="days">
    <strong>
      {#if left === null}
        {t('goal.none')}
      {:else if left <= 0}
        {t('goal.reached')}
      {:else}
        {t('goal.left', { d: formatBand(left) })}
      {/if}
    </strong>
    <span class="hint">
      {#if days === null || days < 0}
        {t('home.noDate')}
      {:else if days === 0}
        {t('home.examToday')}
      {:else}
        {t('home.daysLeft', { days })}
      {/if}
    </span>
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
      <span class="ico"><Icon name={q.icon} /></span>
      <strong>{q.name}</strong>
      <span class="tag">{t(q.hint)}</span>
    </button>
  {/each}
  <button class="card tile" onclick={() => go({ s: 'mock' })}>
    <span class="ico"><Icon name="timer" /></span>
    <strong>{t('mock.title')}</strong>
    <span class="tag">{t('quick.mock')}</span>
  </button>
  <button class="card tile" onclick={() => go({ s: 'practice', part: 'level' })}>
    <span class="ico"><Icon name="target" /></span>
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
    display: grid;
    gap: 10px;
  }

  .ends {
    display: flex;
    justify-content: space-between;
    gap: 12px;
  }

  .right {
    text-align: right;
  }

  .band {
    font-size: 36px;
    font-weight: 700;
    line-height: 1.1;
    font-variant-numeric: tabular-nums;
    color: var(--score);
  }

  .track {
    height: 10px;
    border-radius: 5px;
    background: var(--surface);
    overflow: hidden;
  }

  .track span {
    display: block;
    height: 100%;
    border-radius: 5px;
    background: var(--grad);
  }

  .days {
    display: flex;
    flex-wrap: wrap;
    justify-content: space-between;
    gap: 4px 12px;
  }

  .tour {
    display: grid;
    justify-items: start;
    gap: 6px;
    text-align: left;
    background: var(--grad);
    color: #fff;
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
    background: #fff;
    color: #111;
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

  .ico {
    display: grid;
    place-items: center;
    width: 38px;
    height: 38px;
    margin-bottom: 6px;
    border-radius: 11px;
    background: color-mix(in srgb, var(--accent) 14%, transparent);
    color: var(--accent);
  }

  .tile {
    display: grid;
    gap: 2px;
    align-content: start;
    min-height: 76px;
    text-align: left;
  }
</style>
