<script lang="ts">
  import { onMount } from 'svelte'
  import Head from '../../components/Head.svelte'
  import NewsList from '../../components/NewsList.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { api, type Post } from '../../lib/api'
  import { t } from '../../lib/i18n/index.svelte'

  let posts = $state<Post[] | null>(null)
  let failed = $state(false)

  onMount(async () => {
    try {
      posts = (await api.news(50)).posts
    } catch {
      failed = true
    }
  })
</script>

<Head title={t('news.title')} sub={t('news.lead')} demo={false} />
{#if failed}
  <p class="error">{t('app.error')}</p>
{:else if !posts}
  <Skeleton rows={4} />
{:else}
  <NewsList items={posts} />
{/if}
