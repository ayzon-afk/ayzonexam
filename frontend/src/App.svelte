<script lang="ts">
  import { onMount } from 'svelte'
  import { api, ApiError, dropWebSession, initAuth, webSession, type Me } from './lib/api'
  import { startBeat } from './lib/beat'
  import { initTelegram, inTelegram, startParam, telegramLanguage } from './lib/telegram'
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

  // Админ-панель на сайте: /admin или admin.<домен>, только в браузере (в Mini App — админка из профиля).
  // Кабинет центра — так же: /cabinet или center.<домен> (DECISIONS 2026-11-25).
  const sitePath = (name: string) => [`/${name}`, `/${name}/`].includes(location.pathname)
  const adminSite = !inTelegram() && (location.hostname.startsWith('admin.') || sitePath('admin'))
  const cabinetSite = !inTelegram() && !adminSite && (location.hostname.startsWith('center.') || sitePath('cabinet'))

  const staff = (m: Me) => m.center?.status === 'member' && m.center.role === 'admin'

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
      } else if (staff(me)) {
        // Аккаунт центра и учитель без центра: главный экран — кабинет (аналитика, приём учеников, оплата);
        // приложение ученика — «Назад» (DECISIONS 2026-11-25).
        nav.tab = 'profile'
        nav.stack = [{ s: 'cabinet' }]
      }
      status = 'ready'
      startBeat()
    } catch (e) {
      // Пропуск сайта устарел или отозван — снова страница входа, а не «закройте приложение».
      if (!inTelegram() && webSession() && e instanceof ApiError && e.status === 401) {
        dropWebSession()
        status = 'outside'
        return
      }
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
    if (levelTest) go({ s: 'level' })
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
  {#if adminSite && me.admin}
    {#await import('./screens/admin/AdminSite.svelte')}
      <main class="center"><Loader phrases={false} /></main>
    {:then m}
      <m.default name={me.first_name || me.nickname} onlogout={logout} />
    {:catch}
      <main class="center"><p>{t('app.error')}</p></main>
    {/await}
  {:else if cabinetSite}
    {#await import('./screens/cabinet/CabinetSite.svelte')}
      <main class="center"><Loader phrases={false} /></main>
    {:then m}
      <m.default {me} onlogout={logout} />
    {:catch}
      <main class="center"><p>{t('app.error')}</p></main>
    {/await}
  {:else if me.onboarded || staff(me) || open.startsWith('apply')}
    <Shell {me} onchange={update} onlogout={logout} />
  {:else}
    <Onboarding ondone={onboarded} />
  {/if}
{:else if status === 'outside'}
  <!-- Вне Telegram — открытая страница о продукте (DECISIONS 2026-11-24); Mini App её не качает. -->
  {#await import('./screens/Landing.svelte')}
    <main class="center"><Loader phrases={false} /></main>
  {:then m}
    <m.default />
  {:catch}
    <main class="center"><p>{t('app.outside')}</p></main>
  {/await}
{:else}
  <main class="center">
    {#if status === 'loading'}
      <Loader />
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
