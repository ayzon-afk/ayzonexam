<script lang="ts">
  import { onDestroy, onMount } from 'svelte'
  import BackBar from '../../components/BackBar.svelte'
  import Choice from '../../components/Choice.svelte'
  import Proctor from '../../components/Proctor.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { api, ApiError, type Started } from '../../lib/api'
  import { clock } from '../../lib/date'
  import { t, type Key } from '../../lib/i18n/index.svelte'
  import { back, replace, type TestSource } from '../../lib/nav.svelte'
  import { arm, MAX_LEAVES, warn } from '../../lib/proctor'
  import { haptic } from '../../lib/telegram'

  // Задание с таймером: Reading, Listening, тест уровня — в практике, в mock и в турнире. По нулю ответы
  // сдаются сами. Уход из приложения считается (SPEC, «Честный рейтинг»). Ответы лежат в браузере:
  // закрыл приложение посреди части mock — вернулся и продолжил с теми же ответами.
  let { r }: { r: { from: TestSource } } = $props()

  // svelte-ignore state_referenced_locally
  const from = r.from
  // Часть mock — честный режим: сигнал при возврате, третий уход сдаёт часть (lib/proctor).
  const honest = 'mock' in from

  let started = $state<Started | null>(null)
  let error = $state<Key | ''>('')
  let answers = $state<(number | null)[]>([])
  let leaves = $state(0)
  let sending = $state(false)
  let now = $state(Date.now())

  const store = (id: number) => `ayzon.attempt.${id}`
  const ends = $derived(started ? new Date(started.ends_at).getTime() : 0)
  const answered = $derived(answers.filter((a) => a !== null).length)
  const test = $derived(started?.test)

  function save() {
    if (!started) return
    try {
      localStorage.setItem(store(started.attempt_id), JSON.stringify({ answers, leaves }))
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
        const saved = JSON.parse(localStorage.getItem(store(started.attempt_id)) ?? 'null') as { answers: (number | null)[]; leaves: number } | null
        if (saved && saved.answers.length === answers.length) {
          answers = saved.answers
          leaves = saved.leaves
        }
      } catch {
        // нет сохранённого — начинаем с пустых ответов
      }
    } catch (e) {
      const code = e instanceof ApiError ? e.code : ''
      error = code === 'no_tests' ? 'practice.empty' : code === 'not_live' ? 'tour.notLive' : code === 'part_done' ? 'tour.partDone' : code === 'not_joined' ? 'tour.notJoined' : 'app.error'
    }
  }

  async function submit() {
    if (!started || sending) return
    sending = true
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
      replace({ s: 'attempt', id: res.id })
    } catch (e) {
      sending = false
      if (e instanceof ApiError && e.code === 'finished') {
        replace({ s: 'attempt', id: started.attempt_id })
        return
      }
      haptic.error()
      error = 'app.error'
    }
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

  const tick = setInterval(() => {
    now = Date.now()
    if (started && !sending && now >= ends) submit()
  }, 1000)

  // Listening: запись звучит один раз, как на экзамене, — без перемотки.
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
  })

  onMount(() => {
    document.addEventListener('visibilitychange', onVisibility)
    if (honest) arm()
    start()
  })
  onDestroy(() => {
    clearInterval(tick)
    document.removeEventListener('visibilitychange', onVisibility)
    audio?.pause()
  })
</script>

<BackBar onback={back} />

{#if error}
  <p class="card">{t(error)}</p>
  {#if error === 'app.error'}<button class="btn btn-plain" onclick={start}>{t('app.retry')}</button>{/if}
{:else if !started || !test}
  <Skeleton rows={4} />
{:else}
  <header>
    <h1>{test.kind === 'level' ? t('level.title') : test.title}</h1>
    <p class="hint">{test.kind === 'reading' ? 'Reading' : test.kind === 'listening' ? 'Listening' : ''} {t('practice.sub', { n: test.questions.length, min: test.minutes })}</p>
  </header>

  <div class="card bar">
    <span class="num">{clock(ends - now)}</span>
    <span class="hint">{t('practice.answered', { n: answered, total: test.questions.length })}</span>
  </div>
  {#if honest}
    <Proctor onviolation={violation} />
    <p class="hint warn" role="status">{leaves > 0 ? t('mock.warn', { n: leaves }) : t('mock.honest')}</p>
  {/if}

  {#if test.passage}
    <article class="card passage">
      <h2>{test.title}</h2>
      {#each test.passage.split('\n\n') as p, i (i)}
        <p>{p}</p>
      {/each}
    </article>
  {/if}

  {#if test.audio}
    <div class="card player">
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
    </div>
    <p class="hint">{t('listening.once')}</p>
  {/if}

  {#each test.questions as q, i (i)}
    <Choice n={i + 1} {q} bind:value={answers[i]} />
  {/each}

  {#if !honest && leaves > 0}<p class="hint warn">{t('test.leaves', { n: leaves })}</p>{/if}
  <button class="btn" disabled={sending} onclick={submit}>{t('practice.finish')}</button>
{/if}

<style>
  header {
    display: grid;
    gap: 4px;
  }

  h1 {
    font-size: 24px;
  }

  .bar {
    position: sticky;
    top: calc(var(--safe-top) + 8px);
    z-index: 1;
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 16px;
    box-shadow: 0 2px 12px rgb(0 0 0 / 0.08);
  }

  .bar .num {
    font-size: 18px;
  }

  .passage {
    display: grid;
    gap: 10px;
    line-height: 1.55;
  }

  .passage h2 {
    font-size: 18px;
  }

  .player {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .play {
    width: 44px;
    height: 44px;
    flex: none;
    border-radius: 50%;
    background: var(--accent);
    color: var(--accent-text);
    font-size: 14px;
  }

  .play:disabled {
    opacity: 0.4;
  }

  .track {
    flex: 1;
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

  .warn {
    color: var(--danger);
  }
</style>
