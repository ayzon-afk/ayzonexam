<script lang="ts">
  import { onMount } from 'svelte'
  import Head from '../../components/Head.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { api, type Post } from '../../lib/api'
  import { fmtDate } from '../../lib/date'
  import { t, type Key } from '../../lib/i18n/index.svelte'
  import { share } from '../../lib/share'

  let { r }: { r: { id: number } } = $props()

  let post = $state<Post | null>(null)
  let failed = $state(false)
  let copied = $state(false)

  onMount(async () => {
    try {
      post = await api.post(r.id)
    } catch {
      failed = true
    }
  })
</script>

{#if failed}
  <Head title={t('news.title')} demo={false} />
  <p class="error">{t('app.error')}</p>
{:else if !post}
  <Skeleton rows={4} />
{:else}
  <Head title={post.title} sub="{t(`news.tag.${post.tag}` as Key)} · {fmtDate(post.created_at)}" demo={false} />
  <article class="card stack">
    {#each post.body.split(/\n\s*\n/) as p, i (i)}
      <p>{p}</p>
    {/each}
  </article>
  <button class="btn btn-plain" onclick={async () => (copied = (await share(post!.title)) === 'copied')}>{t('share')}</button>
  {#if copied}<p class="hint">{t('share.copied')}</p>{/if}
{/if}

<style>
  p {
    white-space: pre-wrap;
    line-height: 1.55;
  }
</style>
