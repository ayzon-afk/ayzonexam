<script lang="ts">
  import { onMount, untrack } from 'svelte'
  import CityPicker from '../../components/CityPicker.svelte'
  import Head from '../../components/Head.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { api, ApiError, type Application, type CenterKind, type Me } from '../../lib/api'
  import { isRegion, type City } from '../../lib/cities'
  import { t, type Key } from '../../lib/i18n/index.svelte'
  import { go, type Route } from '../../lib/nav.svelte'
  import { haptic } from '../../lib/telegram'

  // Заявка «Добавить свой центр / Я учитель» (DECISIONS 2026-11-25): кнопка на сайте → бот → эта анкета.
  // Кто подаёт, сервер знает по входу через Telegram; владелец одобряет в боте — кабинет создаётся сам.
  let { r, me }: { r: Route; me: Me } = $props()

  let kind = $state<CenterKind>(untrack(() => (r.s === 'apply' ? r.kind : 'center')))
  let name = $state('')
  let city = $state<'' | City>(untrack(() => (isRegion(me.city) ? me.city : '')))
  let students = $state<number | null>(null)
  let phone = $state('')
  let about = $state('')
  let last = $state<Application | null | undefined>(undefined)
  let sending = $state(false)
  let error = $state<Key | ''>('')

  const staff = $derived(me.center?.status === 'member' && me.center.role === 'admin')
  const ready = $derived(name.trim().length >= 2 && !!city && students !== null && students >= 0)

  onMount(async () => {
    try {
      last = (await api.myApplication()).application
    } catch {
      last = null
    }
  })

  async function send() {
    if (!ready || !city) return
    sending = true
    error = ''
    try {
      last = (await api.apply({ kind, name: name.trim(), city, students: students ?? 0, phone: phone.trim(), about: about.trim() })).application
      haptic.success()
    } catch (e) {
      haptic.error()
      error = e instanceof ApiError && e.code === 'bad_application' ? 'apply.bad' : 'app.error'
    } finally {
      sending = false
    }
  }
</script>

<Head title={t('apply.title')} demo={false} />

{#if staff}
  <section class="card stack">
    <p>{t('apply.already')}</p>
    <button class="btn" onclick={() => go({ s: 'cabinet' })}>{t('apply.open')}</button>
  </section>
{:else if last === undefined}
  <Skeleton rows={4} />
{:else if last?.status === 'pending'}
  <section class="card stack">
    <p>{t('apply.pending', { name: last.name })}</p>
  </section>
{:else}
  {#if last?.status === 'rejected'}<p class="hint">{t('apply.rejected')}</p>{/if}
  <p class="hint">{t('apply.lead')}</p>
  <div class="seg" role="radiogroup">
    {#each [['center', 'apply.center'], ['teacher', 'apply.teacher']] as [k, label] (k)}
      <button role="radio" aria-checked={kind === k} class:on={kind === k} onclick={() => (kind = k as CenterKind)}>{t(label as Key)}</button>
    {/each}
  </div>
  <section class="card stack">
    <label>
      <span class="hint">{kind === 'teacher' ? t('apply.nameTeacher') : t('apply.nameCenter')}</span>
      <input bind:value={name} maxlength="80" autocomplete="organization" />
    </label>
    <label>
      <span class="hint">{t('apply.region')}</span>
      <CityPicker value={city} regions onpick={(c) => (city = c)} />
    </label>
    <label>
      <span class="hint">{t('apply.students')}</span>
      <input type="number" inputmode="numeric" min="0" max="100000" bind:value={students} />
    </label>
    <label>
      <span class="hint">{t('apply.phone')}</span>
      <input type="tel" bind:value={phone} maxlength="30" autocomplete="tel" />
    </label>
    <label>
      <span class="hint">{t('apply.about')}</span>
      <textarea bind:value={about} maxlength="1000" rows="3"></textarea>
    </label>
  </section>
  {#if error}<p class="error" role="alert">{t(error)}</p>{/if}
  <button class="btn" disabled={!ready || sending} onclick={send}>{t('apply.send')}</button>
{/if}

<style>
  label {
    display: grid;
    gap: 4px;
    min-width: 0;
  }

  input,
  textarea {
    width: 100%;
    min-width: 0;
    min-height: 50px;
    padding: 0 14px;
    border: 1px solid var(--separator);
    border-radius: var(--radius);
    background: var(--section);
    color: var(--text);
    font: inherit;
  }

  textarea {
    padding: 12px 14px;
    resize: vertical;
  }
</style>
