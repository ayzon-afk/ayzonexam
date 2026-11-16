<script lang="ts">
  import { onDestroy, onMount } from 'svelte'
  import BackBar from '../../components/BackBar.svelte'
  import { api, ApiError, type Prompt, type Task } from '../../lib/api'
  import { t, type Key } from '../../lib/i18n/index.svelte'
  import { compressPhoto } from '../../lib/photo'
  import { haptic } from '../../lib/telegram'
  import { clearDraft, countWords, EXAM_WORDS, loadDraft, MAX_WORDS, MIN_WORDS, saveDraft } from '../../lib/writing'

  let { onback, oncreated }: { onback: () => void; oncreated: (id: number) => void } = $props()

  type Step = 'task' | 'prompt' | 'write'

  let step = $state<Step>('task')
  let task = $state<Task>('task2')
  let prompts = $state<Prompt[]>([])
  let promptIndex = $state(0)
  let own = $state(false)
  let ownPrompt = $state('')
  let chart = $state<Blob | null>(null)
  let chartURL = $state('')
  let essay = $state('')
  let source = $state<'text' | 'photo'>('text')
  let busy = $state<'' | 'chart' | 'recognize' | 'send'>('')
  let recognized = $state(false)
  let error = $state('')

  const bankPrompt = $derived(prompts.length ? prompts[promptIndex % prompts.length] : null)
  const words = $derived(countWords(essay))
  const tooLong = $derived(words > MAX_WORDS[task])
  const promptReady = $derived(task === 'task1' ? chart !== null : own ? ownPrompt.trim().length >= 20 : bankPrompt !== null)

  // Черновик сохраняется на каждое изменение, как только выбрана задача.
  $effect(() => {
    if (step === 'task') return
    saveDraft({ task, promptId: own || task === 'task1' ? null : (bankPrompt?.id ?? null), prompt: ownPrompt, essay, source })
  })

  onMount(async () => {
    const d = loadDraft()
    if (d) {
      task = d.task
      essay = d.essay
      source = d.source
      ownPrompt = d.prompt
      own = d.task === 'task2' && d.promptId === null && d.prompt !== ''
      // Фото графика в черновике нет — Task 1 начинается с него заново.
      step = d.task === 'task1' ? 'prompt' : 'write'
    }
    try {
      prompts = (await api.prompts('task2')).prompts
      const i = d?.promptId ? prompts.findIndex((p) => p.id === d.promptId) : -1
      promptIndex = i >= 0 ? i : Math.floor(Math.random() * Math.max(prompts.length, 1))
    } catch {
      // Без банка тем остаётся «своя тема».
      own = true
    }
  })

  onDestroy(() => {
    if (chartURL) URL.revokeObjectURL(chartURL)
  })

  function back() {
    error = ''
    if (step === 'write') step = 'prompt'
    else if (step === 'prompt') step = 'task'
    else onback()
  }

  function choose(next: Task) {
    task = next
    step = 'prompt'
    haptic.select()
  }

  function show(code: string) {
    const known: Record<string, Key> = {
      no_text: 'err.no_text',
      daily_limit: 'err.daily_limit',
      ai_unavailable: 'err.ai_unavailable',
      bad_image: 'err.image',
      image_too_large: 'err.image',
      photo_too_large: 'err.image',
      bad_prompt: 'err.prompt',
    }
    error = t(known[code] ?? 'app.error')
    haptic.error()
  }

  function code(e: unknown): string {
    return e instanceof ApiError ? e.code : e instanceof Error ? e.message : ''
  }

  async function pickChart(e: Event) {
    const input = e.currentTarget as HTMLInputElement
    const file = input.files?.[0]
    input.value = ''
    if (!file) return
    busy = 'chart'
    error = ''
    try {
      chart = await compressPhoto(file)
      if (chartURL) URL.revokeObjectURL(chartURL)
      chartURL = URL.createObjectURL(chart)
    } catch (err) {
      show(code(err) || 'bad_image')
    } finally {
      busy = ''
    }
  }

  // Распознанный текст добавляется в конец: можно снимать страницы по одной, написанное не теряется.
  async function pickPages(e: Event) {
    const input = e.currentTarget as HTMLInputElement
    const files = [...(input.files ?? [])].slice(0, 3)
    input.value = ''
    if (!files.length) return
    busy = 'recognize'
    error = ''
    try {
      const pages = await Promise.all(files.map((f) => compressPhoto(f)))
      const { text } = await api.recognize(pages)
      essay = essay.trim() ? `${essay.trim()}\n\n${text}` : text
      source = 'photo'
      recognized = true
      haptic.success()
    } catch (err) {
      show(code(err) || 'bad_image')
    } finally {
      busy = ''
    }
  }

  async function submit() {
    busy = 'send'
    error = ''
    try {
      const { id } = await api.createCheck({
        task,
        source,
        essay,
        promptId: task === 'task2' && !own ? bankPrompt?.id : undefined,
        prompt: task === 'task2' && own ? ownPrompt.trim() : undefined,
        image: task === 'task1' ? (chart ?? undefined) : undefined,
      })
      clearDraft()
      haptic.success()
      oncreated(id)
    } catch (err) {
      show(code(err))
    } finally {
      busy = ''
    }
  }
</script>

<BackBar onback={back} />

{#if step === 'task'}
  <h1>{t('new.chooseTask')}</h1>
  {#each ['task2', 'task1'] as const as k (k)}
    <button class="card choice" onclick={() => choose(k)}>
      <span class="name">{t(k === 'task1' ? 'task.task1' : 'task.task2')}</span>
      <span class="hint">{t(k === 'task1' ? 'task.task1.hint' : 'task.task2.hint')}</span>
    </button>
  {/each}
{:else if step === 'prompt'}
  <h1>{t(task === 'task1' ? 'task.task1' : 'task.task2')}</h1>
  {#if task === 'task1'}
    <section class="card stack">
      <h2>{t('new.chart')}</h2>
      <p class="hint">{t('new.chartHint')}</p>
      {#if chartURL}<img class="chart" src={chartURL} alt={t('new.chart')} />{/if}
      <label class="btn btn-plain file" class:disabled={busy !== ''}>
        {chart ? t('new.retakePhoto') : t('new.takePhoto')}
        <input type="file" accept="image/*" disabled={busy !== ''} onchange={pickChart} />
      </label>
    </section>
  {:else}
    <div class="tabs" role="radiogroup">
      <button role="radio" aria-checked={!own} class:on={!own} disabled={!prompts.length} onclick={() => (own = false)}
        >{t('new.bankPrompt')}</button
      >
      <button role="radio" aria-checked={own} class:on={own} onclick={() => (own = true)}>{t('new.ownPrompt')}</button>
    </div>
    {#if own}
      <section class="card stack">
        <p class="hint">{t('new.ownPromptHint')}</p>
        <textarea rows="5" maxlength="1000" bind:value={ownPrompt}></textarea>
      </section>
    {:else if bankPrompt}
      <section class="card stack">
        <p class="prompt">{bankPrompt.text}</p>
        <button class="link" onclick={() => (promptIndex += 1)}>↻ {t('new.otherPrompt')}</button>
      </section>
    {/if}
  {/if}
  {#if error}<p class="error">{error}</p>{/if}
  <button class="btn" disabled={!promptReady || busy !== ''} onclick={() => (step = 'write')}>{t('onb.next')}</button>
{:else}
  <details class="card task">
    <summary>{t('result.task')}: {t(task === 'task1' ? 'task.task1' : 'task.task2')}</summary>
    {#if task === 'task1'}
      {#if chartURL}<img class="chart" src={chartURL} alt={t('new.chart')} />{/if}
    {:else}
      <p class="prompt">{own ? ownPrompt : bankPrompt?.text}</p>
    {/if}
  </details>

  <section class="stack">
    <h2>{t('new.answer')}</h2>
    <textarea
      class="essay"
      rows="14"
      placeholder={t('new.placeholder')}
      autocapitalize="sentences"
      spellcheck="false"
      bind:value={essay}
    ></textarea>
    <p class="hint" class:error={tooLong}>
      {tooLong ? t('new.tooLong', { max: MAX_WORDS[task] }) : t('new.words', { n: words, min: EXAM_WORDS[task] })}
    </p>
    {#if recognized}<p class="note">{t('new.recognized')}</p>{/if}
  </section>

  <section class="card stack">
    <label class="btn btn-plain file" class:disabled={busy !== ''}>
      {busy === 'recognize' ? t('new.recognizing') : t('new.handwriting')}
      <input type="file" accept="image/*" multiple disabled={busy !== ''} onchange={pickPages} />
    </label>
    <p class="hint">{t('new.handwritingHint')}</p>
  </section>

  {#if error}<p class="error">{error}</p>{/if}
  <button class="btn" disabled={words < MIN_WORDS || tooLong || busy !== ''} onclick={submit}>{t('new.submit')}</button>
{/if}

<style>
  h1 {
    font-size: 24px;
  }

  h2 {
    font-size: 16px;
    font-weight: 600;
  }

  .stack {
    display: grid;
    gap: 10px;
  }

  .choice {
    display: grid;
    gap: 4px;
    text-align: left;
  }

  .name {
    font-size: 18px;
    font-weight: 600;
  }

  .tabs {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px;
  }

  .tabs button {
    min-height: 44px;
    border-radius: 12px;
    background: var(--section);
    font-weight: 500;
  }

  .tabs .on {
    background: var(--accent);
    color: var(--accent-text);
  }

  .prompt {
    white-space: pre-wrap;
  }

  .link {
    justify-self: start;
    color: var(--link);
  }

  textarea {
    width: 100%;
    padding: 12px;
    border: 1px solid var(--separator);
    border-radius: var(--radius);
    background: var(--section);
    color: var(--text);
    font: inherit;
    resize: vertical;
  }

  .essay {
    min-height: 40vh;
    line-height: 1.5;
  }

  .chart {
    width: 100%;
    max-height: 50vh;
    object-fit: contain;
    border-radius: 10px;
  }

  .file {
    display: grid;
    place-items: center;
    cursor: pointer;
  }

  .file input {
    display: none;
  }

  .file.disabled {
    opacity: 0.4;
  }

  .task summary {
    font-weight: 500;
    cursor: pointer;
  }

  .task[open] summary {
    margin-bottom: 10px;
  }

  .note {
    font-size: 14px;
    color: var(--link);
  }
</style>
