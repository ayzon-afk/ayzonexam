<script lang="ts">
  import type { Post } from '../lib/api'
  import { fmtDate } from '../lib/date'
  import { t, type Key } from '../lib/i18n/index.svelte'
  import { go } from '../lib/nav.svelte'

  let { items }: { items: Post[] } = $props()
</script>

{#if items.length === 0}
  <p class="card hint">{t('news.empty')}</p>
{:else}
  <ul class="card rows">
    {#each items as n (n.id)}
      <li>
        <button class="row" onclick={() => go({ s: 'post', id: n.id })}>
          <span class="grow">
            <span class="tag">{t(`news.tag.${n.tag}` as Key)} · {fmtDate(n.created_at)}</span>
            <strong>{n.title}</strong>
          </span>
          <span class="chev" aria-hidden="true">›</span>
        </button>
      </li>
    {/each}
  </ul>
{/if}

<style>
  .grow {
    display: grid;
    gap: 2px;
  }

  strong {
    font-weight: 600;
  }
</style>
