<script lang="ts">
  import { onMount } from 'svelte'
  import Logo from '../components/Logo.svelte'
  import { getLang, LANGS, setLang, t, type Lang } from '../lib/i18n/index.svelte'

  // Открытая страница вне Telegram (хакатон, DECISIONS 2026-11-24): что это, для кого, вход через бота и
  // заявка учебного центра. Грузится лениво — Mini App её не качает.

  let bot = $state('')
  let form = $state({ center: '', name: '', phone: '', students: '', message: '', website: '' })
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

<div class="page">
  <header>
    <Logo size={36} />
    <nav class="langs">
      {#each LANGS as l (l.code)}
        <button class:on={getLang() === l.code} onclick={() => setLang(l.code as Lang)}>{l.code.toUpperCase()}</button>
      {/each}
    </nav>
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

  <section class="card" id="centers">
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

  <footer class="hint">{t('profile.disclaimer')}</footer>
</div>

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
    align-items: center;
    justify-content: space-between;
    gap: 12px;
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
    max-width: 320px;
    text-decoration: none;
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
</style>
