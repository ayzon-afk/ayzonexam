<script lang="ts">
  import { onDestroy, onMount } from 'svelte'
  import BackBar from '../../components/BackBar.svelte'
  import Proctor from '../../components/Proctor.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { api, ApiError, type MockPrompt } from '../../lib/api'
  import { clock } from '../../lib/date'
  import { t, type Key } from '../../lib/i18n/index.svelte'
  import { back } from '../../lib/nav.svelte'
  import { arm, MAX_LEAVES, warn } from '../../lib/proctor'
  import { haptic } from '../../lib/telegram'
  import { countWords, EXAM_WORDS } from '../../lib/writing'

  // Writing полного mock: 60 минут на Task 1 (если в банке есть тема с графиком) и Task 2 — как на компьютерном
  // IELTS (DECISIONS 2026-11-26): на весь экран, вкладки Task 1 / Task 2 внизу, на ноутбуке задание слева —
  // ответ справа; на нуле таймера работа сдаётся сама. Текст лежит в браузере — закрытое приложение ничего
  // не теряет. Проверка — очередью AI, лимит не тратится.
  let { r }: { r: { id: number } } = $props()

  let task1 = $state<MockPrompt | null>(null)
  let task2 = $state<MockPrompt | null>(null)
  let ends = $state(0)
  let a1 = $state('')
  let a2 = $state('')
  let error = $state<Key | ''>('')
  let sending = $state(false)
  let now = $state(Date.now())
  let task = $state<1 | 2>(1)
  // Честный режим (lib/proctor): уходы из приложения; с третьего текст закрыт и работа отправляется.
  let leaves = $state(0)
  const locked = $derived(leaves >= MAX_LEAVES)
  // Третье нарушение отправляет работу один раз. Сервер ответил «слишком коротко» — текст снова открыт: иначе
  // ученик застрял бы, не может ни дописать, ни сдать.
  let autoSent = false
  const frozen = $derived(locked && error !== 'mock.tooShort')
  function autoSubmit() {
    if (locked && !autoSent) {
      autoSent = true
      submit()
    }
  }

  // svelte-ignore state_referenced_locally
  const KEY = `ayzon.mock.writing.${r.id}`

  onMount(async () => {
    try {
      const res = await api.mockWritingStart(r.id)
      task1 = res.task1
      task2 = res.task2
      if (!task1) task = 2
      ends = new Date(res.ends_at).getTime()
      const saved = JSON.parse(localStorage.getItem(KEY) ?? 'null') as { a1: string; a2: string; leaves?: number } | null
      if (saved) ({ a1, a2 } = saved)
      leaves = saved?.leaves ?? 0
    } catch (e) {
      error = e instanceof ApiError && e.code === 'no_tests' ? 'practice.empty' : 'app.error'
    }
  })

  $effect(() => {
    const v = JSON.stringify({ a1, a2, leaves })
    try {
      localStorage.setItem(KEY, v)
    } catch {
      // приватный режим — текст только в памяти
    }
  })

  function onVisibility() {
    if (!task2 || sending) return
    if (document.visibilityState === 'hidden') leaves += 1
    else if (leaves > 0) {
      warn()
      autoSubmit()
    }
  }
  document.addEventListener('visibilitychange', onVisibility)

  // Камера: не на экран — нарушение, как уход из приложения.
  function violation() {
    if (!task2 || sending) return
    leaves += 1
    autoSubmit()
  }
  arm()

  // Время вышло — работа уходит сама, как на экзамене (один раз; «слишком коротко» — остаётся на экране).
  let timeSent = false
  const tick = setInterval(() => {
    now = Date.now()
    if (task2 && ends && now >= ends && !sending && !timeSent) {
      timeSent = true
      submit()
    }
  }, 1000)
  onDestroy(() => {
    clearInterval(tick)
    document.removeEventListener('visibilitychange', onVisibility)
  })

  const w1 = $derived(countWords(a1))
  const w2 = $derived(countWords(a2))

  async function submit() {
    sending = true
    error = ''
    try {
      await api.mockWriting(r.id, a1, a2, leaves)
      try {
        localStorage.removeItem(KEY)
      } catch {
        // нечего стирать
      }
      haptic.success()
      back()
    } catch (e) {
      haptic.error()
      const code = e instanceof ApiError ? e.code : ''
      error = code === 'too_short' ? 'mock.tooShort' : code === 'too_long' ? 'new.tooLong.short' : code === 'part_done' ? 'mock.partDone' : 'app.error'
    } finally {
      sending = false
    }
  }
</script>

{#if error && !task2}
  <BackBar onback={back} />
  <p class="card">{t(error)}</p>
{:else if !task2}
  <BackBar onback={back} />
  <Skeleton rows={4} />
{:else}
  {@const prompt = task === 1 && task1 ? task1 : task2}
  <div class="exam">
    <header class="top">
      <strong class="title">Writing · {t('mock.title')}</strong>
      <span class="num timer" class:low={ends - now < 5 * 60_000}>{clock(ends - now)}</span>
    </header>
    <Proctor onviolation={violation} />
    <p class="hint warn" role="status">{frozen ? t('mock.locked') : leaves > 0 ? t('mock.warn', { n: leaves }) : t('mock.honest')}</p>

    <div class="body">
      <section class="pane prompt">
        <h2>Task {task1 ? task : 2}</h2>
        <p>{prompt.text}</p>
        {#if prompt.image}<img src={prompt.image} alt="" />{/if}
        <p class="hint">{t('mock.writingTime')}</p>
      </section>
      <section class="pane answer">
        {#if task === 1 && task1}
          <textarea bind:value={a1} readonly={frozen} placeholder={t('new.placeholder')} spellcheck="false" autocomplete="off" autocapitalize="off"
          ></textarea>
          <p class="hint">{t('new.words', { n: w1, min: EXAM_WORDS.task1 })}</p>
        {:else}
          <textarea bind:value={a2} readonly={frozen} placeholder={t('new.placeholder')} spellcheck="false" autocomplete="off" autocapitalize="off"
          ></textarea>
          <p class="hint">{t('new.words', { n: w2, min: EXAM_WORDS.task2 })}</p>
        {/if}
        {#if error}<p class="error">{t(error)}</p>{/if}
      </section>
    </div>

    <nav class="navi">
      {#if task1}
        <div class="seg tasks" role="tablist">
          <button role="tab" aria-selected={task === 1} class:on={task === 1} onclick={() => (task = 1)}>Task 1 · {w1}</button>
          <button role="tab" aria-selected={task === 2} class:on={task === 2} onclick={() => (task = 2)}>Task 2 · {w2}</button>
        </div>
      {/if}
      <span class="grow"></span>
      <button class="btn finish" disabled={sending || w2 < 50 || (!!task1 && w1 < 50)} onclick={submit}>{t('new.submit')}</button>
    </nav>
  </div>
{/if}

<style>
  /* На весь экран поверх вкладок, как компьютерный IELTS (как экран задания exam/Test.svelte). */
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

  .timer.low,
  .warn {
    color: var(--danger);
  }

  .warn {
    padding: 4px 16px 0;
  }

  .body {
    flex: 1;
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    grid-auto-rows: max-content;
    min-height: 0;
    overflow-y: auto;
  }

  .pane {
    display: grid;
    gap: 10px;
    align-content: start;
    padding: 12px 16px;
    overflow-wrap: anywhere;
  }

  h2 {
    font-size: 16px;
  }

  img {
    max-width: 100%;
    border-radius: 12px;
  }

  textarea {
    min-height: 45vh;
  }

  .navi {
    display: flex;
    gap: 8px;
    align-items: center;
    padding: 8px 12px calc(var(--safe-bottom) + 8px);
    border-top: 1px solid var(--separator);
    background: var(--section);
    min-width: 0;
  }

  .tasks {
    min-width: 0;
  }

  .grow {
    flex: 1;
  }

  .finish {
    width: auto;
    flex: none;
    min-height: 36px;
    padding: 0 18px;
  }

  /* Ноутбук: задание слева, ответ справа, каждая колонка со своей прокруткой. */
  @media (min-width: 900px) {
    .body {
      grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
      grid-auto-rows: auto;
      overflow: hidden;
    }

    .pane {
      min-height: 0;
      overflow-y: auto;
      padding: 16px 24px;
    }

    .prompt {
      border-right: 1px solid var(--separator);
    }

    .answer {
      grid-template-rows: minmax(0, 1fr) auto auto;
    }

    textarea {
      min-height: 0;
      height: 100%;
      resize: none;
    }
  }
</style>
