<script lang="ts">
  import Avatar from '../components/Avatar.svelte'
  import Delta from '../components/Delta.svelte'
  import Skeleton from '../components/Skeleton.svelte'
  import { api, type Board, type CenterRow, type Me, type Scope } from '../lib/api'
  import { formatBand } from '../lib/exam'
  import { t, type Key } from '../lib/i18n/index.svelte'
  import { go } from '../lib/nav.svelte'
  import { haptic } from '../lib/telegram'

  // Рейтинг по реальным баллам таймерных попыток за 60 дней: Ташкент, мой центр, друзья, страна, центры.
  let { me }: { me: Me } = $props()

  let scope = $state<Scope | 'centers'>('city')
  let board = $state<Board | null>(null)
  let centers = $state<CenterRow[] | null>(null)
  let failed = $state(false)

  const scopes: { id: Scope | 'centers'; label: Key }[] = [
    { id: 'city', label: 'rating.city' },
    { id: 'center', label: 'rating.center' },
    { id: 'friends', label: 'rating.friends' },
    { id: 'country', label: 'rating.country' },
    { id: 'centers', label: 'rating.centers' },
  ]

  // Своё место — снизу, если его нет среди первых строк.
  const showMine = $derived(board?.mine && !board.rows.some((r) => r.me))

  $effect(() => {
    const s = scope
    failed = false
    board = null
    if (s === 'centers') {
      api.centerRating().then((r) => (centers = r.centers), () => (failed = true))
    } else {
      api.rating(s).then((b) => scope === s && (board = b), () => (failed = true))
    }
  })
</script>

<h1>{t('tab.rating')}</h1>

<button class="card duel" onclick={() => go({ s: 'duel' })}>
  <span class="grow">
    <strong>{t('duel.title')}</strong>
    <span class="tag">{t('duel.lead')}</span>
  </span>
  <span class="btn btn-small">{t('duel.challenge')}</span>
</button>

<div class="seg" role="tablist">
  {#each scopes as s (s.id)}
    <button
      role="tab"
      aria-selected={scope === s.id}
      class:on={scope === s.id}
      onclick={() => {
        haptic.select()
        scope = s.id
      }}>{t(s.label)}</button
    >
  {/each}
</div>

{#if failed}
  <p class="error">{t('app.error')}</p>
{:else if scope === 'centers'}
  {#if !centers}
    <Skeleton rows={4} />
  {:else if centers.length === 0}
    <p class="card hint">{t('rating.noCenters')}</p>
  {:else}
    <ol class="card rows">
      {#each centers as c (c.id)}
        <li>
          <button class="row" onclick={() => go({ s: 'center', id: c.id })}>
            <span class="place" class:top={c.place <= 3}>{c.place}</span>
            <Avatar name={c.name} />
            <span class="grow">
              <strong>{c.name}</strong>
              <span class="tag">{t('center.students', { n: c.students })}</span>
            </span>
            <Delta n={c.delta} />
            <span class="num">{c.avg.toFixed(1)}</span>
          </button>
        </li>
      {/each}
    </ol>
  {/if}
  <p class="hint">{t('rating.centersRule')}</p>
{:else if scope === 'center' && me.center?.status !== 'member'}
  <section class="card stack">
    <p>{me.center?.status === 'pending' ? t('center.waiting', { name: me.center.name }) : t('rating.noCenter')}</p>
    <button class="btn btn-plain" onclick={() => go({ s: 'centers' })}>{t('center.choose')}</button>
  </section>
{:else if !board}
  <Skeleton rows={5} />
{:else}
  {#if board.rows.length === 0}
    <p class="card hint">{scope === 'friends' ? t('rating.noFriends') : t('rating.empty')}</p>
  {:else}
    <ol class="card rows">
      {#each board.rows as p (p.place + p.nickname)}
        <li class="row" class:me={p.me}>
          <span class="place" class:top={p.place <= 3}>{p.place}</span>
          <Avatar name={p.nickname} />
          <span class="grow">
            <strong>{p.me ? t('rating.you') : p.nickname}</strong>
            {#if p.center}<span class="tag">{p.center}</span>{/if}
          </span>
          <Delta n={p.delta} />
          <span class="num">{formatBand(p.band)}</span>
        </li>
      {/each}
    </ol>
  {/if}

  {#if showMine && board.mine}
    <div class="card row mine">
      <span class="place">{board.mine.place}</span>
      <Avatar name={me.nickname} />
      <span class="grow">
        <strong>{t('rating.you')}</strong>
        <span class="tag">{t('rating.of', { n: board.total })} <Delta n={board.mine.delta} /> {board.mine.delta ? t('delta.week') : ''}</span>
      </span>
      <span class="num">{formatBand(board.mine.band)}</span>
    </div>
  {:else if !board.mine}
    <button class="card row" onclick={() => go({ s: 'practice', part: 'reading' })}>
      <span class="grow">{t('rating.notIn')}</span>
      <span class="chev" aria-hidden="true">›</span>
    </button>
  {/if}
  {#if scope === 'friends'}
    <button class="btn btn-plain" onclick={() => go({ s: 'friends' })}>{t('friends.title')}</button>
  {/if}
  <p class="hint">{t('rating.rule')}</p>
{/if}

<style>
  h1 {
    font-size: 26px;
  }

  .duel {
    display: flex;
    align-items: center;
    gap: 12px;
    text-align: left;
  }

  .duel strong,
  .grow strong {
    display: block;
  }

  .place {
    width: 28px;
    flex: none;
    text-align: center;
    font-weight: 600;
    color: var(--hint);
    font-variant-numeric: tabular-nums;
  }

  .place.top {
    color: #d69a00;
  }

  .num {
    font-size: 18px;
  }

  li.me {
    background: color-mix(in srgb, var(--accent) 10%, transparent);
  }

  .mine {
    position: sticky;
    bottom: calc(var(--tabbar-h) + var(--safe-bottom) + 8px);
    padding: 8px 16px;
    box-shadow: 0 4px 18px rgb(0 0 0 / 0.12);
  }

  .mine .place {
    width: auto;
    min-width: 28px;
    color: var(--accent);
  }

  @media (min-width: 900px) {
    .mine {
      bottom: 16px;
    }
  }
</style>
