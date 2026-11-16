<script lang="ts">
  import { onMount } from 'svelte'
  import { api, ApiError, initAuth, type Me } from './lib/api'
  import { initTelegram, telegramLanguage } from './lib/telegram'
  import { langFromTelegram, ready, setLang, t } from './lib/i18n/index.svelte'
  import { go, nav } from './lib/nav.svelte'
  import Loader, { type LoaderKind } from './components/Loader.svelte'
  import Logo from './components/Logo.svelte'
  import Onboarding from './screens/Onboarding.svelte'
  import Shell from './screens/Shell.svelte'

  type Status = 'loading' | 'outside' | 'expired' | 'error' | 'ready' | 'out'

  let status = $state<Status>('loading')
  let me = $state<Me | null>(null)
  let noDict = $state(false)

  // Сравнение вариантов анимации загрузки — только при разработке: ?preview=loaders.
  const preview = import.meta.env.DEV && new URLSearchParams(location.search).get('preview') === 'loaders'
  const kinds: LoaderKind[] = ['penguin', 'chart']

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

{#if preview && ready()}
  <main class="center preview">
    {#each kinds as k, i (k)}
      <div class="card"><p class="hint">{i === 0 ? 'A' : 'B'}</p><Loader kind={k} /></div>
    {/each}
  </main>
{:else if !ready()}
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

  .preview {
    grid-auto-flow: column;
    gap: 16px;
  }

  .preview .card {
    display: grid;
    gap: 8px;
    width: 240px;
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
