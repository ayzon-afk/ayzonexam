<script lang="ts">
  import Head from '../../components/Head.svelte'
  import { report } from '../../lib/demo'
  import { formatBand } from '../../lib/exam'
  import { t } from '../../lib/i18n/index.svelte'

  let shared = $state(false)
</script>

<Head title={t('report.title')} />

<section class="card overall">
  <p class="hint">Overall Band</p>
  <p class="num band">{formatBand(report.overall)}</p>
  <div class="parts">
    {#each report.parts as p (p.name)}
      <div>
        <p class="num">{formatBand(p.band)}</p>
        <p class="hint">{p.name}</p>
      </div>
    {/each}
  </div>
</section>

<section class="card stack">
  <h2>{t('report.weak')}</h2>
  <ul>
    {#each report.weak as w (w)}<li>{t(w)}</li>{/each}
  </ul>
</section>

<section class="card stack">
  <h2>{t('report.plan')}</h2>
  <ol>
    {#each report.plan as p (p)}<li>{t(p)}</li>{/each}
  </ol>
</section>

<div class="sect"><h2>{t('report.story')}</h2></div>
<div class="story" aria-label={t('report.story')}>
  <p class="brand">AYZON EXAM</p>
  <p class="big">{formatBand(report.overall)}</p>
  <p>Overall Band · IELTS-style mock</p>
  <div class="mini">
    {#each report.parts as p (p.name)}<span>{p.name.slice(0, 1)} {formatBand(p.band)}</span>{/each}
  </div>
</div>
<button class="btn" onclick={() => (shared = true)}>{t('share')}</button>
{#if shared}<p class="hint">{t('demo.later')}</p>{/if}
<p class="hint">{t('result.disclaimer')}</p>

<style>
  h2 {
    font-size: 17px;
  }

  .overall {
    text-align: center;
    display: grid;
    gap: 4px;
  }

  .band {
    font-size: 52px;
    line-height: 1.1;
  }

  .parts {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 4px;
    margin-top: 8px;
  }

  .parts .num {
    font-size: 20px;
  }

  ul,
  ol {
    margin: 0;
    padding-left: 20px;
    display: grid;
    gap: 8px;
  }

  .story {
    aspect-ratio: 9 / 12;
    max-width: 320px;
    width: 100%;
    margin: 0 auto;
    padding: 24px;
    border-radius: 20px;
    display: grid;
    align-content: center;
    justify-items: center;
    gap: 6px;
    text-align: center;
    color: #fff;
    background: linear-gradient(160deg, #1d4ed8, #7c3aed 60%, #db2777);
  }

  .brand {
    font-weight: 800;
    letter-spacing: 0.08em;
    opacity: 0.85;
  }

  .big {
    font-size: 88px;
    font-weight: 800;
    line-height: 1;
    font-variant-numeric: tabular-nums;
  }

  .mini {
    display: flex;
    gap: 10px;
    margin-top: 8px;
    font-weight: 600;
    font-variant-numeric: tabular-nums;
  }
</style>
