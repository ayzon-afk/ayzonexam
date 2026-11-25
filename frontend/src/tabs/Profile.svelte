<script lang="ts">
  import { onMount } from 'svelte'
  import BandPicker from '../components/BandPicker.svelte'
  import Chart from '../components/Chart.svelte'
  import { api, ApiError, type Me, type Stats } from '../lib/api'
  import { fmtDate } from '../lib/date'
  import { examDateRange, formatBand } from '../lib/exam'
  import { LANGS, setLang, t, type Key, type Lang } from '../lib/i18n/index.svelte'
  import { go } from '../lib/nav.svelte'
  import { closeApp, haptic, inTelegram } from '../lib/telegram'
  import { applyTheme, loadTheme, type Theme } from '../lib/theme'

  let { me, onchange, onlogout }: { me: Me; onchange: (me: Me) => void; onlogout: () => void } = $props()

  let stats = $state<Stats | null>(null)
  onMount(() => {
    api.stats().then(
      (s) => (stats = s),
      () => {},
    )
  })

  const badges: { id: string; title: Key }[] = [
    { id: 'first6', title: 'badge.first6' },
    { id: 'first7', title: 'badge.first7' },
    { id: 'duel10', title: 'badge.duel10' },
    { id: 'tour', title: 'badge.tour' },
    { id: 'mock10', title: 'badge.mock10' },
    { id: 'essay100', title: 'badge.essay100' },
  ]

  // Ник — публичное имя в рейтинге: a–z, 0–9, «_» и «.», 3–20 знаков (как api.validNick).
  let nick = $state('')
  let nickEdit = $state(false)
  let nickError = $state<Key | ''>('')
  async function saveNick() {
    nickError = ''
    try {
      onchange(await api.updateMe({ nickname: nick }))
      nickEdit = false
      haptic.success()
    } catch (e) {
      haptic.error()
      const code = e instanceof ApiError ? e.code : ''
      nickError = code === 'nick_taken' ? 'nick.taken' : code === 'bad_nickname' ? 'nick.bad' : 'app.error'
    }
  }

  async function setCity(city: 'tashkent' | 'other') {
    if (city === me.city) return
    haptic.select()
    try {
      onchange(await api.updateMe({ city }))
    } catch {
      status = 'error'
    }
  }

  const telegram = inTelegram()
  // На сайте «как в системе» — светлая (lib/theme), поэтому выбор только из двух.
  let theme = $state<Theme>(!telegram && loadTheme() === 'auto' ? 'light' : loadTheme())
  const themes: { id: Theme; label: Key }[] = [
    ...(telegram ? [{ id: 'auto' as Theme, label: 'theme.telegram' as Key }] : []),
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
  <div class="grow">
    <h1>{me.first_name || me.nickname}</h1>
    {#if nickEdit}
      <div class="nick">
        <input bind:value={nick} maxlength="20" autocapitalize="off" autocomplete="off" />
        <button class="btn btn-small" onclick={saveNick}>{t('profile.save')}</button>
      </div>
      <p class="hint">{nickError ? t(nickError) : t('nick.hint')}</p>
    {:else}
      <button class="hint" onclick={() => ((nick = me.nickname), (nickEdit = true))}>{me.nickname} ✎</button>
    {/if}
  </div>
</div>

<section class="card">
  <div class="stats">
    <div>
      <p class="hint">{t('profile.current')}</p>
      <p class="num big">{stats?.current == null ? '—' : formatBand(stats.current)}</p>
      {#if stats?.place}<p class="tag">{t('profile.place', { n: stats.place })}</p>{/if}
    </div>
    <div>
      <p class="hint">{t('home.goal')}</p>
      <p class="num big">{me.goal_band === null ? '—' : formatBand(me.goal_band)}</p>
    </div>
  </div>
  {#if stats && stats.progress.length >= 2}
    <Chart points={stats.progress} />
  {:else}
    <p class="hint">{t('profile.noProgress')}</p>
  {/if}
  <button class="more" onclick={() => go({ s: 'history' })}>{t('history.title')} ›</button>
</section>

<section class="card">
  <h2>{t('profile.badges')}</h2>
  <ul class="badges">
    {#each badges as b (b.id)}
      <li class:off={!stats?.badges[b.id]}>
        <span class="medal" aria-hidden="true">★</span>
        <span>{t(b.title)}</span>
      </li>
    {/each}
  </ul>
</section>

<ul class="card rows">
  <li>
    <button class="row" onclick={() => go(me.center?.status === 'member' ? { s: 'center', id: me.center.id } : { s: 'centers' })}>
      <span class="grow">{t('profile.center')}</span>
      <span class="hint">{me.center ? me.center.name + (me.center.status === 'pending' ? ' ⏳' : '') : t('center.choose')}</span>
      <span class="chev" aria-hidden="true">›</span>
    </button>
  </li>
  {#if me.center?.status === 'member' && me.center.role}
    <li>
      <button class="row" onclick={() => go({ s: 'cabinet' })}>
        <span class="grow">{t('cab.title')}<span class="tag">{t('cab.openSub')}</span></span>
        <span class="chev" aria-hidden="true">›</span>
      </button>
    </li>
  {/if}
  <li>
    <button class="row" onclick={() => go({ s: 'premium' })}>
      <span class="grow">Premium</span>
      <span class="hint">{me.plan?.premium && me.plan.until ? t('premium.until', { date: fmtDate(me.plan.until) }) : t('premium.off')}</span>
      <span class="chev" aria-hidden="true">›</span>
    </button>
  </li>
  <li>
    <button class="row" onclick={() => go({ s: 'friends' })}>
      <span class="grow">{t('profile.friends')}</span>
      <span class="hint">{t('premium.invited', { n: stats?.invited ?? 0 })}</span>
      <span class="chev" aria-hidden="true">›</span>
    </button>
  </li>
</ul>

<section class="card">
  <h2>{t('profile.city')}</h2>
  <div class="langs two" role="radiogroup">
    <button role="radio" aria-checked={me.city === 'tashkent'} class:on={me.city === 'tashkent'} onclick={() => setCity('tashkent')}>{t('rating.city')}</button>
    <button role="radio" aria-checked={me.city === 'other'} class:on={me.city === 'other'} onclick={() => setCity('other')}>{t('city.other')}</button>
  </div>
</section>

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

<ul class="card rows">
  {#if me.admin}
    <li>
      <button class="row" onclick={() => go({ s: 'admin' })}>
        <span class="grow">{t('profile.admin')}</span>
        <span class="chev" aria-hidden="true">›</span>
      </button>
    </li>
  {/if}
  <li>
    <button class="row" onclick={() => go({ s: 'support' })}>
      <span class="grow">{t('support.title')}</span>
      <span class="chev" aria-hidden="true">›</span>
    </button>
  </li>
</ul>

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

  .nick {
    display: flex;
    gap: 8px;
    margin-top: 4px;
  }

  .nick input {
    flex: 1;
    min-width: 0;
    min-height: 36px;
    padding: 0 10px;
    border: 1px solid var(--separator);
    border-radius: 10px;
    background: var(--section);
    color: var(--text);
    font: inherit;
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
