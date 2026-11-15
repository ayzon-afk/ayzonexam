<script lang="ts">
  import { onMount } from 'svelte'
  import { api, ApiError, initAuth, type Me } from './lib/api'
  import { initTelegram, telegramLanguage } from './lib/telegram'
  import { langFromTelegram, setLang, t } from './lib/i18n/index.svelte'
  import Onboarding from './screens/Onboarding.svelte'
  import Shell from './screens/Shell.svelte'

  type Status = 'loading' | 'outside' | 'expired' | 'error' | 'ready'

  let status = $state<Status>('loading')
  let me = $state<Me | null>(null)

  async function load() {
    status = 'loading'
    try {
      if (!(await initAuth())) {
        status = 'outside'
        return
      }
      me = await api.me()
      setLang(me.lang)
      status = 'ready'
    } catch (e) {
      status = e instanceof ApiError && e.code === 'expired' ? 'expired' : 'error'
    }
  }

  function update(next: Me) {
    me = next
    setLang(next.lang)
  }

  onMount(() => {
    initTelegram()
    setLang(langFromTelegram(telegramLanguage()))
    load()
  })
</script>

{#if status === 'ready' && me}
  {#if me.onboarded}
    <Shell {me} onchange={update} />
  {:else}
    <Onboarding ondone={update} />
  {/if}
{:else}
  <main class="center">
    {#if status === 'loading'}
      <p class="hint">{t('app.loading')}</p>
    {:else if status === 'outside'}
      <p>{t('app.outside')}</p>
    {:else if status === 'expired'}
      <p>{t('app.expired')}</p>
    {:else}
      <p>{t('app.error')}</p>
      <button class="btn" onclick={load}>{t('app.retry')}</button>
    {/if}
  </main>
{/if}

<style>
  .center {
    min-height: 100vh;
    display: grid;
    place-content: center;
    gap: 16px;
    padding: 24px;
    text-align: center;
  }
</style>
