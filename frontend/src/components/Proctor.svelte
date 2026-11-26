<script lang="ts">
  import { onDestroy, onMount } from 'svelte'
  import { AWAY_MS, openCamera, stopCamera, watch, type Look } from '../lib/camera'
  import { t } from '../lib/i18n/index.svelte'
  import { arm, canLock, lockScreen, secondScreen, unlockScreen, warn } from '../lib/proctor'

  // Камера во время части полного mock (DECISIONS 2026-11-24): маленькое окно в углу. Не на экран дольше
  // AWAY_MS (нет лица, два лица, взгляд или голова в сторону) — сигнал и нарушение; каждые следующие
  // AWAY_MS — ещё одно. Без камеры часть закрыта полупрозрачным экраном с просьбой включить её.
  // sound = false — без сигнала, только вибрация и надпись: в Speaking сигнал попал бы в запись ответа.
  // Как на экзамене (DECISIONS 2026-11-25): часть — на весь экран браузера, выход — нарушение; копировать,
  // вставлять, меню и перетаскивание — нельзя, попытка вставить — нарушение; второй монитор — часть закрыта.
  let { onviolation, sound = true }: { onviolation: () => void; sound?: boolean } = $props()

  let video = $state<HTMLVideoElement | null>(null)
  let phase = $state<'loading' | 'on' | 'denied'>('loading')
  let now = $state<Look>('ok')
  let stream: MediaStream | null = null
  let stop: (() => void) | undefined
  let awaySince = 0
  let gone = false // экран закрыт, пока грузилась камера, — сразу выключить её
  let full = $state(true) // на весь экран (или браузер этого не умеет)
  let extended = $state(secondScreen())
  let noPaste = $state(false)
  let noPasteTimer: ReturnType<typeof setTimeout> | undefined

  function violation() {
    warn(sound)
    onviolation()
  }

  async function goFull() {
    full = !canLock() || (await lockScreen())
  }

  // Вышел из полного экрана во время части — нарушение и просьба вернуться. Ушёл в другое окно (Alt+Tab
  // тоже выходит из полного экрана) — это уже уход со страницы, его считает сама часть: не вдвойне.
  function onFullChange() {
    if (gone || !canLock()) return
    const now = !!document.fullscreenElement
    if (full && !now) setTimeout(() => !gone && !document.hidden && document.hasFocus() && violation(), 300)
    full = now
  }

  function block(e: Event) {
    e.preventDefault()
    if (e.type === 'paste' || e.type === 'drop') violation()
    noPaste = true
    clearTimeout(noPasteTimer)
    noPasteTimer = setTimeout(() => (noPaste = false), 3000)
  }
  const blocked = ['copy', 'cut', 'paste', 'drop', 'contextmenu'] as const

  function onLook(l: Look) {
    now = l
    if (l === 'ok') {
      awaySince = 0
      return
    }
    const ts = Date.now()
    if (!awaySince) awaySince = ts
    if (ts - awaySince >= AWAY_MS) {
      awaySince = ts
      violation()
    }
  }

  async function start() {
    phase = 'loading'
    try {
      stream = await openCamera(video!)
      stop = await watch(video!, onLook)
      if (gone) {
        stop()
        stopCamera(stream)
        return
      }
      phase = 'on'
    } catch {
      stopCamera(stream)
      stream = null
      phase = 'denied'
    }
  }

  onMount(() => {
    arm()
    start()
    goFull()
    document.addEventListener('fullscreenchange', onFullChange)
    for (const ev of blocked) document.addEventListener(ev, block, true)
  })
  onDestroy(() => {
    gone = true
    stop?.()
    stopCamera(stream)
    clearTimeout(noPasteTimer)
    document.removeEventListener('fullscreenchange', onFullChange)
    for (const ev of blocked) document.removeEventListener(ev, block, true)
    unlockScreen()
  })
</script>

<div class="cam" class:bad={phase === 'on' && now !== 'ok'}>
  <video bind:this={video} muted playsinline></video>
  <p>
    {#if phase === 'loading'}{t('cam.loading')}
    {:else if now === 'none'}{t('cam.none')}
    {:else if now === 'many'}{t('cam.many')}
    {:else if now === 'away'}{t('cam.away')}
    {:else}{t('cam.watch')}{/if}
  </p>
</div>

{#if noPaste}<p class="toast" role="status">{t('cam.noPaste')}</p>{/if}

{#if extended}
  <div class="block" role="alertdialog" aria-live="assertive">
    <div class="card">
      <p>{t('cam.screens')}</p>
      <button class="btn" onclick={() => (extended = secondScreen())}>{t('cam.recheck')}</button>
    </div>
  </div>
{:else if !full}
  <div class="block" role="alertdialog" aria-live="assertive">
    <div class="card">
      <p>{t('cam.full')}</p>
      <button class="btn" onclick={goFull}>{t('cam.fullStart')}</button>
    </div>
  </div>
{/if}

{#if phase === 'denied'}
  <div class="block" role="alertdialog" aria-live="assertive">
    <div class="card">
      <p>{t('cam.denied')}</p>
      <button class="btn" onclick={start}>{t('cam.retry')}</button>
    </div>
  </div>
{/if}

<style>
  .cam {
    position: fixed;
    right: 16px;
    bottom: calc(16px + var(--safe-bottom));
    z-index: 30;
    width: 168px;
    background: var(--section);
    border: 2px solid var(--ok);
    border-radius: var(--radius);
    overflow: hidden;
    box-shadow: 0 4px 16px rgb(0 0 0 / 0.15);
  }

  .cam.bad {
    border-color: var(--danger);
  }

  video {
    display: block;
    width: 100%;
    aspect-ratio: 4 / 3;
    object-fit: cover;
    transform: scaleX(-1); /* как в зеркале — привычнее */
    background: var(--shimmer);
  }

  p {
    font-size: 12px;
    padding: 6px 8px;
  }

  .bad p {
    color: var(--danger);
    font-weight: 600;
  }

  .block {
    position: fixed;
    inset: 0;
    z-index: 40;
    display: grid;
    place-items: center;
    padding: 16px;
    background: rgb(0 0 0 / 0.6);
  }

  .toast {
    position: fixed;
    left: 50%;
    top: calc(var(--safe-top) + 16px);
    z-index: 45;
    transform: translateX(-50%);
    max-width: min(420px, calc(100vw - 32px));
    padding: 10px 14px;
    border-radius: var(--radius);
    background: var(--danger);
    color: #fff;
    font-size: 14px;
    font-weight: 600;
  }

  .block .card {
    display: grid;
    gap: 12px;
    max-width: 360px;
  }
</style>
