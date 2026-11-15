<script lang="ts">
  import BandPicker from '../components/BandPicker.svelte'
  import { api, ApiError, type Me } from '../lib/api'
  import { examDateRange } from '../lib/exam'
  import { LANGS, setLang, t, type Lang } from '../lib/i18n/index.svelte'
  import { haptic } from '../lib/telegram'

  let { me, onchange }: { me: Me; onchange: (me: Me) => void } = $props()

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

  .langs .on {
    background: var(--accent);
    color: var(--accent-text);
  }

  .center {
    text-align: center;
  }
</style>
