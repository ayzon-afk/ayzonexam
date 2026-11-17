<script lang="ts">
  import { onDestroy, onMount } from 'svelte'
  import Avatar from '../../components/Avatar.svelte'
  import Head from '../../components/Head.svelte'
  import ShareCard from '../../components/ShareCard.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { api, ApiError, type Duel, type DuelQ, type Friend, type Me } from '../../lib/api'
  import { clock } from '../../lib/date'
  import { t, type Key } from '../../lib/i18n/index.svelte'
  import { go } from '../../lib/nav.svelte'
  import { inviteLink, share } from '../../lib/share'
  import { haptic } from '../../lib/telegram'

  // Дуэль 1 на 1 (SPEC): 5 минут, синонимы. Вызов ссылкой или другу напрямую. Каждый играет свои 5 минут на
  // тех же словах; итог — когда сыграли оба. Сервер проверяет каждый ответ и время.
  let { r, me }: { r: { code?: string }; me: Me } = $props()

  type Phase = 'hub' | 'view' | 'play'
  let phase = $state<Phase>('hub')
  let duel = $state<Duel | null>(null)
  let friends = $state<Friend[]>([])
  let recent = $state<Duel[]>([])
  let questions = $state<DuelQ[]>([])
  let i = $state(0)
  let score = $state(0)
  let picked = $state<number | null>(null)
  let right = $state<number | null>(null)
  let error = $state<Key | ''>('')
  let busy = $state(false)
  let now = $state(Date.now())
  let shared = $state('')

  async function hub() {
    phase = 'hub'
    duel = null
    try {
      const [f, d] = await Promise.all([api.friends(), api.duels()])
      friends = f.friends
      recent = d.duels
    } catch {
      error = 'app.error'
    }
  }

  async function open(code: string) {
    error = ''
    try {
      duel = await api.duelByCode(code)
      phase = 'view'
    } catch {
      error = 'duel.notFound'
    }
  }

  async function challenge(friend?: Friend) {
    busy = true
    error = ''
    try {
      const d = await api.newDuel(friend?.id)
      haptic.success()
      await play(d.id)
    } catch {
      error = 'app.error'
    } finally {
      busy = false
    }
  }

  async function shareLink(d: Duel) {
    const res = await share(t('duel.invite', { nick: me.nickname }), inviteLink(me.bot, `d-${d.code}`))
    shared = res === 'copied' ? t('share.copied') : ''
  }

  async function play(id: number) {
    error = ''
    try {
      const res = await api.playDuel(id)
      duel = res.duel
      questions = res.questions
      i = res.answered
      score = (res.duel.challenger.nickname === me.nickname ? res.duel.challenger : res.duel.opponent)?.score ?? 0
      if (!res.duel.ends_at || i >= questions.length) {
        duel = await api.duel(id)
        phase = 'view'
        return
      }
      phase = 'play'
    } catch (e) {
      const code = e instanceof ApiError ? e.code : ''
      error = code === 'busy' ? 'duel.busy' : code === 'expired' ? 'duel.expired' : 'app.error'
    }
  }

  async function answer(k: number) {
    if (!duel || picked !== null) return
    picked = k
    try {
      const res = await api.answerDuel(duel.id, i, k)
      right = res.answer
      score = res.score
      if (res.correct) haptic.success()
      else haptic.error()
      setTimeout(async () => {
        picked = right = null
        if (res.finished) await finish()
        else i += 1
      }, 500)
    } catch (e) {
      picked = null
      if (e instanceof ApiError && e.code === 'over') await finish()
      else error = 'app.error'
    }
  }

  async function finish() {
    if (!duel) return
    try {
      duel = await api.finishDuel(duel.id)
    } catch {
      // итог покажет повторное открытие
    }
    phase = 'view'
  }

  const tick = setInterval(() => {
    now = Date.now()
    if (phase === 'play' && duel?.ends_at && now >= new Date(duel.ends_at).getTime()) finish()
  }, 500)
  onDestroy(() => clearInterval(tick))

  onMount(() => (r.code ? open(r.code) : hub()))

  const iAmChallenger = $derived(duel?.challenger.nickname === me.nickname)
  const mine = $derived(duel ? (iAmChallenger ? duel.challenger : duel.opponent) : null)
  const rival = $derived(duel ? (iAmChallenger ? duel.opponent : duel.challenger) : null)
  const result = $derived(mine && rival ? (mine.score > rival.score ? 'duel.win' : mine.score === rival.score ? 'duel.draw' : 'duel.lose') : null)
  const w = $derived(questions[i])
</script>

<Head title={t('duel.title')} sub={t('duel.rules')} demo={false} />

{#if error}<p class="error">{t(error)}</p>{/if}

{#if phase === 'hub'}
  <button class="btn" disabled={busy} onclick={() => challenge()}>{t('duel.link')}</button>
  <p class="hint">{t('duel.linkHint')}</p>
  {#if friends.length}
    <div class="sect"><h2>{t('duel.friends')}</h2></div>
    <ul class="card rows">
      {#each friends as f (f.id)}
        <li>
          <button class="row" disabled={busy} onclick={() => challenge(f)}>
            <Avatar name={f.nickname} />
            <span class="grow">{f.nickname}</span>
            <span class="btn btn-small">{t('duel.challenge')}</span>
          </button>
        </li>
      {/each}
    </ul>
  {:else}
    <button class="card row" onclick={() => go({ s: 'friends' })}>
      <span class="grow">{t('duel.noFriends')}</span>
      <span class="chev" aria-hidden="true">›</span>
    </button>
  {/if}
  {#if recent.length}
    <div class="sect"><h2>{t('duel.recent')}</h2></div>
    <ul class="card rows">
      {#each recent as d (d.id)}
        {@const me1 = d.challenger.nickname === me.nickname}
        {@const other = me1 ? d.opponent : d.challenger}
        <li>
          <button class="row" onclick={() => open(d.code)}>
            <Avatar name={other?.nickname ?? '?'} />
            <span class="grow">
              <strong>{other?.nickname ?? t('duel.waiting')}</strong>
              <span class="tag">{t(`duel.state.${d.state}` as Key)}</span>
            </span>
            {#if d.state === 'done' && other}<span class="num">{(me1 ? d.challenger : d.opponent)?.score}:{other.score}</span>{/if}
          </button>
        </li>
      {/each}
    </ul>
  {/if}
{:else if phase === 'play' && duel && w}
  <div class="card score">
    <span><strong>{score}</strong> {t('rating.you')}</span>
    <span class="num">{duel.ends_at ? clock(new Date(duel.ends_at).getTime() - now) : ''}</span>
    <span class="hint">{i + 1}/{questions.length}</span>
  </div>
  <section class="card word">
    <p class="hint">{t('duel.synonym')}</p>
    <p class="big">{w.q}</p>
  </section>
  <div class="options">
    {#each w.options as o, k (k)}
      <button class="card" class:right={right !== null && k === right} class:wrong={picked === k && right !== null && k !== right} onclick={() => answer(k)}>{o}</button>
    {/each}
  </div>
{:else if duel}
  {#if !duel.mine && duel.state === 'open'}
    <section class="card stack center">
      <Avatar name={duel.challenger.nickname} size={56} />
      <p>{t('duel.calls', { nick: duel.challenger.nickname })}</p>
      <button class="btn" onclick={() => play(duel!.id)}>{t('duel.accept')}</button>
    </section>
  {:else if duel.state === 'expired'}
    <p class="card">{t('duel.expired')}</p>
  {:else if mine && !mine.finished}
    <button class="btn" onclick={() => play(duel!.id)}>{mine.started_at ? t('duel.continue') : t('duel.play')}</button>
  {:else if duel.state === 'done' && result && mine && rival}
    <ShareCard
      big="{mine.score} : {rival.score}"
      title={t(result as Key)}
      sub={t('duel.vs', { nick: rival.nickname })}
      text={t('share.duel', { score: `${mine.score}:${rival.score}`, nick: rival.nickname })}
      url={inviteLink(me.bot, '')}
    />
  {:else}
    <section class="card stack center">
      <p class="big">{mine?.score ?? 0}</p>
      <p>{rival?.started_at ? t('duel.rivalPlaying', { nick: rival.nickname }) : t('duel.waitRival')}</p>
    </section>
  {/if}
  {#if duel.mine && duel.state === 'open'}
    <button class="btn btn-plain" onclick={() => shareLink(duel!)}>{t('duel.share')}</button>
    {#if shared}<p class="hint">{shared}</p>{/if}
  {/if}
  <button class="btn btn-plain" onclick={hub}>{t('duel.other')}</button>
{:else if !error}
  <Skeleton rows={3} />
{/if}

<style>
  strong {
    display: block;
  }

  .score {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 8px;
  }

  .score strong {
    display: inline;
    font-size: 22px;
    font-variant-numeric: tabular-nums;
  }

  .word {
    text-align: center;
    display: grid;
    gap: 6px;
    padding: 28px 16px;
  }

  .big {
    font-size: 30px;
    font-weight: 700;
  }

  .center {
    justify-items: center;
    text-align: center;
  }

  .options {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
  }

  .options button {
    min-height: 64px;
    font-size: 17px;
    font-weight: 500;
    border: 2px solid transparent;
  }

  .right {
    border-color: #2e9d58 !important;
  }

  .wrong {
    border-color: var(--danger) !important;
  }

  .num {
    font-size: 18px;
  }
</style>
