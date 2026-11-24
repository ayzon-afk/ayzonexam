<script lang="ts">
  import { onMount } from 'svelte'
  import { api, ApiError, initAuth, type Me } from './lib/api'
  import { startBeat } from './lib/beat'
  import { initTelegram, startParam, telegramLanguage } from './lib/telegram'
  import { langFromTelegram, ready, setLang, t } from './lib/i18n/index.svelte'
  import { go, nav, routeFromOpen } from './lib/nav.svelte'
  import Loader from './components/Loader.svelte'
  import Logo from './components/Logo.svelte'
  import Onboarding from './screens/Onboarding.svelte'
  import Shell from './screens/Shell.svelte'

  type Status = 'loading' | 'outside' | 'expired' | 'error' | 'ready' | 'out'

  let status = $state<Status>('loading')
  let me = $state<Me | null>(null)
  let noDict = $state(false)

  // Кнопка бота «Открыть разбор» открывает приложение с ?check=<id>; остальные кнопки и ссылки-приглашения —
  // с ?open=… (bot.AppURL): r-<код> — друг, c-<код> — центр, d-<код> — дуэль, mock-<id>, premium, …
  const params = new URLSearchParams(location.search)
  const checkParam = Number(params.get('check'))
  const openCheck = Number.isInteger(checkParam) && checkParam > 0 ? checkParam : null
  const open = params.get('open') || startParam()

  // Ссылка-приглашение: друг или центр. Ошибка (чужой или старый код) не мешает открыть приложение.
  async function invite(m: Me): Promise<Me> {
    if (open.startsWith('r-')) await api.refer(open.slice(2)).catch(() => {})
    if (open.startsWith('c-')) {
      try {
        m = await api.joinCenter(open.slice(2))
        if (m.center) {
          nav.tab = 'profile'
          nav.stack = [{ s: 'center', id: m.center.id }]
        }
      } catch {
        // неверный код — просто открыть приложение
      }
    }
    return m
  }

  async function load() {
    status = 'loading'
    try {
      if (!(await initAuth())) {
        status = 'outside'
        return
      }
      me = await invite(await api.me())
      setLang(me.lang)
      const target = open ? routeFromOpen(open) : null
      if (openCheck) {
        nav.tab = 'exams'
        nav.stack = [{ s: 'writingCheck', id: openCheck }]
      } else if (target) {
        nav.tab = target.tab
        nav.stack = target.route ? [target.route] : []
      }
      status = 'ready'
      startBeat()
    } catch (e) {
      status = e instanceof ApiError && e.code === 'expired' ? 'expired' : 'error'
    }
  }

  function update(next: Me) {
    me = next
    setLang(next.lang)
  }

  // Выход на сайте: стираются черновик и прочее личное этого браузера, тема устройства остаётся.
  // Вход вне Telegram появится в этапе 13; до него «Войти снова» — это просто повторный вход.
  function logout() {
    try {
      for (const k of Object.keys(localStorage)) if (k.startsWith('ayzon.') && k !== 'ayzon.theme') localStorage.removeItem(k)
    } catch {
      // хранилище недоступно — стирать нечего
    }
    me = null
    nav.stack = []
    nav.tab = 'home'
    status = 'out'
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
  {:else}
    <main class="center"><Loader phrases={false} /></main>
  {/if}
{:else if status === 'ready' && me}
  {#if me.onboarded}
    <Shell {me} onchange={update} onlogout={logout} />
  {:else}
    <Onboarding ondone={onboarded} />
  {/if}
{:else}
  <main class="center">
    {#if status === 'loading'}
      <Loader />
    {:else if status === 'outside'}
      <p>{t('app.outside')}</p>
    {:else if status === 'out'}
      <p class="brand"><Logo size={40} /></p>
      <p>{t('logout.done')}</p>
      <button class="btn" onclick={load}>{t('logout.again')}</button>
    {:else if status === 'expired'}
      <p>{t('app.expired')}</p>
    {:else}
      <p>{t('app.error')}</p>
      <button class="btn" onclick={load}>{t('app.retry')}</button>
    {/if}
  </main>
{/if}

<style>
  .brand {
    justify-self: center;
  }

  .center {
    min-height: 100vh;
    display: grid;
    place-content: center;
    gap: 16px;
    padding: 24px;
    text-align: center;
  }
</style>
