<script lang="ts">
  import TabBar, { type Tab } from '../components/TabBar.svelte'
  import Soon from '../components/Soon.svelte'
  import Home from '../tabs/Home.svelte'
  import Exams from '../tabs/Exams.svelte'
  import Profile from '../tabs/Profile.svelte'
  import WritingNew from './writing/New.svelte'
  import WritingResult from './writing/Result.svelte'
  import type { Me } from '../lib/api'

  // openCheck — проверка из кнопки бота (?check=<id>): приложение открывается сразу на разборе.
  let { me, onchange, openCheck }: { me: Me; onchange: (me: Me) => void; openCheck: number | null } = $props()

  // Экран поверх вкладок: без панели вкладок, «Назад» возвращает во вкладку.
  type Sub = { kind: 'new' } | { kind: 'check'; id: number }

  // svelte-ignore state_referenced_locally
  let tab = $state<Tab>(openCheck ? 'exams' : 'home')
  // svelte-ignore state_referenced_locally
  let sub = $state<Sub | null>(openCheck ? { kind: 'check', id: openCheck } : null)

  function open(next: Sub | null) {
    sub = next
    window.scrollTo(0, 0)
  }
</script>

{#if sub}
  <main class="sub">
    {#if sub.kind === 'new'}
      <WritingNew onback={() => open(null)} oncreated={(id) => open({ kind: 'check', id })} />
    {:else}
      {#key sub.id}
        <WritingResult id={sub.id} onback={() => open(null)} onagain={() => open({ kind: 'new' })} />
      {/key}
    {/if}
  </main>
{:else}
  <main>
    {#if tab === 'home'}
      <Home {me} />
    {:else if tab === 'exams'}
      <Exams onnew={() => open({ kind: 'new' })} onopen={(id) => open({ kind: 'check', id })} />
    {:else if tab === 'rating'}
      <Soon title="tab.rating" text="rating.soon" />
    {:else if tab === 'community'}
      <Soon title="tab.community" text="community.soon" />
    {:else}
      <Profile {me} {onchange} />
    {/if}
  </main>

  <TabBar bind:active={tab} />
{/if}

<style>
  main {
    min-height: 100vh;
    padding: calc(var(--safe-top) + 16px) 16px calc(var(--tabbar-h) + var(--safe-bottom) + 16px);
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .sub {
    padding-bottom: calc(var(--safe-bottom) + 24px);
  }
</style>
