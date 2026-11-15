<script lang="ts">
  import type { Me } from '../lib/api'
  import { daysUntil, formatBand } from '../lib/exam'
  import { t } from '../lib/i18n/index.svelte'

  let { me }: { me: Me } = $props()

  const days = $derived(me.exam_date ? daysUntil(me.exam_date, new Date()) : null)
</script>

<h1>{t('home.greeting', { name: me.first_name || me.nickname })}</h1>

<div class="card goal">
  <div>
    <p class="hint">{t('home.goal')}</p>
    <p class="band">{me.goal_band === null ? '—' : formatBand(me.goal_band)}</p>
  </div>
  <p class="days">
    {#if days === null || days < 0}
      {t('home.noDate')}
    {:else if days === 0}
      {t('home.examToday')}
    {:else}
      {t('home.daysLeft', { days })}
    {/if}
  </p>
</div>

<div class="card">
  <p class="hint">{t('home.soon')}</p>
</div>

<style>
  h1 {
    font-size: 26px;
  }

  .goal {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .band {
    font-size: 40px;
    font-weight: 700;
    line-height: 1.1;
    font-variant-numeric: tabular-nums;
    color: var(--accent);
  }

  .days {
    text-align: right;
    font-weight: 500;
  }
</style>
