<script lang="ts">
  import { onMount } from 'svelte'
  import Avatar from '../../components/Avatar.svelte'
  import Head from '../../components/Head.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { api, type Me, type Pending } from '../../lib/api'
  import { fmtDate } from '../../lib/date'
  import { t } from '../../lib/i18n/index.svelte'
  import { inviteLink, share } from '../../lib/share'
  import { haptic } from '../../lib/telegram'

  // Аккаунт центра: кто просится в центр (видны только ник и аватар) и ссылка-приглашение.
  let { me }: { me: Me } = $props()

  let list = $state<Pending[] | null>(null)
  let code = $state('')
  let failed = $state(false)

  async function load() {
    try {
      const res = await api.centerRequests()
      list = res.requests
      code = res.invite_code
    } catch {
      failed = true
    }
  }

  async function decide(p: Pending, accept: boolean) {
    try {
      await api.decideRequest(p.user_id, accept)
      haptic.success()
      list = list?.filter((x) => x.user_id !== p.user_id) ?? null
    } catch {
      haptic.error()
    }
  }

  onMount(load)
</script>

<Head title={t('center.requests')} sub={me.center?.name ?? ''} demo={false} />

{#if failed}
  <p class="card">{t('center.notStaff')}</p>
{:else if !list}
  <Skeleton rows={3} />
{:else if list.length === 0}
  <p class="card hint">{t('center.noRequests')}</p>
{:else}
  <ul class="card rows">
    {#each list as p (p.user_id)}
      <li class="row">
        <Avatar name={p.nickname} />
        <span class="grow">
          <strong>{p.nickname}</strong>
          <span class="tag">{fmtDate(p.since)}</span>
        </span>
        <button class="btn btn-small" onclick={() => decide(p, true)}>{t('center.accept')}</button>
        <button class="no" aria-label={t('center.decline')} onclick={() => decide(p, false)}>×</button>
      </li>
    {/each}
  </ul>
{/if}
<p class="hint">{t('center.inviteHint')}</p>
{#if code}
  <button class="btn" onclick={() => share(t('center.inviteText', { name: me.center?.name ?? '' }), inviteLink(me.bot, `c-${code}`))}>{t('center.shareApp')}</button>
{/if}

<style>
  strong {
    display: block;
  }

  .no {
    width: 32px;
    height: 32px;
    color: var(--hint);
    font-size: 20px;
  }
</style>
