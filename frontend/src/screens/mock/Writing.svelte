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

  // Writing полного mock: 60 минут на Task 1 (если в банке есть тема с графиком) и Task 2. Текст лежит в
  // браузере — закрытое приложение ничего не теряет. Проверка — очередью AI, лимит не тратится.
  let { r }: { r: { id: number } } = $props()

  let task1 = $state<MockPrompt | null>(null)
  let task2 = $state<MockPrompt | null>(null)
  let ends = $state(0)
  let a1 = $state('')
  let a2 = $state('')
  let error = $state<Key | ''>('')
  let sending = $state(false)
  let now = $state(Date.now())
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

  const tick = setInterval(() => (now = Date.now()), 1000)
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
      await api.mockWriting(r.id, a1, a2)
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

<BackBar onback={back} />

{#if error && !task2}
  <p class="card">{t(error)}</p>
{:else if !task2}
  <Skeleton rows={4} />
{:else}
  <h1>Writing · {t('mock.title')}</h1>
  <div class="card bar">
    <span class="num">{clock(ends - now)}</span>
    <span class="hint">{t('mock.writingTime')}</span>
  </div>
  <Proctor onviolation={violation} />
  <p class="hint warn" role="status">{frozen ? t('mock.locked') : leaves > 0 ? t('mock.warn', { n: leaves }) : t('mock.honest')}</p>

  {#if task1}
    <section class="card stack">
      <h2>Task 1</h2>
      <p>{task1.text}</p>
      {#if task1.image}<img src={task1.image} alt="" />{/if}
      <textarea bind:value={a1} readonly={frozen} placeholder={t('new.placeholder')} rows="10"></textarea>
      <p class="hint">{t('new.words', { n: w1, min: EXAM_WORDS.task1 })}</p>
    </section>
  {/if}

  <section class="card stack">
    <h2>Task 2</h2>
    <p>{task2.text}</p>
    <textarea bind:value={a2} readonly={frozen} placeholder={t('new.placeholder')} rows="14"></textarea>
    <p class="hint">{t('new.words', { n: w2, min: EXAM_WORDS.task2 })}</p>
  </section>

  {#if error}<p class="error">{t(error)}</p>{/if}
  <button class="btn" disabled={sending || w2 < 50 || (!!task1 && w1 < 50)} onclick={submit}>{t('new.submit')}</button>
  <p class="hint">{t('mock.writingHint')}</p>
{/if}

<style>
  .warn {
    color: var(--danger);
  }

  h1 {
    font-size: 24px;
  }

  h2 {
    font-size: 16px;
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

  img {
    max-width: 100%;
    border-radius: 12px;
  }

  textarea {
    min-height: 220px;
  }
</style>
