<script lang="ts">
  import { onDestroy } from 'svelte'
  import { markCameraChecked, openCamera, stopCamera, watch, type Look } from '../lib/camera'
  import { t } from '../lib/i18n/index.svelte'

  // Проверка камеры перед полным mock: лицо видно секунду подряд — проверка пройдена (DECISIONS 2026-11-24).
  // Заодно грузится распознавание, и в частях mock камера включается сразу.
  let { ondone }: { ondone: () => void } = $props()

  let video = $state<HTMLVideoElement | null>(null)
  let phase = $state<'idle' | 'loading' | 'on' | 'denied'>('idle')
  let now = $state<Look>('none')
  let stream: MediaStream | null = null
  let stop: (() => void) | undefined
  let okSince = 0
  let gone = false

  function finish() {
    stop?.()
    stopCamera(stream)
    stream = null
  }

  async function check() {
    phase = 'loading'
    try {
      stream = await openCamera(video!)
      stop = await watch(video!, (l) => {
        now = l
        if (l !== 'ok') {
          okSince = 0
          return
        }
        okSince ||= Date.now()
        if (Date.now() - okSince >= 1000) {
          markCameraChecked()
          finish()
          ondone()
        }
      })
      if (gone) return finish()
      phase = 'on'
    } catch {
      finish()
      phase = 'denied'
    }
  }

  onDestroy(() => {
    gone = true
    finish()
  })
</script>

<section class="card stack">
  <h2>{t('cam.check')}</h2>
  <p class="hint">{t('cam.checkHint')}</p>
  <video bind:this={video} muted playsinline class:shown={phase === 'on'}></video>
  {#if phase === 'on'}
    <p class:bad={now !== 'ok'}>{now === 'ok' ? t('cam.ok') : now === 'many' ? t('cam.many') : now === 'away' ? t('cam.away') : t('cam.none')}</p>
  {:else if phase === 'loading'}
    <p class="hint">{t('cam.loading')}</p>
  {:else}
    {#if phase === 'denied'}<p class="error">{t('cam.denied')}</p>{/if}
    <button class="btn" onclick={check}>{t('cam.start')}</button>
  {/if}
</section>

<style>
  h2 {
    font-size: 17px;
  }

  video {
    display: none;
    width: 100%;
    max-width: 320px;
    aspect-ratio: 4 / 3;
    object-fit: cover;
    border-radius: var(--radius);
    transform: scaleX(-1);
    background: var(--shimmer);
  }

  video.shown {
    display: block;
  }

  .bad {
    color: var(--danger);
  }
</style>
