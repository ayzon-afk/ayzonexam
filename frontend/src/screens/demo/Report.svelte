<script lang="ts">
  import Delta from '../../components/Delta.svelte'
  import Head from '../../components/Head.svelte'
  import ShareCard from '../../components/ShareCard.svelte'
  import { report } from '../../lib/demo'
  import { formatBand } from '../../lib/exam'
  import { t } from '../../lib/i18n/index.svelte'
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
        <Delta n={p.band - p.prev} band />
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
<ShareCard
  big={formatBand(report.overall)}
  title="Overall Band · IELTS-style mock"
  sub={report.parts.map((p) => `${p.name.slice(0, 1)} ${formatBand(p.band)}`).join('  ')}
  text={t('share.report', { band: formatBand(report.overall) })}
/>
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
</style>
