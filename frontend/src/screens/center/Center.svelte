<script lang="ts">
  import { onMount } from 'svelte'
  import Avatar from '../../components/Avatar.svelte'
  import Head from '../../components/Head.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { api, type CenterInfo, type Me } from '../../lib/api'
  import { formatBand } from '../../lib/exam'
  import { t } from '../../lib/i18n/index.svelte'
  import { go } from '../../lib/nav.svelte'
  import { haptic } from '../../lib/telegram'

  // Страница центра: средний балл учеников, место среди центров, лучшие ученики, «присоединиться».
  let { r, me, onchange }: { r: { id: number }; me: Me; onchange: (me: Me) => void } = $props()

  let c = $state<CenterInfo | null>(null)
  let status = $state<'pending' | 'member' | null>(null)
  let failed = $state(false)
  let busy = $state(false)

  async function load() {
    try {
      const res = await api.center(r.id)
      c = res.center
      status = res.status
    } catch {
      failed = true
    }
  }

  async function request() {
    busy = true
    try {
      onchange(await api.requestCenter(r.id))
      status = 'pending'
      haptic.success()
    } catch {
      haptic.error()
    } finally {
      busy = false
    }
  }

  onMount(load)
</script>

{#if failed}
  <Head title={t('profile.center')} demo={false} />
  <p class="error">{t('app.error')}</p>
{:else if !c}
  <Skeleton rows={4} />
{:else}
  <Head title={c.name} sub={c.city === 'tashkent' ? t('rating.city') : t('city.other')} demo={false} />
  <section class="card hero">
    <Avatar name={c.name} size={64} />
    <div class="stats">
      <div><p class="num">{c.avg === null ? '—' : c.avg.toFixed(1)}</p><p class="hint">{t('center.avg')}</p></div>
      <div><p class="num">{c.students}</p><p class="hint">{t('center.studentsShort')}</p></div>
      <div><p class="num">{c.place === null ? '—' : `#${c.place}`}</p><p class="hint">{t('center.place')}</p></div>
    </div>
  </section>
  {#if c.about}<p class="card about">{c.about}</p>{/if}

  {#if status === 'member'}
    <p class="card mine">{t('center.yours')}</p>
    {#if me.center?.role}
      <button class="btn btn-plain" onclick={() => go({ s: 'centerRequests' })}>{t('center.requests')}</button>
    {/if}
  {:else if status === 'pending'}
    <p class="card mine">{t('center.requested')}</p>
  {:else}
    <button class="btn" disabled={busy} onclick={request}>{t('center.join')}</button>
    {#if me.center}<p class="hint">{t('center.switch', { name: me.center.name })}</p>{/if}
  {/if}

  {#if c.top?.length}
    <div class="sect"><h2>{t('center.best')}</h2></div>
    <ol class="card rows">
      {#each c.top as p (p.nickname)}
        <li class="row">
          <span class="hint">{p.place}</span>
          <Avatar name={p.nickname} />
          <span class="grow">{p.nickname}</span>
          <span class="num">{formatBand(p.band)}</span>
        </li>
      {/each}
    </ol>
  {/if}
  <p class="hint">{t('rating.centersRule')}</p>
{/if}

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

  .about {
    white-space: pre-wrap;
  }

  .mine {
    text-align: center;
    font-weight: 500;
  }
</style>
