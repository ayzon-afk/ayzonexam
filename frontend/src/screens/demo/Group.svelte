<script lang="ts">
  import Head from '../../components/Head.svelte'
  import { fmtDate } from '../../lib/date'
  import { groups, homework } from '../../lib/demo'
  import { formatBand } from '../../lib/exam'
  import { t, type Key } from '../../lib/i18n/index.svelte'
  import { go, type Route } from '../../lib/nav.svelte'

  let { r }: { r: { id: number } } = $props()

  const g = $derived(groups.find((x) => x.id === r.id) ?? groups[0]!)
  const list = $derived(homework.filter((h) => h.group === g.id))

  const kind: Record<'writing' | 'speaking' | 'mock', { name: Key; r: Route }> = {
    writing: { name: 'hw.writing', r: { s: 'writingNew' } },
    speaking: { name: 'hw.speaking', r: { s: 'speaking' } },
    mock: { name: 'mock.title', r: { s: 'mock' } },
  }
</script>

<Head title={g.name} sub={t('group.teacher', { nick: g.teacher })} />

<div class="sect"><h2>{t('group.homework')}</h2></div>
{#each list as h (h.id)}
  <section class="card stack">
    <p class="tag">{t(kind[h.kind].name)} · {t('group.due', { date: fmtDate(h.due) })}</p>
    <p>{h.title}</p>
    {#if h.band !== undefined}
      <p class="ok">{t('group.done')} · <span class="num">{formatBand(h.band)}</span></p>
    {:else}
      <button class="btn btn-small" onclick={() => go(kind[h.kind].r)}>{t('group.submit')}</button>
    {/if}
  </section>
{/each}

<style>
  .ok {
    color: #2e9d58;
    font-weight: 600;
  }
</style>
