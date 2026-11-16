<script lang="ts">
  import Head from '../../components/Head.svelte'
  import NewsList from '../../components/NewsList.svelte'
  import { news } from '../../lib/demo'
  import { t } from '../../lib/i18n/index.svelte'
</script>

<Head title={t('news.title')} sub={t('news.lead')} />
<NewsList items={news} />
