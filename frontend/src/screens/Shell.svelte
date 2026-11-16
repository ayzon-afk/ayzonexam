<script lang="ts">
  import type { Component } from 'svelte'
  import TabBar from '../components/TabBar.svelte'
  import Home from '../tabs/Home.svelte'
  import Exams from '../tabs/Exams.svelte'
  import Rating from '../tabs/Rating.svelte'
  import Community from '../tabs/Community.svelte'
  import Profile from '../tabs/Profile.svelte'
  import WritingNew from './writing/New.svelte'
  import WritingResult from './writing/Result.svelte'
  import type { Me } from '../lib/api'
  import { t } from '../lib/i18n/index.svelte'
  import { back, current, nav, replace } from '../lib/nav.svelte'

  let { me, onchange }: { me: Me; onchange: (me: Me) => void } = $props()

  type Lazy = () => Promise<{ default: Component<any> }>

  // Экраны поверх вкладок грузятся при первом открытии: телефон не качает то, куда ученик не заходил.
  const screens: Record<string, Lazy> = {
    news: () => import('./demo/News.svelte'),
    post: () => import('./demo/Post.svelte'),
    tournament: () => import('./demo/Tournament.svelte'),
    center: () => import('./demo/Center.svelte'),
    practice: () => import('./demo/Practice.svelte'),
    speaking: () => import('./demo/Speaking.svelte'),
    mock: () => import('./demo/Mock.svelte'),
    report: () => import('./demo/Report.svelte'),
    duel: () => import('./demo/Duel.svelte'),
    teacher: () => import('./demo/Teacher.svelte'),
    review: () => import('./demo/Review.svelte'),
    group: () => import('./demo/Group.svelte'),
    premium: () => import('./demo/Premium.svelte'),
    history: () => import('./demo/History.svelte'),
  }

  const top = $derived(current())
</script>

<div class="shell">
  <main class:sub={!!top}>
    {#if !top}
      {#if nav.tab === 'home'}
        <Home {me} />
      {:else if nav.tab === 'exams'}
        <Exams />
      {:else if nav.tab === 'rating'}
        <Rating {me} />
      {:else if nav.tab === 'community'}
        <Community />
      {:else}
        <Profile {me} {onchange} />
      {/if}
    {:else if top.s === 'writingNew'}
      <WritingNew onback={back} oncreated={(id) => replace({ s: 'writingCheck', id })} />
    {:else if top.s === 'writingCheck'}
      {#key top.id}
        <WritingResult id={top.id} onback={back} onagain={() => replace({ s: 'writingNew' })} />
      {/key}
    {:else}
      {#key top}
        {#await screens[top.s]!()}
          <p class="hint">{t('app.loading')}</p>
        {:then m}
          <m.default r={top} {me} />
        {:catch}
          <p class="error">{t('app.error')}</p>
          <button class="btn btn-plain" onclick={back}>{t('onb.back')}</button>
        {/await}
      {/key}
    {/if}
  </main>

  <TabBar sub={!!top} />
</div>

<style>
  main {
    min-height: 100vh;
    max-width: 720px;
    margin: 0 auto;
    padding: calc(var(--safe-top) + 16px) 16px calc(var(--tabbar-h) + var(--safe-bottom) + 16px);
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .sub {
    padding-bottom: calc(var(--safe-bottom) + 24px);
  }

  @media (min-width: 900px) {
    .shell {
      padding-left: var(--side-w);
    }

    main,
    .sub {
      padding: 32px 24px 48px;
    }
  }
</style>
