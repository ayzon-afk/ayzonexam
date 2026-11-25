<script lang="ts">
  import Logo from '../../components/Logo.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { ApiError, type Me } from '../../lib/api'
  import { t } from '../../lib/i18n/index.svelte'
  import { cab, sectionsFor, type Cabinet } from './cabinet'

  // Кабинет центра на сайте (DECISIONS 2026-11-25): /cabinet или center.<домен>, во всю ширину, как
  // админ-панель владельца: разделы слева, на телефоне — полосой сверху; раздел — в адресе (#students).
  let { me, onlogout }: { me: Me; onlogout: () => void } = $props()

  let data = $state<Cabinet | null>(null)
  let denied = $state(false)
  let failed = $state(false)

  async function load() {
    failed = false
    try {
      data = await cab.get()
    } catch (e) {
      if (e instanceof ApiError && e.status === 403) denied = true
      else failed = true
    }
  }
  load()

  const list = $derived(data ? sectionsFor(data.role) : [])
  const fromHash = () => location.hash.slice(1) || 'overview'
  let current = $state(fromHash())
  const section = $derived(list.find((s) => s.id === current) ?? list[0])

  function goto(id: string) {
    current = id
    history.replaceState(null, '', `#${id}`)
    window.scrollTo(0, 0)
  }

  $effect(() => {
    const on = () => (current = fromHash())
    addEventListener('hashchange', on)
    return () => removeEventListener('hashchange', on)
  })
</script>

{#if denied}
  <main class="alone">
    <Logo size={40} />
    <p>{t('cab.noAccess')}</p>
    <a class="btn" href="/">{t('cab.toApp')}</a>
    <button class="link" onclick={onlogout}>{t('cab.logout')}</button>
  </main>
{:else if failed}
  <main class="alone">
    <p>{t('cab.loadFail')}</p>
    <button class="btn" onclick={load}>{t('app.retry')}</button>
  </main>
{:else}
  <div class="site">
    <aside>
      <div class="brand"><Logo size={30} /></div>
      <div class="who">
        <strong>{data?.center.name ?? me.center?.name ?? ''}</strong>
        <span class="hint">{data?.role === 'admin' ? t('cab.roleAdmin') : t('cab.roleTeacher')} · {me.nickname}</span>
      </div>
      <nav aria-label={t('cab.sections')}>
        {#each list as s (s.id)}
          <button class:on={section?.id === s.id} aria-current={section?.id === s.id ? 'page' : undefined} onclick={() => goto(s.id)}>
            {t(s.name)}
            {#if s.id === 'requests' && data?.center.pending}<span class="badge">{data.center.pending}</span>{/if}
          </button>
        {/each}
      </nav>
      <div class="foot">
        <a href="/">{t('cab.toApp')}</a>
        <button onclick={onlogout}>{t('cab.logout')}</button>
      </div>
    </aside>

    <main>
      {#if !data || !section}
        <Skeleton rows={6} />
      {:else}
        <h1>{t(section.name)}</h1>
        {#key section.id}
          {#await section.load()}
            <Skeleton rows={6} />
          {:then m}
            <m.default {me} {data} onchange={(c: Cabinet) => (data = c)} {goto} />
          {:catch}
            <p class="error">{t('cab.loadFail')}</p>
          {/await}
        {/key}
      {/if}
    </main>
  </div>
{/if}

<style>
  .site {
    min-height: 100vh;
    display: grid;
    grid-template-columns: 248px minmax(0, 1fr);
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
    display: grid;
    gap: 2px;
    padding: 0 10px;
    overflow-wrap: anywhere;
  }

  nav {
    display: grid;
    gap: 2px;
    overflow-y: auto;
  }

  nav button {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    min-height: 40px;
    padding: 9px 10px;
    border-radius: 10px;
    text-align: left;
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

  .badge {
    min-width: 22px;
    padding: 1px 7px;
    border-radius: 999px;
    background: var(--danger);
    color: #fff;
    font-size: 12px;
    font-weight: 700;
    text-align: center;
  }

  .foot {
    margin-top: auto;
    display: grid;
    gap: 8px;
    padding: 0 10px;
    font-size: 14px;
  }

  .foot a,
  .foot button,
  .link {
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
    gap: 14px;
  }

  h1 {
    font-size: 26px;
  }

  .alone {
    min-height: 100vh;
    max-width: 440px;
    margin: 0 auto;
    padding: 24px 16px;
    display: grid;
    place-content: center;
    justify-items: center;
    gap: 16px;
    text-align: center;
  }

  .alone .btn {
    display: grid;
    place-items: center;
    text-decoration: none;
  }

  /* Планшет и телефон: шапка с названием, разделы — полосой, листаются вбок. */
  @media (max-width: 899px) {
    .site {
      grid-template-columns: minmax(0, 1fr);
    }

    aside {
      position: sticky;
      z-index: 2;
      height: auto;
      display: grid;
      grid-template-columns: minmax(0, 1fr) auto;
      align-items: center;
      gap: 10px 12px;
      padding: 12px 16px 10px;
      border-right: 0;
      border-bottom: 1px solid var(--separator);
    }

    .brand {
      display: none;
    }

    .who {
      padding: 0;
    }

    .who strong,
    .who .hint {
      overflow: hidden;
      white-space: nowrap;
      text-overflow: ellipsis;
    }

    nav {
      grid-column: 1 / -1;
      grid-row: 2;
      display: flex;
      gap: 6px;
      overflow-x: auto;
      scrollbar-width: none;
    }

    nav button {
      flex: none;
      min-height: 36px;
      padding: 0 14px;
      border-radius: 999px;
      background: var(--surface);
      white-space: nowrap;
      font-size: 14px;
    }

    .foot {
      margin: 0;
      padding: 0;
      display: flex;
      gap: 12px;
      font-size: 13px;
    }

    .foot a {
      display: none;
    }

    main {
      padding: 20px 16px 48px;
    }

    h1 {
      font-size: 22px;
    }
  }
</style>
