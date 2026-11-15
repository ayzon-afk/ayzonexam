<script lang="ts">
  import TabBar, { type Tab } from '../components/TabBar.svelte'
  import Soon from '../components/Soon.svelte'
  import Home from '../tabs/Home.svelte'
  import Profile from '../tabs/Profile.svelte'
  import type { Me } from '../lib/api'

  let { me, onchange }: { me: Me; onchange: (me: Me) => void } = $props()

  let tab = $state<Tab>('home')
</script>

<main>
  {#if tab === 'home'}
    <Home {me} />
  {:else if tab === 'exams'}
    <Soon title="tab.exams" text="exams.soon" />
  {:else if tab === 'rating'}
    <Soon title="tab.rating" text="rating.soon" />
  {:else if tab === 'community'}
    <Soon title="tab.community" text="community.soon" />
  {:else}
    <Profile {me} {onchange} />
  {/if}
</main>

<TabBar bind:active={tab} />

<style>
  main {
    min-height: 100vh;
    padding: calc(var(--safe-top) + 16px) 16px calc(var(--tabbar-h) + var(--safe-bottom) + 16px);
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
</style>
