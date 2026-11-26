<script lang="ts">
  import { onDestroy, onMount, tick } from 'svelte'
  import BackBar from '../../components/BackBar.svelte'
  import Proctor from '../../components/Proctor.svelte'
  import Questions from '../../components/Questions.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { api, ApiError, type Answer, type Part, type Started } from '../../lib/api'
  import { clock } from '../../lib/date'
  import { t, type Key } from '../../lib/i18n/index.svelte'
  import { back, replace, type TestSource } from '../../lib/nav.svelte'
  import { arm, MAX_LEAVES, warn } from '../../lib/proctor'
  import { haptic } from '../../lib/telegram'

  // Задание как на компьютерном IELTS (DECISIONS 2026-11-26): на весь экран, части (Listening Part 1–4,
  // Reading Passage 1–3), на ноутбуке текст слева — вопросы справа, на телефоне — «Текст / Вопросы»;
  // внизу навигатор по номерам с отметками «есть ответ» и «вернуться позже». Таймер на нуле сдаёт ответы
  // сам; уход из приложения считается (SPEC, «Честный рейтинг»). Ответы и отметки лежат в браузере:
  // закрыл приложение посреди части — вернулся и продолжил. В строгом mock после сдачи — сразу следующая часть.
  let { r }: { r: { from: TestSource } } = $props()

  // svelte-ignore state_referenced_locally
  const from = r.from
  // Часть mock — честный режим: сигнал при возврате, третий уход сдаёт часть (lib/proctor).
  const honest = 'mock' in from

  let started = $state<Started | null>(null)
  let error = $state<Key | ''>('')
  let answers = $state<(Answer | null)[]>([])
  let flags = $state<number[]>([])
  let leaves = $state(0)
  let sending = $state(false)
  let now = $state(Date.now())
  let part = $state(0)
  let current = $state(0)
  let pane = $state<'text' | 'questions'>('text')
  let confirming = $state(false)

  const store = (id: number) => `ayzon.attempt.${id}`
  const ends = $derived(started ? new Date(started.ends_at).getTime() : 0)
  const test = $derived(started?.test)
  const filled = (a: Answer | null | undefined) => a !== null && a !== undefined && a !== '' && a !== -1
  const answered = $derived(answers.filter(filled).length)
  // Задание без частей (свои, тест уровня) — одна часть с общим текстом.
  const parts = $derived<Part[]>(
    test ? (test.parts?.length ? test.parts : [{ title: '', passage: test.passage, from: 1, to: test.questions.length }]) : [],
  )
  const cur = $derived(parts[part])
  const unanswered = $derived(answers.map((a, i) => (filled(a) ? 0 : i + 1)).filter(Boolean))
  const range = (a: number, b: number) => Array.from({ length: b - a + 1 }, (_, i) => a + i)

  function save() {
    if (!started) return
    try {
      localStorage.setItem(store(started.attempt_id), JSON.stringify({ answers, leaves, flags }))
    } catch {
      // приватный режим — ответы живут только в памяти
    }
  }

  async function start() {
    error = ''
    try {
      started =
        'test' in from
          ? await api.startTest(from.test)
          : 'mock' in from
            ? await api.mockPart(from.mock, from.kind)
            : await api.startTournament(from.tour)
      answers = started.test.questions.map(() => null)
      try {
        const saved = JSON.parse(localStorage.getItem(store(started.attempt_id)) ?? 'null') as {
          answers: (Answer | null)[]
          leaves: number
          flags?: number[]
        } | null
        if (saved && saved.answers.length === answers.length) {
          answers = saved.answers
          leaves = saved.leaves
          flags = saved.flags ?? []
        }
      } catch {
        // нет сохранённого — начинаем с пустых ответов
      }
    } catch (e) {
      const code = e instanceof ApiError ? e.code : ''
      error =
        code === 'no_tests'
          ? 'practice.empty'
          : code === 'not_live'
            ? 'tour.notLive'
            : code === 'part_done'
              ? 'tour.partDone'
              : code === 'not_joined'
                ? 'tour.notJoined'
                : code === 'order'
                  ? 'mock.order'
                  : 'app.error'
    }
  }

  // Следующая часть строгого mock: Listening → Reading → Writing, без экрана с баллом.
  function nextPart() {
    if (!('mock' in from)) return
    if (from.kind === 'listening') replace({ s: 'test', from: { mock: from.mock, kind: 'reading' } })
    else replace({ s: 'mockWriting', id: from.mock })
  }

  async function submit() {
    if (!started || sending) return
    sending = true
    confirming = false
    try {
      const res = await api.submit(
        started.attempt_id,
        answers.map((a) => a ?? -1),
        leaves,
      )
      try {
        localStorage.removeItem(store(started.attempt_id))
      } catch {
        // нечего стирать
      }
      haptic.success()
      if ('hidden' in res) nextPart()
      else replace({ s: 'attempt', id: res.id })
    } catch (e) {
      sending = false
      if (e instanceof ApiError && e.code === 'finished') {
        if (honest) back()
        else replace({ s: 'attempt', id: started.attempt_id })
        return
      }
      haptic.error()
      error = 'app.error'
    }
  }

  // Переход к вопросу i (с 0): его часть, на телефоне — вопросы, прокрутка к нему.
  async function goTo(i: number) {
    if (!test || i < 0 || i >= test.questions.length) return
    const k = parts.findIndex((p) => i + 1 >= p.from && i + 1 <= p.to)
    if (k >= 0) part = k
    current = i
    pane = 'questions'
    await tick()
    document.getElementById(`q${i + 1}`)?.scrollIntoView({ block: 'start', behavior: 'smooth' })
  }

  // Текущий вопрос — тот, в котором ученик что-то нажал или пишет.
  function track(e: Event) {
    const el = (e.target as HTMLElement).closest<HTMLElement>('[data-q]')
    if (el?.dataset.q) current = Number(el.dataset.q)
  }

  function toggleFlag() {
    flags = flags.includes(current) ? flags.filter((f) => f !== current) : [...flags, current]
    haptic.select()
  }

  // Уход из приложения во время задания.
  function onVisibility() {
    if (!started || sending) return
    if (document.visibilityState === 'hidden') {
      leaves += 1
      save()
    } else if (honest && leaves > 0) {
      warn()
      if (leaves >= MAX_LEAVES) submit()
    }
  }

  // Камера увидела, что ученик не смотрит на экран, — такое же нарушение, как уход из приложения.
  function violation() {
    if (!started || sending) return
    leaves += 1
    save()
    if (leaves >= MAX_LEAVES) submit()
  }

  const timer = setInterval(() => {
    now = Date.now()
    if (started && !sending && now >= ends) submit()
  }, 1000)

  // Listening: запись звучит один раз на все части, как на экзамене, — без перемотки.
  let audio = $state<HTMLAudioElement | null>(null)
  let playing = $state(false)
  let played = $state(false)
  let pos = $state(0)
  let dur = $state(0)

  function toggle() {
    if (!audio || played) return
    if (audio.paused) {
      audio.play()
      haptic.select()
    } else audio.pause()
  }

  $effect(() => {
    save()
    void answers.length
    void flags.length
  })

  // Новая часть — с начала: текст и вопросы прокручены вверх.
  let body = $state<HTMLElement | null>(null)
  $effect(() => {
    void part
    body?.querySelectorAll('.pane').forEach((el) => (el.scrollTop = 0))
  })

  onMount(() => {
    document.addEventListener('visibilitychange', onVisibility)
    if (honest) arm()
    start()
  })
  onDestroy(() => {
    clearInterval(timer)
    document.removeEventListener('visibilitychange', onVisibility)
    audio?.pause()
  })
</script>

{#if error}
  <BackBar onback={back} />
  <p class="card">{t(error)}</p>
  {#if error === 'app.error'}<button class="btn btn-plain" onclick={start}>{t('app.retry')}</button>{/if}
{:else if !started || !test || !cur}
  <BackBar onback={back} />
  <Skeleton rows={4} />
{:else}
  <div class="exam">
    <header class="top">
      {#if !honest}<BackBar onback={back} />{/if}
      <strong class="title">{test.kind === 'level' ? t('level.title') : test.title}</strong>
      <span class="num timer" class:low={ends - now < 5 * 60_000}>{clock(ends - now)}</span>
      <span class="hint count">{t('practice.answered', { n: answered, total: test.questions.length })}</span>
    </header>

    {#if honest}
      <Proctor onviolation={violation} />
      <p class="hint warn" role="status">{leaves > 0 ? t('mock.warn', { n: leaves }) : t('mock.honest')}</p>
    {/if}

    {#if test.audio}
      <div class="player">
        <button class="play" disabled={played} aria-label={playing ? t('listening.pause') : t('listening.play')} onclick={toggle}>
          {playing ? '❚❚' : '▶'}
        </button>
        <span class="track"><span style:width="{dur ? (pos / dur) * 100 : 0}%"></span></span>
        <span class="hint">{clock(pos * 1000)} / {clock(dur * 1000)}</span>
        <audio
          bind:this={audio}
          src={test.audio}
          preload="auto"
          onplay={() => (playing = true)}
          onpause={() => (playing = false)}
          onended={() => ((playing = false), (played = true))}
          ontimeupdate={() => (pos = audio?.currentTime ?? 0)}
          onloadedmetadata={() => (dur = audio?.duration ?? 0)}
        ></audio>
        <span class="hint once">{t('listening.once')}</span>
      </div>
    {/if}

    <div class="partbar">
      <p>
        {#if cur.title}<strong>{cur.title}</strong>{/if}
        <span class="hint">{cur.intro || t('exam.range', { from: cur.from, to: cur.to })}</span>
      </p>
      {#if cur.passage}
        <div class="seg panes" role="tablist">
          <button role="tab" aria-selected={pane === 'text'} class:on={pane === 'text'} onclick={() => (pane = 'text')}>{t('exam.text')}</button>
          <button role="tab" aria-selected={pane === 'questions'} class:on={pane === 'questions'} onclick={() => (pane = 'questions')}
            >{t('exam.questions')}</button
          >
        </div>
      {/if}
    </div>

    <div class="body" class:split={!!cur.passage} bind:this={body}>
      {#if cur.passage}
        <article class="pane passage" class:off={pane !== 'text'}>
          {#each cur.passage.split('\n\n') as p, i (i)}<p>{p}</p>{/each}
        </article>
      {/if}
      <!-- svelte-ignore a11y_no_static_element_interactions, a11y_click_events_have_key_events -->
      <section class="pane qs" class:off={!!cur.passage && pane !== 'questions'} onfocusin={track} onclick={track}>
        <Questions {test} bind:answers from={cur.from - 1} to={cur.to - 1} />
        {#if !honest && leaves > 0}<p class="hint warn">{t('test.leaves', { n: leaves })}</p>{/if}
      </section>
    </div>

    <nav class="navi" aria-label={t('exam.nav')}>
      <div class="map">
        {#each parts as p, k (k)}
          <div class="pgroup" class:on={k === part}>
            {#if parts.length > 1}<button class="plabel" onclick={() => goTo(p.from - 1)}>{p.title || `${p.from}–${p.to}`}</button>{/if}
            {#each range(p.from, p.to) as n (n)}
              <button
                class="n"
                class:done={filled(answers[n - 1])}
                class:flag={flags.includes(n - 1)}
                class:cur={current === n - 1}
                aria-label="{n}{flags.includes(n - 1) ? ' · ' + t('exam.flagged') : ''}"
                onclick={() => goTo(n - 1)}>{n}</button
              >
            {/each}
          </div>
        {/each}
      </div>
      <div class="acts">
        <button class="flagbtn" class:on={flags.includes(current)} aria-pressed={flags.includes(current)} onclick={toggleFlag}>
          ⚑ {t('exam.flag')} · {current + 1}
        </button>
        <span class="grow"></span>
        <button class="arrow" aria-label={t('exam.prev')} disabled={current === 0} onclick={() => goTo(current - 1)}>‹</button>
        <button class="arrow" aria-label={t('exam.next')} disabled={current >= test.questions.length - 1} onclick={() => goTo(current + 1)}>›</button>
        <button class="btn finish" disabled={sending} onclick={() => (confirming = true)}>{t('exam.finish')}</button>
      </div>
    </nav>

    {#if confirming}
      <div class="veil">
        <div class="card sheet" role="dialog" aria-modal="true" aria-label={t('exam.confirm')}>
          <h2>{t('exam.confirm')}</h2>
          <p>{unanswered.length ? t('exam.unanswered', { list: unanswered.join(', ') }) : t('exam.all')}</p>
          {#if flags.length}<p class="hint">{t('exam.marked', { list: [...flags].sort((a, b) => a - b).map((f) => f + 1).join(', ') })}</p>{/if}
          <button class="btn" disabled={sending} onclick={submit}>{t('exam.send')}</button>
          <button class="btn btn-plain" onclick={() => (confirming = false)}>{t('exam.stay')}</button>
        </div>
      </div>
    {/if}
  </div>
{/if}

<style>
  /* Экзамен — на весь экран поверх вкладок, как компьютерный IELTS. */
  .exam {
    position: fixed;
    inset: 0;
    z-index: 30;
    display: flex;
    flex-direction: column;
    background: var(--bg);
    padding-top: var(--safe-top);
  }

  .top {
    display: flex;
    gap: 12px;
    align-items: center;
    padding: 8px 16px;
    border-bottom: 1px solid var(--separator);
    background: var(--section);
    min-width: 0;
  }

  .title {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .timer {
    font-size: 18px;
  }

  .timer.low {
    color: var(--danger);
  }

  .count {
    white-space: nowrap;
  }

  .warn {
    padding: 4px 16px 0;
    color: var(--danger);
  }

  .player {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 8px 12px;
    padding: 8px 16px;
    border-bottom: 1px solid var(--separator);
  }

  .play {
    width: 40px;
    height: 40px;
    flex: none;
    border-radius: 50%;
    background: var(--accent);
    color: var(--accent-text);
    font-size: 13px;
  }

  .play:disabled {
    opacity: 0.4;
  }

  .track {
    flex: 1;
    min-width: 80px;
    height: 6px;
    border-radius: 3px;
    background: var(--surface);
    overflow: hidden;
  }

  .track span {
    display: block;
    height: 100%;
    background: var(--accent);
  }

  .once {
    flex-basis: 100%;
  }

  .partbar {
    display: flex;
    flex-wrap: wrap;
    gap: 8px 16px;
    align-items: center;
    justify-content: space-between;
    padding: 8px 16px;
    background: var(--surface);
  }

  .partbar p {
    display: flex;
    flex-wrap: wrap;
    gap: 0 8px;
    min-width: 0;
  }

  .body {
    flex: 1;
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    min-height: 0;
  }

  .pane {
    min-height: 0;
    overflow-y: auto;
    overscroll-behavior: contain;
    padding: 12px 16px 24px;
  }

  .qs {
    display: grid;
    gap: 12px;
    align-content: start;
  }

  .passage {
    display: grid;
    gap: 10px;
    align-content: start;
    line-height: 1.6;
    overflow-wrap: anywhere;
  }

  .off {
    display: none;
  }

  .navi {
    display: grid;
    gap: 6px;
    padding: 8px 12px calc(var(--safe-bottom) + 8px);
    border-top: 1px solid var(--separator);
    background: var(--section);
  }

  .map {
    display: flex;
    gap: 12px;
    overflow-x: auto;
    padding-bottom: 2px;
  }

  .pgroup {
    display: flex;
    gap: 4px;
    align-items: center;
    flex: none;
    opacity: 0.75;
  }

  .pgroup.on {
    opacity: 1;
  }

  .plabel {
    padding: 0 6px;
    font-weight: 600;
    white-space: nowrap;
    color: var(--link);
  }

  .n {
    width: 30px;
    height: 30px;
    flex: none;
    border: 1px solid var(--separator);
    border-radius: 8px;
    font-size: 13px;
  }

  .n.done {
    background: color-mix(in srgb, var(--accent) 18%, transparent);
    border-color: var(--accent);
  }

  .n.flag {
    box-shadow: inset 0 -3px 0 #e0a100;
  }

  .n.cur {
    outline: 2px solid var(--text);
    outline-offset: 1px;
  }

  .acts {
    display: flex;
    gap: 8px;
    align-items: center;
    min-width: 0;
  }

  .grow {
    flex: 1;
  }

  .flagbtn {
    min-height: 36px;
    padding: 0 10px;
    border: 1px solid var(--separator);
    border-radius: 10px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    min-width: 0;
  }

  .flagbtn.on {
    border-color: #e0a100;
    background: color-mix(in srgb, #e0a100 16%, transparent);
  }

  .arrow {
    width: 36px;
    height: 36px;
    flex: none;
    border: 1px solid var(--separator);
    border-radius: 10px;
    font-size: 18px;
  }

  .arrow:disabled {
    opacity: 0.4;
  }

  .finish {
    width: auto;
    flex: none;
    min-height: 36px;
    padding: 0 18px;
  }

  .veil {
    position: fixed;
    inset: 0;
    z-index: 31;
    display: grid;
    place-items: center;
    padding: 16px;
    background: rgb(0 0 0 / 0.4);
  }

  .sheet {
    display: grid;
    gap: 10px;
    width: min(440px, 100%);
    overflow-wrap: anywhere;
  }

  .sheet h2 {
    font-size: 18px;
  }

  /* Ноутбук: текст слева, вопросы справа — оба видны, переключатель не нужен. */
  @media (min-width: 900px) {
    .panes {
      display: none;
    }

    .body.split {
      grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
    }

    .body.split .passage {
      border-right: 1px solid var(--separator);
    }

    .off {
      display: grid;
    }

    .pane {
      padding: 16px 24px 32px;
    }
  }
</style>
