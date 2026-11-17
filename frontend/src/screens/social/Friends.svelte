<script lang="ts">
  import { onMount } from 'svelte'
  import Avatar from '../../components/Avatar.svelte'
  import Head from '../../components/Head.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { api, type Friend, type Me } from '../../lib/api'
  import { formatBand } from '../../lib/exam'
  import { t } from '../../lib/i18n/index.svelte'
  import { go } from '../../lib/nav.svelte'
  import { inviteLink, share } from '../../lib/share'

  // Друзья: рейтинг друзей, дуэли, бонусы за приглашённых (SPEC, «Деньги»).
  let { me }: { me: Me } = $props()

  let list = $state<Friend[] | null>(null)
  let note = $state('')

  async function load() {
    try {
      list = (await api.friends()).friends
    } catch {
      list = []
    }
  }

  async function invite() {
    const res = await share(t('friends.inviteText'), inviteLink(me.bot, `r-${me.invite}`))
    note = res === 'copied' ? t('share.copied') : ''
  }

  async function remove(f: Friend) {
    await api.unfriend(f.id).catch(() => {})
    list = list?.filter((x) => x.id !== f.id) ?? null
  }

  onMount(load)
</script>

<Head title={t('friends.title')} sub={t('premium.bonus')} demo={false} />

<button class="btn" onclick={invite}>{t('premium.invite')}</button>
{#if note}<p class="hint">{note}</p>{/if}
{#if me.plan && me.plan.bonus > 0}<p class="card">{t('quota.bonus', { n: me.plan.bonus })}</p>{/if}

{#if !list}
  <Skeleton rows={3} />
{:else if list.length === 0}
  <p class="card hint">{t('friends.empty')}</p>
{:else}
  <ul class="card rows">
    {#each list as f (f.id)}
      <li class="row">
        <Avatar name={f.nickname} />
        <span class="grow">
          <strong>{f.nickname}</strong>
          <span class="tag">{f.band === null ? t('friends.noBand') : formatBand(f.band)}</span>
        </span>
        <button class="btn btn-small" onclick={() => go({ s: 'duel' })}>{t('duel.challenge')}</button>
        <button class="del" aria-label={t('friends.remove')} onclick={() => remove(f)}>×</button>
      </li>
    {/each}
  </ul>
{/if}

<style>
  strong {
    display: block;
  }

  .del {
    width: 32px;
    height: 32px;
    color: var(--hint);
    font-size: 20px;
  }
</style>
