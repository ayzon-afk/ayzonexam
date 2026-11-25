<script lang="ts">
  import { onMount } from 'svelte'
  import Logo from '../components/Logo.svelte'
  import { webLogin } from '../lib/api'
  import { getLang, LANGS, setLang, t, type Lang } from '../lib/i18n/index.svelte'

  // Сайт вне Telegram (DECISIONS 2026-11-24): «/» — вход на весь экран, после него — то же приложение, что в
  // боте; «/centers» — о продукте и заявка центра; «/admin» и admin.<домен> — вход в админ-панель;
  // «/cabinet» и center.<домен> — вход в кабинет центра. Грузится лениво — Mini App эту страницу не качает.

  let bot = $state('')
  let form = $state({ center: '', name: '', phone: '', students: '', message: '', website: '' })
  let loginBox = $state<HTMLDivElement | null>(null)
  let loginFailed = $state(false)
  const path = location.pathname.replace(/\/+$/, '')
  const mode: 'login' | 'centers' | 'admin' | 'cabinet' =
    location.hostname.startsWith('admin.') || path === '/admin'
      ? 'admin'
      : location.hostname.startsWith('center.') || path === '/cabinet'
        ? 'cabinet'
        : path === '/centers'
          ? 'centers'
          : 'login'

  // Виджет «Войти через Telegram»: скрипт Telegram рисует кнопку и зовёт onTelegramAuth. Домен сайта должен
  // быть указан боту в @BotFather (/setdomain), иначе виджет пишет «Bot domain invalid».
  $effect(() => {
    if (!bot || !loginBox) return
    const w = window as unknown as { onTelegramAuth?: (u: Record<string, unknown>) => void }
    w.onTelegramAuth = async (u) => {
      loginFailed = false
      try {
        await webLogin(u, getLang())
        location.reload()
      } catch {
        loginFailed = true
      }
    }
    const sc = document.createElement('script')
    sc.async = true
    sc.src = 'https://telegram.org/js/telegram-widget.js?22'
    sc.dataset.telegramLogin = bot
    sc.dataset.size = 'large'
    sc.dataset.onauth = 'onTelegramAuth(user)'
    sc.dataset.requestAccess = 'write'
    loginBox.replaceChildren(sc)
  })

  let status = $state<'' | 'sending' | 'sent' | 'bad' | 'many' | 'error'>('')

  onMount(async () => {
    const nav = navigator.language.slice(0, 2)
    if (getLang() === 'uz' && (nav === 'ru' || nav === 'en')) setLang(nav)
    try {
      const r = await fetch('/api/info')
      bot = ((await r.json()) as { bot: string }).bot
    } catch {
      // без адреса бота кнопка просто не показывается
    }
  })

  async function send(e: SubmitEvent) {
    e.preventDefault()
    status = 'sending'
    const students = parseInt(form.students, 10)
    try {
      const r = await fetch('/api/leads', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...form, students: Number.isFinite(students) ? students : null, lang: getLang() }),
      })
      status = r.ok ? 'sent' : r.status === 400 ? 'bad' : r.status === 429 ? 'many' : 'error'
    } catch {
      status = 'error'
    }
  }
</script>

{#snippet langs()}
  <nav class="langs">
    {#each LANGS as l (l.code)}
      <button class:on={getLang() === l.code} onclick={() => setLang(l.code as Lang)}>{l.code.toUpperCase()}</button>
    {/each}
  </nav>
{/snippet}

{#snippet widget()}
  <div class="widget" bind:this={loginBox}></div>
  {#if loginFailed}<p class="error">{t('land.loginFail')}</p>{/if}
{/snippet}

{#if mode !== 'centers'}
  <div class="gate">
    <div class="gate-card">
      <header>
        <Logo size={34} />
        {@render langs()}
      </header>
      <h1>{mode === 'admin' ? t('land.admin') : mode === 'cabinet' ? t('land.cabinet') : t('land.title')}</h1>
      {#if mode === 'login'}<p class="lead">{t('land.lead')}</p>{/if}
      {#if bot}
        {@render widget()}
        <p class="hint">{mode === 'cabinet' ? t('land.cabinetHint') : t('land.loginHint')}</p>
        {#if mode === 'login'}
          <p class="or">{t('land.or')}</p>
          <a class="btn btn-plain open" href="https://t.me/{bot}" target="_blank" rel="noopener">{t('land.open')}</a>
        {/if}
      {/if}
      {#if mode === 'login'}<a class="more" href="/centers">{t('land.forCenters')}</a>{/if}
      {#if mode === 'cabinet'}<a class="more" href="/centers">{t('land.forCenters')}</a>{/if}
    </div>
    <footer class="hint">{t('profile.disclaimer')}</footer>
  </div>
{:else}
  <div class="page">
    <header>
      <a href="/" class="home"><Logo size={36} /></a>
      {@render langs()}
    </header>

    <section class="hero">
      <h1>{t('land.title')}</h1>
      <p class="lead">{t('land.lead')}</p>
      {#if bot}
        <a class="btn open" href="https://t.me/{bot}" target="_blank" rel="noopener">{t('land.open')}</a>
      {/if}
      <p class="hint">{t('land.free')}</p>
    </section>

    <div class="cols">
      <section class="card">
        <h2>{t('land.students')}</h2>
        <ul>
          <li>{t('land.s1')}</li>
          <li>{t('land.s2')}</li>
          <li>{t('land.s3')}</li>
          <li>{t('land.s4')}</li>
        </ul>
      </section>
      <section class="card">
        <h2>{t('land.centers')}</h2>
        <ul>
          <li>{t('land.c1')}</li>
          <li>{t('land.c2')}</li>
          <li>{t('land.c3')}</li>
        </ul>
      </section>
    </div>

    <section class="card" id="form">
      <h2>{t('land.form')}</h2>
      {#if status === 'sent'}
        <p class="ok">{t('land.sent')}</p>
      {:else}
        <form onsubmit={send}>
          <label>{t('land.center')}<input required maxlength="120" bind:value={form.center} /></label>
          <label>{t('land.name')}<input required maxlength="80" autocomplete="name" bind:value={form.name} /></label>
          <label>{t('land.phone')}<input required type="tel" maxlength="30" autocomplete="tel" placeholder="+998" bind:value={form.phone} /></label>
          <label>{t('land.count')}<input type="number" min="0" max="100000" inputmode="numeric" bind:value={form.students} /></label>
          <label>{t('land.message')}<textarea maxlength="1000" rows="3" bind:value={form.message}></textarea></label>
          <!-- Ловушка для спам-ботов: человек это поле не видит. -->
          <input class="trap" tabindex="-1" autocomplete="off" aria-hidden="true" bind:value={form.website} />
          {#if status === 'bad'}<p class="error">{t('land.bad')}</p>{/if}
          {#if status === 'many'}<p class="error">{t('land.many')}</p>{/if}
          {#if status === 'error'}<p class="error">{t('app.error')}</p>{/if}
          <button class="btn" disabled={status === 'sending'}>{t('land.send')}</button>
        </form>
      {/if}
    </section>

    <a class="more" href="/">{t('land.back')}</a>
    <footer class="hint">{t('profile.disclaimer')}</footer>
  </div>
{/if}

<style>
  .page {
    max-width: 880px;
    margin: 0 auto;
    padding: 16px 16px 40px;
    display: grid;
    gap: 16px;
  }

  header {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 8px 12px;
  }

  .langs {
    display: flex;
    gap: 4px;
  }

  .langs button {
    padding: 6px 10px;
    border-radius: 10px;
    font-size: 14px;
    color: var(--hint);
  }

  .langs button.on {
    background: var(--section);
    color: var(--text);
    font-weight: 600;
  }

  /* Вход: одна карточка по центру экрана — как заставка приложения. */
  .gate {
    min-height: 100vh;
    display: grid;
    place-items: center;
    align-content: center;
    gap: 16px;
    padding: 24px 16px;
  }

  .gate-card {
    width: 100%;
    max-width: 460px;
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: 14px;
    background: var(--section);
    border-radius: 20px;
    padding: 28px 24px;
  }

  .gate h1 {
    font-size: 26px;
    line-height: 1.2;
  }

  .gate .lead {
    font-size: 16px;
  }

  .widget {
    min-height: 44px;
  }

  .or {
    text-align: center;
    color: var(--hint);
    font-size: 14px;
  }

  .more {
    color: var(--link);
    text-decoration: none;
    font-size: 15px;
    justify-self: start;
  }

  .home {
    color: inherit;
    text-decoration: none;
  }

  .hero {
    display: grid;
    gap: 14px;
    padding: 28px 0 8px;
  }

  h1 {
    font-size: clamp(28px, 6vw, 44px);
    line-height: 1.1;
  }

  .lead {
    font-size: 18px;
    max-width: 640px;
  }

  .open {
    display: grid;
    place-items: center;
    text-decoration: none;
  }

  .hero .open {
    max-width: 320px;
  }

  .cols {
    display: grid;
    gap: 16px;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  }

  h2 {
    font-size: 20px;
    margin-bottom: 10px;
  }

  ul {
    margin: 0;
    padding-left: 20px;
    display: grid;
    gap: 8px;
  }

  form {
    display: grid;
    gap: 12px;
    max-width: 480px;
  }

  label {
    display: grid;
    gap: 6px;
    font-size: 14px;
    color: var(--hint);
  }

  input,
  textarea {
    font: inherit;
    color: var(--text);
    background: var(--surface);
    border: 1px solid var(--separator);
    border-radius: 10px;
    padding: 12px;
    width: 100%;
  }

  .trap {
    position: absolute;
    left: -9999px;
    width: 1px;
    height: 1px;
  }

  .ok {
    color: var(--ok);
    font-weight: 600;
  }

  footer {
    text-align: center;
  }

  /* Узкий телефон (320–379): карточка входа — с меньшими полями, чтобы заголовок и кнопки не теснились. */
  @media (max-width: 379px) {
    .gate-card {
      padding: 22px 16px;
    }

    .gate h1 {
      font-size: 23px;
    }
  }
</style>
