<script lang="ts">
  import { onMount } from 'svelte'
  import Head from '../../components/Head.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { api, type TestSummary } from '../../lib/api'
  import { formatBand } from '../../lib/exam'
  import { t } from '../../lib/i18n/index.svelte'
  import { go, replace, type Part } from '../../lib/nav.svelte'

  // Список сетов практики одного вида; у теста уровня сет один — он открывается сразу.
  let { r }: { r: { part: Part } } = $props()

  // svelte-ignore state_referenced_locally
  const part = r.part
  const title = part === 'reading' ? 'Reading' : part === 'listening' ? 'Listening' : t('level.title')

  let list = $state<TestSummary[] | null>(null)
  let failed = $state(false)

  async function load() {
    failed = false
    try {
      list = (await api.tests(part)).tests
      if (part === 'level' && list.length === 1) replace({ s: 'test', from: { test: list[0]!.id } })
    } catch {
      failed = true
    }
  }

  onMount(load)
</script>

<Head {title} sub={t(part === 'level' ? 'level.lead' : 'practice.lead')} demo={false} />

{#if failed}
  <p class="error">{t('app.error')}</p>
  <button class="btn btn-plain" onclick={load}>{t('app.retry')}</button>
{:else if !list}
  <Skeleton rows={3} />
{:else if list.length === 0}
  <p class="card hint">{t('practice.empty')}</p>
{:else}
  <ul class="card rows">
    {#each list as s (s.id)}
      <li>
        <button class="row" onclick={() => go({ s: 'test', from: { test: s.id } })}>
          <span class="grow">
            <strong>{s.title}</strong>
            <span class="tag">{t('practice.sub', { n: s.count, min: s.minutes })}{s.full_size ? ` · ${t('practice.full')}` : ''}</span>
          </span>
          {#if s.last_band !== null}<span class="num">{formatBand(s.last_band)}</span>{/if}
          <span class="chev" aria-hidden="true">›</span>
        </button>
      </li>
    {/each}
  </ul>
  <p class="hint">{t('practice.rule')}</p>
{/if}

<style>
  strong {
    display: block;
    font-weight: 600;
  }

  .num {
    font-size: 20px;
  }
</style>
