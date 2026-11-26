<script lang="ts">
  import { onDestroy, onMount } from 'svelte'
  import CameraCheck from '../../components/CameraCheck.svelte'
  import Head from '../../components/Head.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { api, ApiError, type MockPart, type MockQuota, type MockSummary, type MockView } from '../../lib/api'
  import { cameraChecked, isLaptop } from '../../lib/camera'
  import { clock, fmtDate } from '../../lib/date'
  import { formatBand } from '../../lib/exam'
  import { t, type Key } from '../../lib/i18n/index.svelte'
  import { go, type Route } from '../../lib/nav.svelte'
  import { haptic } from '../../lib/telegram'

  // Полный mock: Listening → Reading → Writing → Speaking. Новый (strict, DECISIONS 2026-11-26) — как на
  // экзамене: L → R → W подряд, следующая часть открывается только после предыдущей, её таймер идёт сам;
  // Speaking — в любое время; баллы — в конце. Mock до выкладки — по частям с паузами, как раньше.
  // Writing и Speaking проверяет AI — бот сообщит итог.
  let active = $state<MockView | null>(null)
  let past = $state<MockSummary[]>([])
  let quota = $state<MockQuota | null>(null)
  let failed = $state(false)
  let starting = $state(false)
  let error = $state<Key | ''>('')
  let now = $state(Date.now())
  // Полный mock — только с ноутбука и после проверки камеры (DECISIONS 2026-11-24).
  const laptop = isLaptop()
  let camOk = $state(cameraChecked())
  const ready = $derived(laptop && camOk)
  let agreed = $state(false)

  const parts: { id: MockPart; name: string; min: number }[] = [
    { id: 'listening', name: 'Listening', min: 30 },
    { id: 'reading', name: 'Reading', min: 60 },
    { id: 'writing', name: 'Writing', min: 60 },
    { id: 'speaking', name: 'Speaking', min: 14 },
  ]
  const label: Record<string, Key> = { todo: 'mock.todo', running: 'mock.now', checking: 'mock.checking', done: 'mock.done' }

  // Строгий mock: Reading — после Listening, Writing — после Reading; Speaking — когда угодно.
  const before: Partial<Record<MockPart, MockPart>> = { reading: 'listening', writing: 'reading' }
  function locked(m: MockView, p: MockPart): boolean {
    const prev = before[p]
    const st = prev ? m.parts[prev].state : 'done'
    return m.strict && (st === 'todo' || st === 'running')
  }

  async function load() {
    failed = false
    try {
      const res = await api.mocks()
      active = res.active
      past = res.past
      quota = res.quota
    } catch {
      failed = true
    }
  }

  async function start() {
    if (!ready) return
    starting = true
    error = ''
    try {
      active = await api.newMock()
      haptic.success()
    } catch (e) {
      haptic.error()
      error = e instanceof ApiError && e.code === 'payment_required' ? 'mock.pay' : 'app.error'
    } finally {
      starting = false
    }
  }

  function open(p: MockPart) {
    if (!active || !ready) return
    const st = active.parts[p].state
    if (st === 'done' || st === 'checking' || locked(active, p)) return
    const r: Route =
      p === 'listening' || p === 'reading'
        ? { s: 'test', from: { mock: active.id, kind: p } }
        : p === 'writing'
          ? { s: 'mockWriting', id: active.id }
          : { s: 'mockSpeaking', id: active.id }
    go(r)
  }

  // Пока AI проверяет Writing или Speaking — обновлять раз в 10 с.
  const tick = setInterval(() => {
    now = Date.now()
    if (active && Object.values(active.parts).some((p) => p.state === 'checking') && now % 10_000 < 1000) load()
  }, 1000)
  onDestroy(() => clearInterval(tick))
  onMount(load)

  const failedChecks = $derived(active ? active.writing_checks.some((c) => c.status === 'failed') || active.speaking_check?.status === 'failed' : false)
</script>

<Head title={t('mock.title')} sub={active && !active.strict ? t('mock.lead') : t('mock.strictLead')} demo={false} />

{#if !laptop}
  <p class="card">{t('mock.laptopOnly', { site: location.host })}</p>
{:else if !camOk}
  <CameraCheck ondone={() => (camOk = true)} />
{/if}

{#if failed}
  <p class="error">{t('app.error')}</p>
  <button class="btn btn-plain" onclick={load}>{t('app.retry')}</button>
{:else if !quota}
  <Skeleton rows={4} />
{:else if active}
  <section class="card stack">
    <div class="top">
      <h2>{t('mock.current')}</h2>
      <span class="hint">{fmtDate(active.started_at)}</span>
    </div>
    <ol class="rows parts">
      {#each parts as p (p.id)}
        {@const st = active.parts[p.id]}
        <li>
          <button class="row" disabled={!ready || st.state === 'done' || st.state === 'checking' || locked(active, p.id)} onclick={() => open(p.id)}>
            <span class="dot {st.state}" aria-hidden="true"></span>
            <span class="grow">
              <strong>{p.name}</strong>
              <span class="tag">
                {t('time.min', { n: p.min })} · {t(label[st.state]!)}{st.ends_at && st.state === 'running' ? ` · ${clock(new Date(st.ends_at).getTime() - now)}` : ''}
              </span>
            </span>
            {#if st.band !== null}<span class="num">{formatBand(st.band)}</span>{:else if st.state !== 'checking'}<span class="chev" aria-hidden="true">›</span>{/if}
          </button>
        </li>
      {/each}
    </ol>
    {#if failedChecks}<p class="error">{t('mock.failedCheck')}</p>{/if}
    <p class="hint">{active.strict ? t('mock.strictSaved') : t('mock.saved')}</p>
  </section>
{:else}
  <section class="card stack">
    <h2>{t('mock.new')}</h2>
    <p class="hint">
      {#if quota.premium}
        {t('mock.quota', { n: Math.max(quota.limit - quota.used, 0), limit: quota.limit })}
      {:else}
        {t('mock.free')}
      {/if}
      {#if quota.credits > 0}{t('mock.credits', { n: quota.credits })}{/if}
    </p>
    <!-- Правила — до таймера, как инструктаж на экзамене (DECISIONS 2026-11-25). -->
    <h3>{t('mock.rules')}</h3>
    <ul class="rules">
      {#each ['mock.rule5', 'mock.rule1', 'mock.rule2', 'mock.rule3', 'mock.rule4'] as const as k (k)}<li>{t(k)}</li>{/each}
    </ul>
    <label class="agree"><input type="checkbox" bind:checked={agreed} /> {t('mock.agree')}</label>
    <button class="btn" disabled={starting || !ready || !agreed} onclick={start}>{t('mock.start')}</button>
    {#if error === 'mock.pay'}
      <p>{t('mock.pay')}</p>
      <button class="btn btn-plain" onclick={() => go({ s: 'premium' })}>{t('premium.go')}</button>
    {:else if error}
      <p class="error">{t(error)}</p>
    {/if}
  </section>
{/if}

{#if past.length}
  <div class="sect"><h2>{t('mock.past')}</h2></div>
  <ul class="card rows">
    {#each past as m (m.id)}
      <li>
        <button class="row" onclick={() => go({ s: 'report', id: m.id })}>
          <span class="grow">{t('mock.title')} · {fmtDate(m.finished_at ?? m.started_at)}</span>
          {#if m.overall !== null}<span class="num">{formatBand(m.overall)}</span>{/if}
          <span class="chev" aria-hidden="true">›</span>
        </button>
      </li>
    {/each}
  </ul>
{/if}

<style>
  .top {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
  }

  h2 {
    font-size: 17px;
  }

  .parts {
    padding: 0;
  }

  strong {
    display: block;
    font-weight: 600;
  }

  .num {
    font-size: 20px;
  }

  .row:disabled {
    cursor: default;
    color: inherit;
  }

  .dot {
    width: 12px;
    height: 12px;
    flex: none;
    border-radius: 50%;
    border: 2px solid var(--hint);
  }

  .dot.done {
    background: #2e9d58;
    border-color: #2e9d58;
  }

  .dot.running,
  .dot.checking {
    border-color: var(--accent);
    background: color-mix(in srgb, var(--accent) 35%, transparent);
  }

  h3 {
    margin: 4px 0 0;
    font-size: 15px;
  }

  .rules {
    display: grid;
    gap: 6px;
    margin: 0;
    padding-left: 20px;
    font-size: 14px;
  }

  .agree {
    display: flex;
    align-items: center;
    gap: 8px;
  }
</style>
