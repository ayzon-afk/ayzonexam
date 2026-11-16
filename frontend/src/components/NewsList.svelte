<script lang="ts">
  import { fmtDate } from '../lib/date'
  import type { News } from '../lib/demo'
  import { t } from '../lib/i18n/index.svelte'
  import { go } from '../lib/nav.svelte'

  let { items }: { items: News[] } = $props()
</script>

<ul class="card rows">
  {#each items as n (n.id)}
    <li>
      <button class="row" onclick={() => go({ s: 'post', id: n.id })}>
        <span class="grow">
          <span class="tag">{t(n.tag)} · {fmtDate(n.date)}</span>
          <strong>{t(n.title)}</strong>
        </span>
        <span class="chev" aria-hidden="true">›</span>
      </button>
    </li>
  {/each}
</ul>

<style>
  .grow {
    display: grid;
    gap: 2px;
  }

  strong {
    font-weight: 600;
  }
</style>
