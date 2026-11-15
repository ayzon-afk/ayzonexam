<script lang="ts" module>
  export type Tab = 'home' | 'exams' | 'rating' | 'community' | 'profile'
</script>

<script lang="ts">
  import { t, type Key } from '../lib/i18n/index.svelte'
  import { haptic } from '../lib/telegram'

  let { active = $bindable() }: { active: Tab } = $props()

  // Иконки — контуры 24×24, рисуются линией цвета текста.
  const tabs: { id: Tab; label: Key; path: string }[] = [
    { id: 'home', label: 'tab.home', path: 'M3 10.5 12 3l9 7.5V20a1 1 0 0 1-1 1h-5v-6h-6v6H4a1 1 0 0 1-1-1z' },
    { id: 'exams', label: 'tab.exams', path: 'M6 3h9l4 4v13a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1zm8 0v5h5M8 13h8M8 17h5' },
    { id: 'rating', label: 'tab.rating', path: 'M8 21h8M12 17v4M7 4h10v5a5 5 0 0 1-10 0zM7 6H4v1a3 3 0 0 0 3 3M17 6h3v1a3 3 0 0 1-3 3' },
    { id: 'community', label: 'tab.community', path: 'M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM2 21v-1a6 6 0 0 1 12 0v1M16 3.5a4 4 0 0 1 0 7M18 14a6 6 0 0 1 4 6v1' },
    { id: 'profile', label: 'tab.profile', path: 'M12 12a4.5 4.5 0 1 0 0-9 4.5 4.5 0 0 0 0 9zM4 21a8 8 0 0 1 16 0' },
  ]
</script>

<nav>
  {#each tabs as tab (tab.id)}
    <button
      class:on={active === tab.id}
      aria-current={active === tab.id ? 'page' : undefined}
      onclick={() => {
        if (active !== tab.id) haptic.select()
        active = tab.id
      }}
    >
      <svg viewBox="0 0 24 24" aria-hidden="true"><path d={tab.path} /></svg>
      <span>{t(tab.label)}</span>
    </button>
  {/each}
</nav>

<style>
  nav {
    position: fixed;
    inset: auto 0 0;
    display: grid;
    grid-template-columns: repeat(5, 1fr);
    height: calc(var(--tabbar-h) + var(--safe-bottom));
    padding-bottom: var(--safe-bottom);
    background: var(--bg);
    border-top: 1px solid var(--separator);
  }

  button {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 3px;
    color: var(--hint);
    min-width: 0;
  }

  .on {
    color: var(--accent);
  }

  svg {
    width: 24px;
    height: 24px;
    fill: none;
    stroke: currentColor;
    stroke-width: 1.8;
    stroke-linecap: round;
    stroke-linejoin: round;
  }

  span {
    font-size: 11px;
    font-weight: 500;
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
