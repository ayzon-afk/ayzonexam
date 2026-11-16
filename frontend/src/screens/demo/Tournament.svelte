<script lang="ts">
  import Avatar from '../../components/Avatar.svelte'
  import Countdown from '../../components/Countdown.svelte'
  import Head from '../../components/Head.svelte'
  import { fmtDate, fmtMonth } from '../../lib/date'
  import { centerById, nextTournament, TOURNAMENT_PLAYERS, winners } from '../../lib/demo'
  import { formatBand } from '../../lib/exam'
  import { t } from '../../lib/i18n/index.svelte'
  import { go } from '../../lib/nav.svelte'
  import { haptic } from '../../lib/telegram'

  const tour = nextTournament(new Date())
  let joined = $state(false)

  const parts = [
    { name: 'Listening', min: 30 },
    { name: 'Reading', min: 60 },
    { name: 'Writing · Task 2', min: 40 },
  ]
</script>

<Head title={t('tour.title')} sub={t('tour.lead')} />

<section class="card stack">
  <p class="when">{fmtDate(tour, true)}</p>
  <p class="num"><Countdown to={tour} /></p>
  <p class="hint">{t('tour.players', { n: TOURNAMENT_PLAYERS + (joined ? 1 : 0) })}</p>
  <button
    class="btn"
    disabled={joined}
    onclick={() => {
      haptic.success()
      joined = true
    }}>{joined ? t('tour.joined') : t('tour.join')}</button
  >
</section>

<div class="sect"><h2>{t('tour.parts')}</h2></div>
<ul class="card rows">
  {#each parts as p (p.name)}
    <li class="row">
      <span class="grow">{p.name}</span>
      <span class="hint">{t('time.min', { n: p.min })}</span>
    </li>
  {/each}
</ul>

<div class="sect"><h2>{t('tour.rules')}</h2></div>
<ul class="card rules">
  <li>{t('tour.rule1')}</li>
  <li>{t('tour.rule2')}</li>
  <li>{t('tour.rule3')}</li>
  <li>{t('tour.rule4')}</li>
</ul>

<div class="sect"><h2>{t('tour.winners')}</h2></div>
<ul class="card rows">
  {#each winners as w (w.month)}
    <li>
      <button class="row" onclick={() => go({ s: 'center', id: w.center })}>
        <Avatar name={w.nick} />
        <span class="grow">
          <strong>{w.nick}</strong>
          <span class="tag">{fmtMonth(w.month)} · {centerById(w.center).name}</span>
        </span>
        <span class="num">{formatBand(w.band)}</span>
      </button>
    </li>
  {/each}
</ul>

<style>
  .when {
    font-size: 20px;
    font-weight: 600;
  }

  .num {
    font-size: 18px;
  }

  .rules {
    margin: 0;
    padding: 16px 16px 16px 34px;
    display: grid;
    gap: 8px;
  }

  strong {
    display: block;
  }
</style>
