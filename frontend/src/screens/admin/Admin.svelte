<script lang="ts">
  import BackBar from '../../components/BackBar.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { back } from '../../lib/nav.svelte'
  import { sections } from './sections'

  // Админка владельца (этап 17, SPEC «Админка») внутри Mini App; на сайте та же — AdminSite.svelte. Вход
  // проверяет сервер по ADMIN_IDS. Тексты — по-русски, только для владельца и помощников (DECISIONS).

  let current = $state('overview')
  const section = $derived(sections.find((s) => s.id === current)!)
</script>

<BackBar onback={back} />
<h1>Админка</h1>

<div class="seg" role="tablist">
  {#each sections as s (s.id)}
    <button role="tab" aria-selected={current === s.id} class:on={current === s.id} onclick={() => (current = s.id)}>{s.name}</button>
  {/each}
</div>

{#key current}
  {#await section.load()}
    <Skeleton rows={4} />
  {:then m}
    <m.default goto={(id: string) => (current = id)} />
  {:catch}
    <p class="error">Раздел не загрузился. Проверьте интернет.</p>
  {/await}
{/key}

<style>
  h1 {
    font-size: 24px;
  }
</style>
