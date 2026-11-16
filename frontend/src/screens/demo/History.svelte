<script lang="ts">
  import Head from '../../components/Head.svelte'
  import { fmtDate } from '../../lib/date'
  import { attempts, type Attempt } from '../../lib/demo'
  import { formatBand } from '../../lib/exam'
  import { t } from '../../lib/i18n/index.svelte'
  import { go } from '../../lib/nav.svelte'

  function name(k: Attempt['kind']): string {
    if (k === 'mock') return t('mock.title')
    if (k === 'tournament') return t('tour.title')
    return k[0]!.toUpperCase() + k.slice(1)
  }
</script>

<Head title={t('history.title')} />

<ul class="card rows">
  {#each attempts as a (a.date)}
    <li>
      <button class="row" onclick={() => a.kind === 'mock' && go({ s: 'report' })}>
        <span class="grow">
          <strong>{name(a.kind)}</strong>
          <span class="tag">{fmtDate(a.date)}</span>
        </span>
        <span class="num">{formatBand(a.band)}</span>
      </button>
    </li>
  {/each}
</ul>

<style>
  strong {
    display: block;
    font-weight: 600;
  }

  .num {
    font-size: 20px;
  }
</style>
