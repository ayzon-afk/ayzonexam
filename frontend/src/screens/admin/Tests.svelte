<script lang="ts">
  import { onMount } from 'svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { authHeader, request } from '../../lib/api'
  import { compressPhoto } from '../../lib/photo'
  import { adm, type Row } from './admin'

  // Задания: Reading, Listening, тест уровня; аудио Listening — запись владельца или озвучка сценария
  // (Gemini, сжимается здесь в браузере); темы Writing (Task 1 — с картинкой графика для mock); вопросы
  // Speaking. Свои задания, не из книг Cambridge (SPEC).
  type View = 'tests' | 'prompts' | 'questions'
  let view = $state<View>('tests')
  let tests = $state<Row[] | null>(null)
  let prompts = $state<Row[] | null>(null)
  let questions = $state<Row[] | null>(null)
  let edit = $state<Row | null>(null)
  let qjson = $state('')
  let note = $state('')
  let busy = $state('')

  const kinds: Record<string, string> = { reading: 'Reading', listening: 'Listening', level: 'Тест уровня' }
  const sample = `[\n  {"q": "Вопрос", "options": ["A", "B", "C"], "answer": 0, "type": "detail"}\n]`

  async function load() {
    ;[tests, prompts, questions] = await Promise.all([
      adm.get<{ tests: Row[] }>('tests').then((r) => r.tests),
      adm.get<{ prompts: Row[] }>('prompts').then((r) => r.prompts),
      adm.get<{ questions: Row[] }>('questions').then((r) => r.questions),
    ])
  }

  function openTest(t: Row | null) {
    note = ''
    edit = t ? { ...t } : { kind: 'reading', title: '', minutes: 20, full_size: false, passage: '', transcript: '', active: false }
    qjson = t ? JSON.stringify(t.questions, null, 1) : sample
  }

  async function saveTest() {
    if (!edit) return
    let qs
    try {
      qs = JSON.parse(qjson)
    } catch {
      note = 'Вопросы — не JSON. Образец: ' + sample
      return
    }
    const body = { kind: edit.kind, title: edit.title, full_size: edit.full_size, minutes: Number(edit.minutes), passage: edit.passage, transcript: edit.transcript, questions: qs, active: edit.active }
    try {
      const res = edit.id ? await adm.put<{ id: number }>(`tests/${edit.id}`, body) : await adm.post<{ id: number }>('tests', body)
      edit.id = res.id
      note = 'Сохранено.'
      await load()
    } catch {
      note = 'Не сохранилось: проверьте название, минуты (1–90) и вопросы (2–6 вариантов, answer — номер верного с 0).'
    }
  }

  async function upload(blob: Blob) {
    if (!edit?.id) return
    busy = 'Загрузка…'
    try {
      await request('POST', `/api/admin/tests/${edit.id}/audio`, blob)
      note = `Аудио сохранено (${Math.round(blob.size / 1024)} КБ). Listening виден ученикам, если задание активно.`
      await load()
      edit = tests!.find((t) => t.id === edit!.id) ?? edit
    } catch {
      note = 'Аудио не сохранилось: нужен mp3, m4a, ogg, webm или wav до 15 МБ.'
    } finally {
      busy = ''
    }
  }

  // Озвучка: сервер отдаёт WAV (~2.8 МБ в минуту), браузер пережимает его в opus (~0.35 МБ в минуту),
  // проигрывая в реальном времени — сколько длится запись, столько и ждать.
  async function speak() {
    if (!edit?.id) return
    busy = 'Озвучка…'
    note = ''
    try {
      const res = await fetch(`/api/admin/tests/${edit.id}/speak`, { method: 'POST', headers: { Authorization: authHeader() } })
      if (!res.ok) throw new Error('speak')
      const wav = await res.blob()
      const ctx = new AudioContext()
      const buf = await ctx.decodeAudioData(await wav.arrayBuffer())
      const dest = ctx.createMediaStreamDestination()
      const src = ctx.createBufferSource()
      src.buffer = buf
      src.connect(dest)
      const type = ['audio/ogg;codecs=opus', 'audio/webm;codecs=opus', 'audio/mp4'].find((t) => MediaRecorder.isTypeSupported(t))
      const rec = new MediaRecorder(dest.stream, { ...(type ? { mimeType: type } : {}), audioBitsPerSecond: 48_000 })
      const chunks: Blob[] = []
      rec.ondataavailable = (e) => e.data.size && chunks.push(e.data)
      const done = new Promise<void>((ok) => (rec.onstop = () => ok()))
      const started = Date.now()
      const tick = setInterval(() => (busy = `Сжатие: ${Math.round((Date.now() - started) / 1000)} из ${Math.round(buf.duration)} с…`), 1000)
      rec.start(1000)
      src.onended = () => rec.stop()
      src.start()
      await done
      clearInterval(tick)
      ctx.close()
      await upload(new Blob(chunks, { type: rec.mimeType || type || 'audio/webm' }))
    } catch {
      busy = ''
      note = 'Озвучка не удалась: проверьте GEMINI_API_KEY и сценарий («Имя: реплика» по строкам).'
    }
  }

  // Темы Writing.
  let ptask = $state('task2')
  let ptext = $state('')
  let pimage: Blob | null = null
  async function addPrompt() {
    const f = new FormData()
    f.append('task', ptask)
    f.append('text', ptext)
    if (pimage) f.append('image', pimage, 'chart.jpg')
    try {
      await request('POST', '/api/admin/prompts', f)
      ptext = ''
      pimage = null
      note = 'Тема добавлена.'
      await load()
    } catch {
      note = 'Не добавилось: Task 2 — от 20 знаков, картинка — JPG или PNG.'
    }
  }
  async function togglePrompt(p: Row) {
    const f = new FormData()
    f.append('task', p.task)
    f.append('text', p.text)
    f.append('active', String(!p.active))
    await request('POST', `/api/admin/prompts/${p.id}`, f)
    await load()
  }

  // Вопросы Speaking.
  let qpart = $state(1)
  let qtext = $state('')
  async function addQuestion() {
    try {
      await adm.post('questions', { part: Number(qpart), text: qtext, active: true })
      qtext = ''
      await load()
    } catch {
      note = 'Не добавилось.'
    }
  }
  async function toggleQuestion(q: Row) {
    await adm.put(`questions/${q.id}`, { part: q.part, text: q.text, active: !q.active })
    await load()
  }

  onMount(load)
</script>

<div class="seg">
  <button class:on={view === 'tests'} onclick={() => ((view = 'tests'), (edit = null))}>Reading / Listening</button>
  <button class:on={view === 'prompts'} onclick={() => (view = 'prompts')}>Темы Writing</button>
  <button class:on={view === 'questions'} onclick={() => (view = 'questions')}>Вопросы Speaking</button>
</div>
{#if note}<p class="hint">{note}</p>{/if}
{#if busy}<p><b>{busy}</b></p>{/if}

{#if !tests || !prompts || !questions}
  <Skeleton rows={4} />
{:else if view === 'tests'}
  {#if edit}
    <button class="btn btn-plain" onclick={() => (edit = null)}>‹ К списку</button>
    <section class="card stack">
      <select bind:value={edit.kind}>
        {#each Object.entries(kinds) as [k, v] (k)}<option value={k}>{v}</option>{/each}
      </select>
      <input bind:value={edit.title} placeholder="Название" />
      <label>Минут на таймере <input type="number" min="1" max="90" bind:value={edit.minutes} /></label>
      <label class="check"><input type="checkbox" bind:checked={edit.full_size} /> Полноразмерная часть для mock (около 40 вопросов)</label>
      {#if edit.kind === 'reading'}<textarea bind:value={edit.passage} rows="10" placeholder="Текст Reading. Абзацы — через пустую строку."></textarea>{/if}
      {#if edit.kind === 'listening'}<textarea bind:value={edit.transcript} rows="10" placeholder="Сценарий: «Имя: реплика» по строкам. Ученик видит его после сдачи; по нему же — озвучка."></textarea>{/if}
      <textarea class="mono" bind:value={qjson} rows="12"></textarea>
      <p class="hint">type: tfng, detail, main, vocab, grammar — по нему считаются слабые места в отчёте mock.</p>
      <label class="check"><input type="checkbox" bind:checked={edit.active} /> Видно ученикам</label>
      <button class="btn" onclick={saveTest}>Сохранить</button>
    </section>
    {#if edit.id && edit.kind === 'listening'}
      <section class="card stack">
        <h2>Аудио {edit.audio_id ? '✓ есть' : '— нет, ученикам не видно'}</h2>
        {#if edit.audio_id}<audio controls src="/media/{edit.audio_id}"></audio>{/if}
        <label class="btn btn-plain file">Загрузить запись (mp3, m4a, ogg)<input type="file" accept="audio/*" onchange={(e) => upload((e.target as HTMLInputElement).files![0]!)} /></label>
        <button class="btn btn-plain" disabled={!!busy || !edit.transcript} onclick={speak}>Озвучить сценарий (AI, два голоса)</button>
      </section>
    {/if}
  {:else}
    <button class="btn" onclick={() => openTest(null)}>+ Новое задание</button>
    <ul class="card rows">
      {#each tests as t (t.id)}
        <li>
          <button class="row" onclick={() => openTest(t)}>
            <span class="grow">
              <strong>{t.title}</strong>
              <span class="tag">{kinds[t.kind]} · {t.questions.length} вопр. · {t.minutes} мин{t.full_size ? ' · для mock' : ''}{t.kind === 'listening' && !t.audio_id ? ' · без аудио' : ''} · сдали {t.attempts}</span>
            </span>
            <span class="tag">{t.active ? 'видно' : 'скрыто'}</span>
          </button>
        </li>
      {/each}
    </ul>
  {/if}
{:else if view === 'prompts'}
  <section class="card stack">
    <select bind:value={ptask}><option value="task2">Task 2 — эссе</option><option value="task1">Task 1 — график (с картинкой)</option></select>
    <textarea bind:value={ptext} rows="4" placeholder="Текст задания"></textarea>
    {#if ptask === 'task1'}
      <label class="btn btn-plain file">Картинка графика<input type="file" accept="image/*" onchange={async (e) => (pimage = await compressPhoto((e.target as HTMLInputElement).files![0]!))} /></label>
    {/if}
    <button class="btn" onclick={addPrompt}>Добавить тему</button>
  </section>
  <ul class="card rows">
    {#each prompts as p (p.id)}
      <li class="row">
        <span class="grow"><span class="tag">{p.task}{p.image_id ? ' · с графиком' : ''}</span>{p.text}</span>
        <button class="btn btn-small btn-plain" onclick={() => togglePrompt(p)}>{p.active ? 'Скрыть' : 'Показать'}</button>
      </li>
    {/each}
  </ul>
{:else}
  <section class="card stack">
    <select bind:value={qpart}><option value={1}>Part 1</option><option value={2}>Part 2 — карточка</option><option value={3}>Part 3</option></select>
    <textarea bind:value={qtext} rows="3" placeholder="Вопрос. Для карточки Part 2 — тема, с новой строки «You should say: …»"></textarea>
    <button class="btn" disabled={!qtext.trim()} onclick={addQuestion}>Добавить вопрос</button>
  </section>
  <ul class="card rows">
    {#each questions as q (q.id)}
      <li class="row">
        <span class="grow"><span class="tag">Part {q.part}</span>{q.text}</span>
        <button class="btn btn-small btn-plain" onclick={() => toggleQuestion(q)}>{q.active ? 'Скрыть' : 'Показать'}</button>
      </li>
    {/each}
  </ul>
{/if}

<style>
  h2 {
    font-size: 16px;
  }

  strong,
  .grow .tag {
    display: block;
  }

  input:not([type='checkbox']):not([type='file']),
  select {
    width: 100%;
    min-height: 40px;
    padding: 0 10px;
    border: 1px solid var(--separator);
    border-radius: 10px;
    background: var(--section);
    color: var(--text);
    font: inherit;
  }

  label {
    display: grid;
    gap: 4px;
  }

  .check {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .mono {
    font-family: ui-monospace, monospace;
    font-size: 13px;
  }

  .file {
    position: relative;
    overflow: hidden;
    text-align: center;
  }

  .file input {
    position: absolute;
    inset: 0;
    opacity: 0;
  }

  audio {
    width: 100%;
  }
</style>
