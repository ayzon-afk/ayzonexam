<script lang="ts">
  import { onMount } from 'svelte'
  import { api, ApiError, initAuth, type Me } from './lib/api'
  import { initTelegram, telegramLanguage } from './lib/telegram'
  import { langFromTelegram, ready, setLang, t } from './lib/i18n/index.svelte'
  import { go, nav } from './lib/nav.svelte'
  import Onboarding from './screens/Onboarding.svelte'
  import Shell from './screens/Shell.svelte'

  type Status = 'loading' | 'outside' | 'expired' | 'error' | 'ready'

  let status = $state<Status>('loading')
  let me = $state<Me | null>(null)
  let noDict = $state(false)

  // Кнопка бота «Открыть разбор» открывает приложение с ?check=<id>.
  const checkParam = Number(new URLSearchParams(location.search).get('check'))
  const openCheck = Number.isInteger(checkParam) && checkParam > 0 ? checkParam : null

  async function load() {
    status = 'loading'
    try {
      if (!(await initAuth())) {
        status = 'outside'
        return
      }
      me = await api.me()
      setLang(me.lang)
      if (openCheck) {
        nav.tab = 'exams'
        nav.stack = [{ s: 'writingCheck', id: openCheck }]
      }
      status = 'ready'
    } catch (e) {
      status = e instanceof ApiError && e.code === 'expired' ? 'expired' : 'error'
    }
  }

  function update(next: Me) {
    me = next
    setLang(next.lang)
  }

  // Конец онбординга: третий экран предлагает тест уровня — он открывается поверх главной.
  function onboarded(next: Me, levelTest: boolean) {
    update(next)
    if (levelTest) go({ s: 'practice', part: 'level' })
  }

  onMount(() => {
    initTelegram()
    setLang(langFromTelegram(telegramLanguage())).catch(() => (noDict = true))
    load()
  })
</script>

{#if !ready()}
  <!-- Словарь ещё грузится. Не загрузился — текстов нет совсем, поэтому только знак «повторить». -->
  {#if noDict}
    <main class="center"><button class="btn" aria-label="Retry" onclick={() => location.reload()}>↻</button></main>
  {/if}
{:else if status === 'ready' && me}
  {#if me.onboarded}
    <Shell {me} onchange={update} />
  {:else}
    <Onboarding ondone={onboarded} />
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
