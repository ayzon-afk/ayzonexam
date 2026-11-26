<script lang="ts">
  import Head from '../../components/Head.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { ApiError, type Me } from '../../lib/api'
  import { t } from '../../lib/i18n/index.svelte'
  import type { Route } from '../../lib/nav.svelte'
  import { cab, cabTitle, sectionName, sectionsFor, siteURL, type Cabinet } from './cabinet'

  // Кабинет центра в Mini App: те же разделы, что на сайте (CabinetSite), — переключателем сверху.
  // Кнопка бота «Заявки» (open=center) открывает его сразу на заявках.
  let { r, me }: { r: Route; me: Me } = $props()

  let data = $state<Cabinet | null>(null)
  let denied = $state(false)
  let failed = $state(false)
  const route = $derived(r as Extract<Route, { s: 'cabinet' | 'centerRequests' }>)
  const current = $derived(route.section ?? (route.s === 'centerRequests' ? 'requests' : 'overview'))

  async function load() {
    failed = false
    try {
      data = await cab.get()
    } catch (e) {
      if (e instanceof ApiError && e.status === 403) denied = true
      else failed = true
    }
  }
  load()

  const list = $derived(data ? sectionsFor(data.role) : [])
  const section = $derived(list.find((s) => s.id === current) ?? list[0])

  function goto(id: string) {
    route.section = id
    window.scrollTo(0, 0)
  }
</script>

<Head title={t(cabTitle(data?.center.kind ?? me.center?.kind))} sub={data?.center.name ?? me.center?.name ?? ''} demo={false} />

{#if denied}
  <p class="card">{t('cab.noAccess')}</p>
{:else if failed}
  <p class="error">{t('cab.loadFail')}</p>
  <button class="btn btn-plain" onclick={load}>{t('app.retry')}</button>
{:else if !data || !section}
  <Skeleton rows={4} />
{:else}
  <div class="seg" role="tablist">
    {#each list as s (s.id)}
      <button role="tab" aria-selected={section.id === s.id} class:on={section.id === s.id} onclick={() => goto(s.id)}>
        {t(sectionName(s, data.center.kind))}{#if s.id === 'requests' && data.center.pending}&nbsp;· {data.center.pending}{/if}
      </button>
    {/each}
  </div>
  {#key section.id}
    {#await section.load()}
      <Skeleton rows={4} />
    {:then m}
      <m.default {me} {data} onchange={(c: Cabinet) => (data = c)} {goto} />
    {:catch}
      <p class="error">{t('cab.loadFail')}</p>
    {/await}
  {/key}
  <p class="hint site">{t('cab.siteHint', { url: siteURL().replace(/^https?:\/\//, '') })}</p>
{/if}

<style>
  .site {
    overflow-wrap: anywhere;
  }
</style>
