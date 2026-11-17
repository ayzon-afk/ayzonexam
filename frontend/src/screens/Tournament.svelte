<script lang="ts">
  import { onMount } from 'svelte'
  import Avatar from '../components/Avatar.svelte'
  import Countdown from '../components/Countdown.svelte'
  import Head from '../components/Head.svelte'
  import ShareCard from '../components/ShareCard.svelte'
  import Skeleton from '../components/Skeleton.svelte'
  import { api, ApiError, type TourState, type Winner } from '../lib/api'
  import { fmtDate, fmtMonth } from '../lib/date'
  import { formatBand } from '../lib/exam'
  import { t } from '../lib/i18n/index.svelte'
  import { go } from '../lib/nav.svelte'
  import { haptic } from '../lib/telegram'

  // «Турнир месяца»: запись заранее, в окно турнира — Listening и Reading с таймером, итоги — сразу после
  // окончания. Чемпиона после подтверждения Speaking ставит владелец.
  let cur = $state<TourState | null>(null)
  let last = $state<TourState | null>(null)
  let winners = $state<Winner[]>([])
  let failed = $state(false)
  let busy = $state(false)

  async function load() {
    try {
      const res = await api.tournament()
      cur = res.current
      last = res.last
      winners = res.winners
    } catch {
      failed = true
    }
  }

  async function join() {
    busy = true
    try {
      await api.joinTournament()
      haptic.success()
      await load()
    } catch (e) {
      haptic.error()
      if (!(e instanceof ApiError)) failed = true
    } finally {
      busy = false
    }
  }

  const parts: { kind: 'listening' | 'reading'; name: string; min: number }[] = [
    { kind: 'listening', name: 'Listening', min: 10 },
    { kind: 'reading', name: 'Reading', min: 20 },
  ]

  onMount(load)

  // Итоги показываются для того турнира, который уже кончился: текущий (сразу после окна) или прошлый.
  const done = $derived(cur?.finished ? cur : last?.finished ? last : null)
</script>

<Head title={t('tour.title')} sub={t('tour.lead')} demo={false} />

{#if failed}
  <p class="error">{t('app.error')}</p>
{:else if !cur}
  <Skeleton rows={4} />
{:else}
  {#if !cur.finished}
    <section class="card stack">
      <p class="when">{fmtDate(cur.starts_at, true)}</p>
      {#if cur.live}
        <p class="live">{t('tour.live')}</p>
      {:else}
        <p class="num"><Countdown to={new Date(cur.starts_at)} /></p>
      {/if}
      <p class="hint">{t('tour.players', { n: cur.players })}</p>
      {#if !cur.joined}
        <button class="btn" disabled={busy} onclick={join}>{t('tour.join')}</button>
      {:else if !cur.live}
        <p class="ok">{t('tour.joined')}</p>
      {/if}
    </section>

    {#if cur.live && cur.joined}
      <ul class="card rows">
        {#each parts as p (p.kind)}
          {@const b = cur.parts[p.kind]}
          <li>
            <button class="row" disabled={b !== undefined} onclick={() => go({ s: 'test', from: { tour: p.kind } })}>
              <span class="grow"><strong>{p.name}</strong></span>
              {#if b !== undefined}<span class="num">{formatBand(b)}</span>{:else}<span class="btn btn-small">{t('tour.startPart')}</span>{/if}
            </button>
          </li>
        {/each}
      </ul>
    {/if}
  {/if}

  {#if done}
    <div class="sect"><h2>{t('tour.results', { month: fmtMonth(done.starts_at) })}</h2></div>
    {#if done.place !== null && done.band !== null}
      <ShareCard
        big="#{done.place}"
        title={t('tour.inCountry')}
        sub="{t('result.band')}: {formatBand(done.band)}"
        text={t('share.tour', { place: done.place, band: formatBand(done.band) })}
      />
    {/if}
    {#if done.results?.length}
      <ol class="card rows">
        {#each done.results as p (p.place + p.nickname)}
          <li class="row" class:me={p.me}>
            <span class="place">{p.place}</span>
            <Avatar name={p.nickname} />
            <span class="grow">
              <strong>{p.nickname}</strong>
              {#if p.center}<span class="tag">{p.center}</span>{/if}
            </span>
            <span class="num">{formatBand(p.band)}</span>
          </li>
        {/each}
      </ol>
      <p class="hint">{t('tour.top3')}</p>
    {:else}
      <p class="card hint">{t('tour.noResults')}</p>
    {/if}
  {/if}

  <div class="sect"><h2>{t('tour.rules')}</h2></div>
  <ul class="card rules">
    <li>{t('tour.rule1')}</li>
    <li>{t('tour.rule2')}</li>
    <li>{t('tour.rule3')}</li>
    <li>{t('tour.rule4')}</li>
  </ul>

  {#if winners.length}
    <div class="sect"><h2>{t('tour.winners')}</h2></div>
    <ul class="card rows">
      {#each winners as w (w.month)}
        <li>
          <button class="row" disabled={!w.center_id} onclick={() => w.center_id && go({ s: 'center', id: w.center_id })}>
            <Avatar name={w.nickname} />
            <span class="grow">
              <strong>{w.nickname}</strong>
              <span class="tag">{fmtMonth(w.month)}{w.center ? ` · ${w.center}` : ''}</span>
            </span>
            <span class="num">{formatBand(w.band)}</span>
          </button>
        </li>
      {/each}
    </ul>
  {/if}
{/if}

<style>
  .when {
    font-size: 20px;
    font-weight: 600;
  }

  .live {
    font-weight: 700;
    color: var(--danger);
  }

  .ok {
    color: #2e9d58;
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

  .place {
    width: 28px;
    flex: none;
    text-align: center;
    font-weight: 600;
    color: var(--hint);
  }

  .me {
    background: color-mix(in srgb, var(--accent) 10%, transparent);
  }

  .row:disabled {
    color: inherit;
    cursor: default;
  }
</style>
