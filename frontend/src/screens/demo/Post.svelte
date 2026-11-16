<script lang="ts">
  import Head from '../../components/Head.svelte'
  import { fmtDate } from '../../lib/date'
  import { news } from '../../lib/demo'
  import { t } from '../../lib/i18n/index.svelte'

  let { r }: { r: { id: number } } = $props()

  const post = $derived(news.find((n) => n.id === r.id) ?? news[0]!)
  let shared = $state(false)
</script>

<Head title={t(post.title)} sub="{t(post.tag)} · {fmtDate(post.date)}" />
<article class="card stack">
  {#each t(post.body).split('\n\n') as p, i (i)}
    <p>{p}</p>
  {/each}
</article>
<button class="btn btn-plain" onclick={() => (shared = true)}>{t('share')}</button>
{#if shared}<p class="hint">{t('demo.later')}</p>{/if}
