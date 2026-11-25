<script lang="ts">
  import Avatar from '../../components/Avatar.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { api, type Pending } from '../../lib/api'
  import { fmtDate } from '../../lib/date'
  import { t } from '../../lib/i18n/index.svelte'
  import { haptic } from '../../lib/telegram'
  import Invite from './Invite.svelte'
  import type { SectionProps } from './cabinet'

  // Кто просится в центр (выбрал его из списка): видны только ник и аватар. Ниже — ссылка, по которой
  // ученик попадает в центр сразу, без заявки.
  let { me, data, onchange }: SectionProps = $props()

  let list = $state<Pending[] | null>(null)
  let failed = $state(false)
  let busy = $state<number | null>(null)

  async function load() {
    failed = false
    try {
      list = (await api.centerRequests()).requests
    } catch {
      failed = true
    }
  }
  load()

  async function decide(p: Pending, accept: boolean) {
    busy = p.user_id
    try {
      await api.decideRequest(p.user_id, accept)
      haptic.success()
      list = list?.filter((x) => x.user_id !== p.user_id) ?? null
      const c = data.center
      onchange({ ...data, center: { ...c, pending: Math.max(0, c.pending - 1), students: c.students + (accept ? 1 : 0) } })
    } catch {
      haptic.error()
    } finally {
      busy = null
    }
  }
</script>

{#if failed}
  <p class="error">{t('cab.loadFail')}</p>
  <button class="btn btn-plain" onclick={load}>{t('app.retry')}</button>
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
          <strong class="nick">{p.nickname}</strong>
          <span class="tag">{fmtDate(p.since)}</span>
        </span>
        <span class="acts">
          <button class="btn btn-small" disabled={busy === p.user_id} onclick={() => decide(p, true)}>{t('center.accept')}</button>
          <button class="btn btn-small btn-plain" disabled={busy === p.user_id} onclick={() => decide(p, false)}>{t('center.decline')}</button>
        </span>
      </li>
    {/each}
  </ul>
{/if}

<section class="card">
  <h2>{t('cab.invite')}</h2>
  <Invite {me} name={data.center.name} code={data.center.invite_code} />
</section>

<style>
  .row {
    flex-wrap: wrap;
  }

  /* Узко — кнопки уходят на вторую строку, а не сжимают ник и дату. */
  .row .grow {
    flex: 1 1 150px;
  }

  .nick {
    display: block;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }

  .acts {
    display: flex;
    gap: 6px;
    margin-left: auto;
  }

  .acts .btn-plain {
    background: var(--surface);
  }

  section {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: 12px;
  }

  h2 {
    font-size: 17px;
  }
</style>
