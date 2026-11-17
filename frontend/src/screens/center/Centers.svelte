<script lang="ts">
  import { onMount } from 'svelte'
  import Avatar from '../../components/Avatar.svelte'
  import Head from '../../components/Head.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { api, type CenterInfo, type Me } from '../../lib/api'
  import { t } from '../../lib/i18n/index.svelte'
  import { go } from '../../lib/nav.svelte'
  import { haptic } from '../../lib/telegram'

  // Выбор своего центра: из списка (центр подтвердит) или кодом из ссылки-приглашения (сразу).
  let { me, onchange }: { me: Me; onchange: (me: Me) => void } = $props()

  let list = $state<CenterInfo[] | null>(null)
  let code = $state('')
  let error = $state('')
  let busy = $state(false)

  onMount(async () => {
    try {
      list = (await api.centers()).centers
    } catch {
      list = []
    }
  })

  async function join() {
    busy = true
    error = ''
    try {
      // Код можно вставить и целой ссылкой: берётся часть после «c-».
      const c = code.trim().replace(/^.*c-/, '')
      onchange(await api.joinCenter(c))
      haptic.success()
    } catch {
      haptic.error()
      error = t('center.badCode')
    } finally {
      busy = false
    }
  }

  async function leave() {
    busy = true
    try {
      onchange(await api.leaveCenter())
    } finally {
      busy = false
    }
  }
</script>

<Head title={t('center.choose')} sub={t('center.chooseLead')} demo={false} />

{#if me.center}
  <section class="card stack">
    <p>{me.center.status === 'member' ? t('center.youAreIn', { name: me.center.name }) : t('center.waiting', { name: me.center.name })}</p>
    <button class="btn btn-plain" disabled={busy} onclick={leave}>{t('center.leave')}</button>
  </section>
{/if}

<section class="card stack">
  <h2>{t('center.byCode')}</h2>
  <input bind:value={code} placeholder={t('center.codePlaceholder')} autocomplete="off" />
  <button class="btn" disabled={busy || code.trim().length < 6} onclick={join}>{t('center.joinCode')}</button>
  {#if error}<p class="error">{error}</p>{/if}
</section>

{#if !list}
  <Skeleton rows={3} />
{:else if list.length}
  <div class="sect"><h2>{t('center.list')}</h2></div>
  <ul class="card rows">
    {#each list as c (c.id)}
      <li>
        <button class="row" onclick={() => go({ s: 'center', id: c.id })}>
          <Avatar name={c.name} />
          <span class="grow">
            <strong>{c.name}</strong>
            <span class="tag">{t('center.students', { n: c.students })}</span>
          </span>
          <span class="chev" aria-hidden="true">›</span>
        </button>
      </li>
    {/each}
  </ul>
{/if}

<style>
  h2 {
    font-size: 16px;
  }

  input {
    width: 100%;
    min-height: 44px;
    padding: 0 14px;
    border: 1px solid var(--separator);
    border-radius: var(--radius);
    background: var(--section);
    color: var(--text);
    font: inherit;
  }

  strong {
    display: block;
  }
</style>
