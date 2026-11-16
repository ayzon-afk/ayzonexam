<script lang="ts">
  import BackBar from './BackBar.svelte'
  import Demo from './Demo.svelte'
  import { back } from '../lib/nav.svelte'

  // Шапка экрана поверх вкладок: «Назад», заголовок, пометка «демо».
  let { title, sub = '', demo = true }: { title: string; sub?: string; demo?: boolean } = $props()
</script>

<BackBar onback={back} />
<header>
  <h1>{title} {#if demo}<Demo />{/if}</h1>
  {#if sub}<p class="hint">{sub}</p>{/if}
</header>

<style>
  header {
    display: grid;
    gap: 4px;
  }

  h1 {
    font-size: 24px;
    line-height: 1.25;
  }
</style>
