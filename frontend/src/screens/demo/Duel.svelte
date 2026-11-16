<script lang="ts">
  import { onDestroy } from 'svelte'
  import Avatar from '../../components/Avatar.svelte'
  import Head from '../../components/Head.svelte'
  import ShareCard from '../../components/ShareCard.svelte'
  import { clock } from '../../lib/date'
  import { friends } from '../../lib/demo'
  import { duelWords } from '../../lib/demo-exam'
  import { t } from '../../lib/i18n/index.svelte'
  import { haptic } from '../../lib/telegram'

  // Дуэль 1 на 1: 5 минут, синонимы. Соперник в демо — придуманный, отвечает сам раз в несколько секунд.
  type Phase = 'pick' | 'play' | 'done'
  let phase = $state<Phase>('pick')
  let rival = $state('')
  let i = $state(0)
  let mine = $state(0)
  let theirs = $state(0)
  let picked = $state<number | null>(null)
  let ends = $state(0)
  let now = $state(Date.now())
  let linkCopied = $state(false)

  let timer: ReturnType<typeof setInterval> | undefined
  let next: ReturnType<typeof setTimeout> | undefined

  function start(nick: string) {
    rival = nick
    phase = 'play'
    i = mine = theirs = 0
    picked = null
    ends = Date.now() + 5 * 60_000
    timer = setInterval(() => {
      now = Date.now()
      if (Math.random() < 0.3 && theirs < duelWords.length) theirs += 1
      if (now >= ends) end()
    }, 1000)
  }

  function answer(k: number) {
    if (picked !== null) return
    picked = k
    const ok = k === duelWords[i]!.answer
    if (ok) mine += 1
    if (ok) haptic.success()
    else haptic.error()
    next = setTimeout(() => {
      picked = null
      if (i + 1 < duelWords.length) i += 1
      else end()
    }, 600)
  }

  function end() {
    clearInterval(timer)
    phase = 'done'
  }

  onDestroy(() => {
    clearInterval(timer)
    clearTimeout(next)
  })

  const w = $derived(duelWords[i]!)
</script>

<Head title={t('duel.title')} sub={t('duel.rules')} />

{#if phase === 'pick'}
  <button class="btn" onclick={() => (linkCopied = true)}>{t('duel.link')}</button>
  {#if linkCopied}<p class="hint">{t('demo.later')}</p>{/if}
  <div class="sect"><h2>{t('duel.friends')}</h2></div>
  <ul class="card rows">
    {#each friends as f (f)}
      <li>
        <button class="row" onclick={() => start(f)}>
          <Avatar name={f} />
          <span class="grow">{f}</span>
          <span class="btn btn-small">{t('duel.challenge')}</span>
        </button>
      </li>
    {/each}
  </ul>
{:else if phase === 'play'}
  <div class="card score">
    <span><strong>{mine}</strong> {t('rating.you')}</span>
    <span class="num">{clock(ends - now)}</span>
    <span>{rival} <strong>{theirs}</strong></span>
  </div>
  <section class="card word">
    <p class="hint">{t('duel.synonym')} · {i + 1}/{duelWords.length}</p>
    <p class="big">{w.q}</p>
  </section>
  <div class="options">
    {#each w.options as o, k (k)}
      <button
        class="card"
        class:right={picked !== null && k === w.answer}
        class:wrong={picked === k && k !== w.answer}
        onclick={() => answer(k)}>{o}</button
      >
    {/each}
  </div>
{:else}
  <ShareCard
    big="{mine} : {theirs}"
    title={mine > theirs ? t('duel.win') : mine === theirs ? t('duel.draw') : t('duel.lose')}
    sub={t('duel.vs', { nick: rival })}
    text={t('share.duel', { score: `${mine}:${theirs}`, nick: rival })}
  />
  <button class="btn" onclick={() => start(rival)}>{t('duel.rematch')}</button>
  <button class="btn btn-plain" onclick={() => (phase = 'pick')}>{t('duel.other')}</button>
{/if}

<style>
  .score {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 8px;
  }

  .score strong {
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
</style>
