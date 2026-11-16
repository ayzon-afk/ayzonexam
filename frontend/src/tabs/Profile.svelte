<script lang="ts">
  import BandPicker from '../components/BandPicker.svelte'
  import Chart from '../components/Chart.svelte'
  import Demo from '../components/Demo.svelte'
  import { api, ApiError, type Me } from '../lib/api'
  import { badges, centerById, mine, MY_CENTER } from '../lib/demo'
  import { examDateRange, formatBand } from '../lib/exam'
  import { LANGS, setLang, t, type Key, type Lang } from '../lib/i18n/index.svelte'
  import { go } from '../lib/nav.svelte'
  import { closeApp, haptic, inTelegram } from '../lib/telegram'
  import { applyTheme, loadTheme, type Theme } from '../lib/theme'

  let { me, onchange, onlogout }: { me: Me; onchange: (me: Me) => void; onlogout: () => void } = $props()

  const telegram = inTelegram()
  let theme = $state<Theme>(loadTheme())
  const themes: { id: Theme; label: Key }[] = [
    { id: 'auto', label: telegram ? 'theme.telegram' : 'theme.system' },
    { id: 'light', label: 'theme.light' },
    { id: 'dark', label: 'theme.dark' },
  ]

  function pickTheme(next: Theme) {
    haptic.select()
    theme = next
    applyTheme(next)
  }

  // Выход: в Telegram — закрыть приложение (аккаунт и есть Telegram), на сайте — спросить и выйти.
  let confirmOut = $state(false)

  // Черновик правок: заполняется из профиля один раз при открытии вкладки.
  // svelte-ignore state_referenced_locally
  let goal = $state<number | null>(me.goal_band)
  // svelte-ignore state_referenced_locally
  let examDate = $state(me.exam_date ?? '')
  let saving = $state(false)
  let status = $state<'' | 'saved' | 'error' | 'date'>('')

  const range = examDateRange(new Date())
  const dirty = $derived(goal !== me.goal_band || (examDate || null) !== me.exam_date)

  async function save() {
    saving = true
    status = ''
    try {
      onchange(await api.updateMe({ goal_band: goal, exam_date: examDate || null }))
      haptic.success()
      status = 'saved'
    } catch (e) {
      haptic.error()
      status = e instanceof ApiError && e.code === 'bad_exam_date' ? 'date' : 'error'
    } finally {
      saving = false
    }
  }

  // Язык меняется сразу: интерфейс переключается до ответа сервера и возвращается, если тот отказал.
  async function changeLang(lang: Lang) {
    if (lang === me.lang) return
    const prev = me.lang
    setLang(lang)
    haptic.select()
    try {
      onchange(await api.updateMe({ lang }))
    } catch {
      setLang(prev)
      status = 'error'
    }
  }
</script>

<div class="who">
  {#if me.photo_url}
    <img src={me.photo_url} alt="" width="56" height="56" referrerpolicy="no-referrer" />
  {:else}
    <div class="avatar" aria-hidden="true">{(me.first_name || me.nickname).slice(0, 1).toUpperCase()}</div>
  {/if}
  <div>
    <h1>{me.first_name || me.nickname}</h1>
    <p class="hint">@{me.nickname}</p>
  </div>
</div>

<section class="card">
  <div class="stats">
    <div>
      <p class="hint">{t('profile.current')} <Demo /></p>
      <p class="num big">{formatBand(mine.band)}</p>
    </div>
    <div>
      <p class="hint">{t('home.goal')}</p>
      <p class="num big">{me.goal_band === null ? '—' : formatBand(me.goal_band)}</p>
    </div>
  </div>
  <Chart points={mine.progress} />
</section>

<section class="card">
  <h2>{t('profile.badges')} <Demo /></h2>
  <ul class="badges">
    {#each badges as b (b.title)}
      <li class:off={!b.got}>
        <span class="medal" aria-hidden="true">★</span>
        <span>{t(b.title)}</span>
      </li>
    {/each}
  </ul>
</section>

<ul class="card rows">
  <li>
    <button class="row" onclick={() => go({ s: 'center', id: MY_CENTER })}>
      <span class="grow">{t('profile.center')} <Demo /></span>
      <span class="hint">{centerById(MY_CENTER).name}</span>
      <span class="chev" aria-hidden="true">›</span>
    </button>
  </li>
  <li>
    <button class="row" onclick={() => go({ s: 'premium' })}>
      <span class="grow">Premium <Demo /></span>
      <span class="chev" aria-hidden="true">›</span>
    </button>
  </li>
  <li>
    <button class="row" onclick={() => go({ s: 'premium' })}>
      <span class="grow">{t('profile.friends')} <Demo /></span>
      <span class="hint">{t('premium.invited', { n: mine.invited })}</span>
      <span class="chev" aria-hidden="true">›</span>
    </button>
  </li>
</ul>

<section class="card">
  <h2>{t('profile.goal')}</h2>
  <BandPicker bind:value={goal} />
</section>

<section class="card">
  <h2>{t('profile.examDate')}</h2>
  <input type="date" min={range.min} max={range.max} bind:value={examDate} />
  {#if !examDate}<p class="hint">{t('profile.notSet')}</p>{/if}
</section>

<button class="btn" disabled={!dirty || saving} onclick={save}>{t('profile.save')}</button>
{#if status === 'saved' && !dirty}<p class="hint center">{t('profile.saved')}</p>{/if}
{#if status === 'date'}<p class="error center">{t('error.date')}</p>{/if}
{#if status === 'error'}<p class="error center">{t('app.error')}</p>{/if}

<section class="card">
  <h2>{t('profile.language')}</h2>
  <div class="langs" role="radiogroup">
    {#each LANGS as l (l.code)}
      <button role="radio" aria-checked={me.lang === l.code} class:on={me.lang === l.code} onclick={() => changeLang(l.code)}
        >{l.name}</button
      >
    {/each}
  </div>
</section>

<section class="card">
  <h2>{t('profile.theme')}</h2>
  <div class="langs" role="radiogroup">
    {#each themes as th (th.id)}
      <button role="radio" aria-checked={theme === th.id} class:on={theme === th.id} onclick={() => pickTheme(th.id)}>{t(th.label)}</button>
    {/each}
  </div>
</section>

{#if telegram}
  <button class="btn btn-plain out" onclick={closeApp}>{t('logout.close')}</button>
{:else if confirmOut}
  <section class="card">
    <p>{t('logout.confirm')}</p>
    <div class="langs two">
      <button onclick={() => (confirmOut = false)}>{t('logout.cancel')}</button>
      <button class="danger" onclick={onlogout}>{t('logout.title')}</button>
    </div>
  </section>
{:else}
  <button class="btn btn-plain out" onclick={() => (confirmOut = true)}>{t('logout.title')}</button>
{/if}

<p class="hint center">{t('profile.disclaimer')}</p>

<style>
  .who {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  img,
  .avatar {
    width: 56px;
    height: 56px;
    border-radius: 50%;
    flex: none;
  }

  .avatar {
    display: grid;
    place-items: center;
    background: var(--accent);
    color: var(--accent-text);
    font-size: 24px;
    font-weight: 700;
  }

  h1 {
    font-size: 22px;
  }

  h2 {
    font-size: 15px;
    font-weight: 600;
    margin-bottom: 12px;
  }

  section .hint {
    margin-top: 8px;
  }

  .langs {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 8px;
  }

  .langs button {
    min-height: 44px;
    border-radius: 12px;
    background: var(--surface);
    font-size: 14px;
    font-weight: 500;
  }

  .langs.two {
    grid-template-columns: 1fr 1fr;
    margin-top: 12px;
  }

  .langs .danger {
    background: var(--danger);
    color: #fff;
  }

  .out {
    color: var(--danger);
  }

  .langs .on {
    background: var(--accent);
    color: var(--accent-text);
  }

  .center {
    text-align: center;
  }

  .stats {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
    margin-bottom: 8px;
  }

  .big {
    font-size: 32px;
    line-height: 1.1;
  }

  .badges {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 12px 8px;
  }

  .badges li {
    display: grid;
    justify-items: center;
    gap: 4px;
    text-align: center;
    font-size: 12px;
  }

  .medal {
    display: grid;
    place-items: center;
    width: 48px;
    height: 48px;
    border-radius: 50%;
    background: linear-gradient(135deg, #f7c948, #e08e0b);
    color: #fff;
    font-size: 22px;
  }

  .badges .off {
    color: var(--hint);
  }

  .badges .off .medal {
    background: var(--surface);
    color: var(--hint);
  }
</style>
