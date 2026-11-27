<script lang="ts">
  import { onDestroy, onMount } from 'svelte'
  import BackBar from '../../components/BackBar.svelte'
  import Proctor from '../../components/Proctor.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { api, type SpeakingItem } from '../../lib/api'
  import { clock } from '../../lib/date'
  import { t, type Key } from '../../lib/i18n/index.svelte'
  import { back } from '../../lib/nav.svelte'
  import { canRecord, startRecording, type Recorder, type Recording } from '../../lib/recorder'
  import { haptic } from '../../lib/telegram'

  // Speaking полного mock как на экзамене (DECISIONS 2026-11-27): Part 1 — интервью по трём темам, Part 2 —
  // карточка, минута подготовки и до 2 минут речи, Part 3 — обсуждение на тему карточки. Одна запись на часть:
  // вопросы внутри части идут по кнопке «Следующий вопрос», секунда каждого уходит AI вместе с записью.
  // Все три записи — одной проверкой: AI оценивает Speaking целиком, как экзаменатор.
  let { r }: { r: { id: number } } = $props()

  const PREP = 60_000
  const PART2_MAX = 120_000

  type Part = { part: number; items: SpeakingItem[] }

  let parts = $state<Part[] | null>(null)
  let ends = $state(0)
  let p = $state(0) // часть
  let k = $state(0) // вопрос в части
  let phase = $state<'intro' | 'prep' | 'rec'>('intro')
  let takes = $state<Recording[]>([])
  let marks = $state<number[][]>([])
  let rec: Recorder | null = null
  let starting = false // микрофон включается: таймер подготовки не должен начать вторую запись
  let since = $state(0)
  let prepUntil = $state(0)
  let notes = $state('')
  let thanks = $state(false)
  let error = $state<Key | ''>('')
  let sending = $state(false)
  let now = $state(Date.now())

  onMount(async () => {
    try {
      const res = await api.mockSpeakingStart(r.id)
      // Вопросы подряд одной части — одна запись.
      const out: Part[] = []
      for (const q of res.questions) {
        const last = out[out.length - 1]
        if (last && last.part === q.part) last.items.push(q)
        else out.push({ part: q.part, items: [q] })
      }
      parts = out
      ends = new Date(res.ends_at).getTime()
    } catch {
      error = 'app.error'
    }
  })

  const tick = setInterval(() => {
    now = Date.now()
    // Part 2: минута подготовки кончилась — запись начинается сама; на 2:00 — «Thank you», как у экзаменатора.
    if (phase === 'prep' && now >= prepUntil) record()
    if (phase === 'rec' && cur?.part === 2 && now - since >= PART2_MAX) finishPart(true)
  }, 250)
  onDestroy(() => {
    clearInterval(tick)
    rec?.cancel()
  })

  const cur = $derived(parts?.[p])
  const done = $derived(!!parts && takes.length === parts.length && takes.every(Boolean))

  function begin() {
    if (!cur) return
    error = ''
    thanks = false
    if (cur.part === 2) {
      phase = 'prep'
      prepUntil = Date.now() + PREP
      return
    }
    record()
  }

  async function record() {
    if (rec || starting) return
    starting = true
    try {
      rec = await startRecording()
      phase = 'rec'
      since = Date.now()
      k = 0
      marks[p] = [0]
      haptic.select()
    } catch {
      phase = 'intro'
      error = 'speaking.noMic'
    } finally {
      starting = false
    }
  }

  function nextQuestion() {
    if (!cur || k + 1 >= cur.items.length) return
    k += 1
    marks[p] = [...(marks[p] ?? []), Math.round((Date.now() - since) / 1000)]
    haptic.select()
  }

  async function finishPart(timeUp = false) {
    if (!rec) return
    const r0 = rec
    rec = null
    takes[p] = await r0.stop()
    thanks = timeUp
    haptic.success()
    phase = 'intro'
    if (parts && p + 1 < parts.length) {
      p += 1
      k = 0
    }
  }

  function again() {
    takes = []
    marks = []
    notes = ''
    p = 0
    k = 0
    phase = 'intro'
  }

  async function send() {
    if (!parts) return
    sending = true
    error = ''
    try {
      await api.mockSpeaking(
        r.id,
        takes.map((x) => x.blob),
        takes.reduce((s, x) => s + x.seconds, 0),
        violations,
        marks,
      )
      haptic.success()
      back()
    } catch {
      haptic.error()
      error = 'app.error'
    } finally {
      sending = false
    }
  }

  // Нарушения камеры: запись не обрывается, но с трёх Speaking не идёт в рейтинг (сервер).
  let violations = $state(0)

  const titles: Record<number, Key> = { 1: 'speaking.p1.title', 2: 'speaking.p2.title', 3: 'speaking.p3.title' }
  const leads: Record<number, Key> = { 1: 'speaking.p1.lead', 2: 'speaking.p2.lead', 3: 'speaking.p3.lead' }
  const q = $derived(cur?.items[k])
</script>

<BackBar onback={back} />

{#if !canRecord()}
  <p class="card">{t('speaking.noMic')}</p>
{:else if error === 'app.error' && !parts}
  <p class="error">{t('app.error')}</p>
{:else if !parts || !cur || !q}
  <Skeleton rows={3} />
{:else}
  <h1>Speaking · {t('mock.title')}</h1>
  <div class="card bar">
    <span class="num">{clock(ends - now)}</span>
    <span class="hint">Part {cur.part} · {p + 1}/{parts.length}</span>
  </div>
  <!-- Камера в Speaking предупреждает без звука: запись ответа не обрывается и сигнал в неё не попадает
       (DECISIONS 2026-11-24); нарушения считаются и уходят на сервер (DECISIONS 2026-11-25). -->
  {#if !done}<Proctor sound={false} onviolation={() => (violations += 1)} />{/if}
  {#if violations > 0}<p class="hint warn" role="status">{t('mock.warn', { n: violations })}</p>{/if}
  {#if thanks}<p class="card hint" role="status">{t('speaking.thanks')}</p>{/if}

  {#if done}
    <section class="card stack">
      <p>{t('mock.speakingReady')}</p>
      <button class="btn" disabled={sending} onclick={send}>{t('speaking.send')}</button>
      <button class="btn btn-plain" disabled={sending} onclick={again}>{t('speaking.redo')}</button>
    </section>
  {:else if phase === 'intro'}
    <section class="card stack">
      <h2>{t(titles[cur.part] ?? 'speaking.p1.title')}</h2>
      <p class="hint">{t(leads[cur.part] ?? 'speaking.p1.lead')}</p>
      <button class="btn" onclick={begin}>{t(cur.part === 2 ? 'speaking.showCard' : 'speaking.begin')}</button>
    </section>
  {:else if cur.part === 2}
    <section class="card question">
      {#each q.text.split('\n') as line, i (i)}<p class:q={i === 0}>{line}</p>{/each}
    </section>
    <textarea class="notes" bind:value={notes} rows="4" placeholder={t('speaking.notes')}></textarea>
    <div class="mic">
      {#if phase === 'prep'}
        <p class="num">{clock(prepUntil - now)}</p>
        <p class="hint">{t('speaking.prep')}</p>
        <button class="btn" onclick={record}>{t('speaking.speakNow')}</button>
      {:else}
        <span class="dot" aria-hidden="true"></span>
        <p class="num">{clock(now - since)} / 2:00</p>
        <button class="btn" onclick={() => finishPart()}>{t('speaking.endPart')}</button>
      {/if}
    </div>
  {:else}
    <section class="card question">
      {#if cur.part === 1 && q.topic}<p class="hint">{t('speaking.topic', { t: q.topic })}</p>{/if}
      <p class="hint">{t('speaking.qOf', { n: k + 1, total: cur.items.length })}</p>
      <p class="q">{q.text}</p>
    </section>
    <div class="mic">
      <span class="dot" aria-hidden="true"></span>
      <p class="num">{clock(now - since)}</p>
      {#if k + 1 < cur.items.length}
        <button class="btn" onclick={nextQuestion}>{t('speaking.nextQ')}</button>
      {:else}
        <button class="btn" onclick={() => finishPart()}>{t('speaking.endPart')}</button>
      {/if}
    </div>
  {/if}
  {#if error && error !== 'app.error'}<p class="error">{t(error)}</p>{/if}
  {#if error === 'app.error' && parts}<p class="error">{t('app.error')}</p>{/if}
{/if}

<style>
  h1 {
    font-size: 24px;
  }

  h2 {
    font-size: 18px;
  }

  .bar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 16px;
  }

  .bar .num {
    font-size: 18px;
  }

  .question {
    display: grid;
    gap: 6px;
  }

  .q {
    font-size: 18px;
    line-height: 1.45;
    font-weight: 500;
  }

  .notes {
    width: 100%;
    min-width: 0;
  }

  .mic {
    display: grid;
    justify-items: center;
    gap: 8px;
    padding: 16px 0;
  }

  .mic .btn {
    width: 100%;
  }

  .mic .num {
    font-size: 20px;
  }

  .dot {
    width: 18px;
    height: 18px;
    border-radius: 50%;
    background: var(--danger);
    animation: pulse 1.4s ease-in-out infinite;
  }

  @keyframes pulse {
    50% {
      box-shadow: 0 0 0 10px color-mix(in srgb, var(--danger) 18%, transparent);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .dot {
      animation: none;
    }
  }
</style>
