<script lang="ts">
  import BandPicker from '../../components/BandPicker.svelte'
  import Head from '../../components/Head.svelte'
  import { peerEssay } from '../../lib/demo-exam'
  import { t } from '../../lib/i18n/index.svelte'
  import { haptic } from '../../lib/telegram'

  // Взаимная проверка: оценил чужое эссе — получил бонусную проверку своего (SPEC).
  let band = $state<number | null>(null)
  let comment = $state('')
  let sent = $state(false)
</script>

<Head title={t('review.title')} sub={t('review.lead')} />

{#if sent}
  <section class="card stack done">
    <p class="big">+1</p>
    <p>{t('review.thanks')}</p>
  </section>
{:else}
  <article class="card stack">
    <p class="tag">Task 2 · {t('review.anon')}</p>
    <p class="essay">{peerEssay}</p>
  </article>
  <section class="card stack">
    <h2>{t('review.band')}</h2>
    <BandPicker bind:value={band} />
  </section>
  <textarea bind:value={comment} placeholder={t('review.comment')}></textarea>
  <button
    class="btn"
    disabled={band === null || comment.trim().length < 10}
    onclick={() => {
      haptic.success()
      sent = true
    }}>{t('review.send')}</button
  >
{/if}

<style>
  h2 {
    font-size: 16px;
  }

  .essay {
    line-height: 1.55;
  }

  .done {
    text-align: center;
  }

  .big {
    font-size: 44px;
    font-weight: 800;
    color: var(--accent);
  }
</style>
