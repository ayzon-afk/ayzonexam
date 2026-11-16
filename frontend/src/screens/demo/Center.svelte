<script lang="ts">
  import Avatar from '../../components/Avatar.svelte'
  import Head from '../../components/Head.svelte'
  import { centerById, centers, MY_CENTER, players } from '../../lib/demo'
  import { formatBand } from '../../lib/exam'
  import { t } from '../../lib/i18n/index.svelte'
  import { haptic } from '../../lib/telegram'

  let { r }: { r: { id: number } } = $props()

  const c = $derived(centerById(r.id))
  const place = $derived(centers.indexOf(c) + 1)
  const top = $derived(players.filter((p) => p.center === c.id).slice(0, 5))
  let requested = $state(false)
</script>

<Head title={c.name} sub={t('center.city')} />

<section class="card hero">
  <Avatar name={c.name} size={64} />
  <div class="stats">
    <div><p class="num">{c.avg.toFixed(1)}</p><p class="hint">{t('center.avg')}</p></div>
    <div><p class="num">{c.students}</p><p class="hint">{t('center.studentsShort')}</p></div>
    <div><p class="num">#{place}</p><p class="hint">{t('center.place')}</p></div>
  </div>
</section>

{#if c.id === MY_CENTER}
  <p class="card mine">{t('center.yours')}</p>
{:else}
  <button
    class="btn"
    disabled={requested}
    onclick={() => {
      haptic.success()
      requested = true
    }}>{requested ? t('center.requested') : t('center.join')}</button
  >
{/if}

<div class="sect"><h2>{t('center.best')}</h2></div>
<ol class="card rows">
  {#each top as p, i (p.nick)}
    <li class="row">
      <span class="hint">{i + 1}</span>
      <Avatar name={p.nick} />
      <span class="grow">{p.nick}</span>
      <span class="num">{formatBand(p.band)}</span>
    </li>
  {/each}
</ol>
<p class="hint">{t('center.consent')}</p>

<style>
  .hero {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .stats {
    flex: 1;
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 8px;
    text-align: center;
  }

  .stats .num {
    font-size: 22px;
  }

  .mine {
    text-align: center;
    font-weight: 500;
  }
</style>
