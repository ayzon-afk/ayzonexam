<script lang="ts">
  import Avatar from '../components/Avatar.svelte'
  import Demo from '../components/Demo.svelte'
  import type { Me } from '../lib/api'
  import { centerById, centers, friends, mine, MY_CENTER, players, type Scope } from '../lib/demo'
  import { formatBand } from '../lib/exam'
  import { t, type Key } from '../lib/i18n/index.svelte'
  import { go } from '../lib/nav.svelte'
  import { haptic } from '../lib/telegram'

  let { me }: { me: Me } = $props()

  let scope = $state<Scope | 'centers'>('city')

  const scopes: { id: Scope | 'centers'; label: Key }[] = [
    { id: 'city', label: 'rating.city' },
    { id: 'center', label: 'rating.center' },
    { id: 'friends', label: 'rating.friends' },
    { id: 'country', label: 'rating.country' },
    { id: 'centers', label: 'rating.centers' },
  ]

  const list = $derived(
    scope === 'center'
      ? players.filter((p) => p.center === MY_CENTER)
      : scope === 'friends'
        ? players.filter((p) => friends.includes(p.nick))
        : players,
  )
</script>

<h1>{t('tab.rating')} <Demo /></h1>

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

{#if scope === 'centers'}
  <ol class="card rows">
    {#each centers as c, i (c.id)}
      <li>
        <button class="row" onclick={() => go({ s: 'center', id: c.id })}>
          <span class="place" class:top={i < 3}>{i + 1}</span>
          <Avatar name={c.name} />
          <span class="grow">
            <strong>{c.name}</strong>
            <span class="tag">{t('center.students', { n: c.students })}</span>
          </span>
          <span class="num">{c.avg.toFixed(1)}</span>
        </button>
      </li>
    {/each}
  </ol>
  <p class="hint">{t('rating.centersRule')}</p>
{:else}
  <ol class="card rows">
    {#each list as p, i (p.nick)}
      <li class="row">
        <span class="place" class:top={i < 3}>{i + 1}</span>
        <Avatar name={p.nick} />
        <span class="grow">
          <strong>{p.nick}</strong>
          <span class="tag">{centerById(p.center).name}</span>
        </span>
        <span class="num">{formatBand(p.band)}</span>
      </li>
    {/each}
  </ol>

  <div class="card row mine">
    <span class="place">{mine.place[scope]}</span>
    <Avatar name={me.nickname} />
    <span class="grow">
      <strong>{t('rating.you')}</strong>
      <span class="tag">{t('rating.of', { n: mine.total[scope] })}</span>
    </span>
    <span class="num">{formatBand(mine.band)}</span>
  </div>
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
    color: #c98a00;
  }

  .num {
    font-size: 18px;
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
