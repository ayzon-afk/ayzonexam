<script lang="ts">
  import Logo from '../../components/Logo.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { sections } from './sections'

  // Админ-панель владельца на сайте (DECISIONS 2026-11-24): /admin или admin.<домен> — во всю ширину, разделы
  // слева, раздел — в адресе (#usage), чтобы обновление страницы его не теряло. Разделы те же, что в Mini App.
  let { name, onlogout }: { name: string; onlogout: () => void } = $props()

  const fromHash = () => {
    const id = location.hash.slice(1)
    return sections.some((s) => s.id === id) ? id : 'overview'
  }
  let current = $state(fromHash())
  const section = $derived(sections.find((s) => s.id === current)!)

  function goto(id: string) {
    current = id
    history.replaceState(null, '', `#${id}`)
  }

  $effect(() => {
    const on = () => (current = fromHash())
    addEventListener('hashchange', on)
    return () => removeEventListener('hashchange', on)
  })
</script>

<div class="site">
  <aside>
    <div class="brand"><Logo size={30} /></div>
    <p class="who hint">Владелец · {name}</p>
    <nav aria-label="Разделы">
      {#each sections as s (s.id)}
        <button class:on={current === s.id} aria-current={current === s.id ? 'page' : undefined} onclick={() => goto(s.id)}>{s.name}</button>
      {/each}
    </nav>
    <div class="foot">
      <a href="/">Открыть приложение</a>
      <button onclick={onlogout}>Выйти</button>
    </div>
  </aside>

  <main>
    <h1>{section.name}</h1>
    {#key current}
      {#await section.load()}
        <Skeleton rows={6} />
      {:then m}
        <m.default {goto} />
      {:catch}
        <p class="error">Раздел не загрузился. Проверьте интернет.</p>
      {/await}
    {/key}
  </main>
</div>

<style>
  .site {
    min-height: 100vh;
    display: grid;
    grid-template-columns: 240px minmax(0, 1fr);
  }

  aside {
    position: sticky;
    top: 0;
    height: 100vh;
    display: flex;
    flex-direction: column;
    gap: 16px;
    padding: 24px 16px;
    background: var(--section);
    border-right: 1px solid var(--separator);
  }

  .who {
    padding: 0 10px;
  }

  nav {
    display: grid;
    gap: 2px;
    overflow-y: auto;
  }

  nav button {
    text-align: left;
    padding: 9px 10px;
    border-radius: 10px;
    color: var(--text);
  }

  nav button:hover {
    background: var(--surface);
  }

  nav button.on {
    background: var(--accent);
    color: var(--accent-text);
    font-weight: 600;
  }

  .foot {
    margin-top: auto;
    display: grid;
    gap: 8px;
    padding: 0 10px;
    font-size: 14px;
  }

  .foot a,
  .foot button {
    color: var(--link);
    text-decoration: none;
    text-align: left;
  }

  main {
    min-width: 0;
    max-width: 1200px;
    width: 100%;
    padding: 32px 32px 64px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  h1 {
    font-size: 26px;
  }

  /* Узкий экран: разделы — полосой сверху. */
  @media (max-width: 899px) {
    .site {
      grid-template-columns: minmax(0, 1fr);
    }

    aside {
      position: static;
      height: auto;
      padding: 16px;
      border-right: 0;
      border-bottom: 1px solid var(--separator);
    }

    nav {
      display: flex;
      overflow-x: auto;
    }

    nav button {
      white-space: nowrap;
    }

    .foot {
      display: flex;
      gap: 16px;
    }

    main {
      padding: 20px 16px 48px;
    }
  }
</style>
