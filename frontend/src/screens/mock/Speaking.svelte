<script lang="ts">
  import { onDestroy, onMount } from 'svelte'
  import BackBar from '../../components/BackBar.svelte'
  import Proctor from '../../components/Proctor.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { api } from '../../lib/api'
  import { clock } from '../../lib/date'
  import { t, type Key } from '../../lib/i18n/index.svelte'
  import { back } from '../../lib/nav.svelte'
  import { canRecord, startRecording, type Recorder, type Recording } from '../../lib/recorder'
  import { haptic } from '../../lib/telegram'

  // Speaking полного mock: три части, по записи на часть. Part 2 — минута подготовки по карточке.
  // Все три записи уходят одной проверкой: AI оценивает Speaking целиком, как экзаменатор.
  let { r }: { r: { id: number } } = $props()

  let questions = $state<{ part: number; text: string }[] | null>(null)
  let ends = $state(0)
  let i = $state(0)
  let takes = $state<Recording[]>([])
  let rec: Recorder | null = null
  let recording = $state(false)
  let since = $state(0)
  let prepUntil = $state(0)
  let error = $state<Key | ''>('')
  let sending = $state(false)
  let now = $state(Date.now())

  onMount(async () => {
    try {
      const res = await api.mockSpeakingStart(r.id)
      questions = res.questions
      ends = new Date(res.ends_at).getTime()
    } catch {
      error = 'app.error'
    }
  })

  const tick = setInterval(() => (now = Date.now()), 500)
  onDestroy(() => {
    clearInterval(tick)
    rec?.cancel()
  })

  async function startRec() {
    error = ''
    try {
      rec = await startRecording()
      recording = true
      since = Date.now()
      haptic.select()
    } catch {
      error = 'speaking.noMic'
    }
  }

  async function stopRec() {
    if (!rec) return
    const take = await rec.stop()
    rec = null
    recording = false
    takes[i] = take
    haptic.success()
    if (questions && i + 1 < questions.length) {
      i += 1
      prepUntil = questions[i]!.part === 2 ? Date.now() + 60_000 : 0
    }
  }

  async function send() {
    if (!questions) return
    sending = true
    error = ''
    try {
      await api.mockSpeaking(
        r.id,
        takes.map((x) => x.blob),
        takes.reduce((s, x) => s + x.seconds, 0),
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

  const q = $derived(questions?.[i])
  const done = $derived(!!questions && takes.length === questions.length && takes.every(Boolean))
</script>

<BackBar onback={back} />

{#if !canRecord()}
  <p class="card">{t('speaking.noMic')}</p>
{:else if error === 'app.error' && !questions}
  <p class="error">{t('app.error')}</p>
{:else if !questions || !q}
  <Skeleton rows={3} />
{:else}
  <h1>Speaking · {t('mock.title')}</h1>
  <div class="card bar">
    <span class="num">{clock(ends - now)}</span>
    <span class="hint">Part {q.part} · {i + 1}/{questions.length}</span>
  </div>
  <!-- Камера в Speaking только предупреждает: запись ответа не обрывается (DECISIONS 2026-11-24). -->
  {#if !done}<Proctor onviolation={() => {}} />{/if}

  {#if done}
    <section class="card stack">
      <p>{t('mock.speakingReady')}</p>
      <button class="btn" disabled={sending} onclick={send}>{t('speaking.send')}</button>
      <button class="btn btn-plain" disabled={sending} onclick={() => ((takes = []), (i = 0))}>{t('speaking.again')}</button>
    </section>
  {:else}
    <section class="card question">
      <p class="hint">{t(q.part === 2 ? 'speaking.part2' : 'speaking.part13')}</p>
      {#each q.text.split('\n') as line, k (k)}<p class="q">{line}</p>{/each}
    </section>

    <div class="mic">
      {#if prepUntil > now && !recording}
        <p class="num">{clock(prepUntil - now)}</p>
        <p class="hint">{t('speaking.prep')}</p>
      {/if}
      <button class="rec" class:on={recording} onclick={recording ? stopRec : startRec}>{recording ? '■' : '●'}</button>
      <p class="num">{recording ? clock(now - since) : '0:00'}</p>
      <p class="hint">{recording ? t('speaking.stop') : t('speaking.start')}</p>
    </div>
  {/if}
  {#if error && error !== 'app.error'}<p class="error">{t(error)}</p>{/if}
  {#if error === 'app.error' && questions}<p class="error">{t('app.error')}</p>{/if}
{/if}

<style>
  h1 {
    font-size: 24px;
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
  }

  .mic {
    display: grid;
    justify-items: center;
    gap: 6px;
    padding: 24px 0;
  }

  .rec {
    width: 84px;
    height: 84px;
    border-radius: 50%;
    background: var(--accent);
    color: var(--accent-text);
    font-size: 28px;
  }

  .rec.on {
    background: var(--danger);
    animation: pulse 1.4s ease-in-out infinite;
  }

  @keyframes pulse {
    50% {
      box-shadow: 0 0 0 14px color-mix(in srgb, var(--danger) 18%, transparent);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .rec.on {
      animation: none;
    }
  }

  .mic .num {
    font-size: 20px;
  }
</style>
