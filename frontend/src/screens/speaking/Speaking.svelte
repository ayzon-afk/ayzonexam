<script lang="ts">
  import { onDestroy, onMount } from 'svelte'
  import Head from '../../components/Head.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { api, ApiError, type Quota, type SpeakingCheck, type SpeakingQ } from '../../lib/api'
  import { clock, fmtDate } from '../../lib/date'
  import { formatBand } from '../../lib/exam'
  import { t, type Key } from '../../lib/i18n/index.svelte'
  import { go, replace } from '../../lib/nav.svelte'
  import { canRecord, startRecording, type Recorder, type Recording } from '../../lib/recorder'
  import { haptic } from '../../lib/telegram'

  // Speaking-практика: вопрос одной части → запись → AI ставит балл по 4 критериям. Микрофон не работает —
  // вопрос приходит в бот, ученик отвечает голосовым (SPEC, запасной путь).
  let questions = $state<Record<string, SpeakingQ>>({})
  let history = $state<SpeakingCheck[]>([])
  let quota = $state<Quota | null>(null)
  let part = $state(1)
  let failed = $state(false)
  let rec: Recorder | null = null
  let recording = $state(false)
  let take = $state<Recording | null>(null)
  let since = $state(0)
  let now = $state(Date.now())
  let sending = $state(false)
  let note = $state<Key | ''>('')

  const q = $derived(questions[String(part)])
  const left = $derived(quota ? Math.max(quota.limit - quota.used, 0) : 0)
  const preview = $derived(take ? URL.createObjectURL(take.blob) : '')

  async function load() {
    failed = false
    try {
      const res = await api.speaking()
      questions = res.questions
      history = res.history
      quota = res.quota
    } catch {
      failed = true
    }
  }

  async function startRec() {
    note = ''
    take = null
    try {
      rec = await startRecording()
      recording = true
      since = Date.now()
      haptic.select()
    } catch {
      note = 'speaking.noMic'
    }
  }

  async function stopRec() {
    if (!rec) return
    take = await rec.stop()
    rec = null
    recording = false
  }

  async function send() {
    if (!take || !q) return
    sending = true
    note = ''
    try {
      const { id } = await api.speak(q.id, take.blob, take.seconds)
      haptic.success()
      replace({ s: 'speakingCheck', id })
    } catch (e) {
      haptic.error()
      note = e instanceof ApiError && e.code === 'daily_limit' ? 'speaking.limit' : e instanceof ApiError && e.code === 'bad_audio' ? 'speaking.badAudio' : 'app.error'
    } finally {
      sending = false
    }
  }

  async function viaBot() {
    if (!q) return
    try {
      await api.speakViaBot(q.id)
      note = 'speaking.botSent'
      haptic.success()
    } catch {
      note = 'app.error'
    }
  }

  // Новый вопрос части — просто перечитать: сервер даёт случайный.
  async function another() {
    take = null
    await load()
  }

  const tick = setInterval(() => (now = Date.now()), 500)
  onDestroy(() => {
    clearInterval(tick)
    rec?.cancel()
  })
  onMount(load)
</script>

<Head title="Speaking" sub={t('speaking.lead')} demo={false} />

{#if failed}
  <p class="error">{t('app.error')}</p>
  <button class="btn btn-plain" onclick={load}>{t('app.retry')}</button>
{:else if !quota}
  <Skeleton rows={4} />
{:else}
  <div class="seg" role="tablist">
    {#each [1, 2, 3] as p (p)}
      <button role="tab" aria-selected={part === p} class:on={part === p} disabled={recording} onclick={() => ((part = p), (take = null))}>Part {p}</button>
    {/each}
  </div>

  {#if q}
    <section class="card question">
      <p class="hint">{t(part === 2 ? 'speaking.part2' : 'speaking.part13')}</p>
      {#each q.text.split('\n') as line, k (k)}<p class="q">{line}</p>{/each}
      <button class="more" disabled={recording} onclick={another}>{t('speaking.another')}</button>
    </section>

    <p class="hint">
      {left > 0 ? t(quota.period === 'day' ? 'quota.day' : 'quota.week', { n: left, limit: quota.limit }) : t('speaking.limit')}
      {#if quota.bonus > 0}· {t('quota.bonus', { n: quota.bonus })}{/if}
    </p>

    {#if canRecord()}
      <div class="mic">
        <button class="rec" class:on={recording} onclick={recording ? stopRec : startRec}>{recording ? '■' : '●'}</button>
        <p class="num">{recording ? clock(now - since) : take ? clock(take.seconds * 1000) : '0:00'}</p>
        <p class="hint">{recording ? t('speaking.stop') : t('speaking.start')}</p>
      </div>
      {#if take && !recording}
        <audio controls src={preview}></audio>
        <button class="btn" disabled={sending || (left === 0 && quota.bonus === 0)} onclick={send}>{t('speaking.send')}</button>
      {/if}
    {:else}
      <p class="card">{t('speaking.noMic')}</p>
    {/if}
    <button class="btn btn-plain" onclick={viaBot}>{t('speaking.viaBot')}</button>
    {#if note}<p class={note === 'speaking.botSent' ? 'hint' : 'error'}>{t(note)}</p>{/if}
    {#if note === 'speaking.limit'}<button class="btn btn-plain" onclick={() => go({ s: 'premium' })}>{t('premium.go')}</button>{/if}
  {/if}

  {#if history.length}
    <div class="sect"><h2>{t('speaking.history')}</h2></div>
    <ul class="card rows">
      {#each history as c (c.id)}
        <li>
          <button class="row" onclick={() => go({ s: 'speakingCheck', id: c.id })}>
            <span class="grow">
              <strong>{c.mock_id ? t('mock.title') : c.questions[0]?.part ? `Part ${c.questions[0].part}` : t('history.voice')}</strong>
              <span class="tag">{fmtDate(c.created_at)}</span>
            </span>
            {#if c.band !== null}<span class="num">{formatBand(c.band)}</span>{:else if c.status === 'failed'}<span class="tag error">{t('writing.status.failed')}</span>{:else}<span class="tag">{t('writing.status.pending')}</span>{/if}
          </button>
        </li>
      {/each}
    </ul>
  {/if}
{/if}

<style>
  .question {
    display: grid;
    gap: 6px;
  }

  .q {
    font-size: 18px;
    line-height: 1.45;
  }

  .question .more {
    justify-self: start;
  }

  .mic {
    display: grid;
    justify-items: center;
    gap: 6px;
    padding: 16px 0;
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

  audio {
    width: 100%;
  }

  strong {
    display: block;
    font-weight: 600;
  }

  .num {
    font-size: 20px;
  }
</style>
