<script lang="ts">
  import { onMount } from 'svelte'
  import Head from '../components/Head.svelte'
  import Skeleton from '../components/Skeleton.svelte'
  import { api, type Ticket } from '../lib/api'
  import { fmtDate } from '../lib/date'
  import { t, type Key } from '../lib/i18n/index.svelte'
  import { compressPhoto } from '../lib/photo'
  import { haptic } from '../lib/telegram'

  // Поддержка (SPEC): сообщение (можно с фото) уходит владельцу в админку и в Telegram; ответ приходит
  // сюда и ботом. Тот же путь — /help в боте.
  let tickets = $state<Ticket[] | null>(null)
  let text = $state('')
  let photo = $state<Blob | null>(null)
  let sending = $state(false)
  let note = $state<Key | ''>('')

  async function load() {
    try {
      tickets = (await api.support()).tickets
    } catch {
      tickets = []
    }
  }

  async function pick(e: Event) {
    const f = (e.target as HTMLInputElement).files?.[0]
    if (!f) return
    try {
      photo = await compressPhoto(f)
    } catch {
      note = 'err.image'
    }
  }

  async function send() {
    sending = true
    note = ''
    try {
      await api.writeSupport(text, photo ?? undefined)
      text = ''
      photo = null
      note = 'support.sent'
      haptic.success()
      await load()
    } catch {
      haptic.error()
      note = 'app.error'
    } finally {
      sending = false
    }
  }

  const status: Record<string, Key> = { open: 'support.open', answered: 'support.answered', closed: 'support.closed' }

  onMount(load)
</script>

<Head title={t('support.title')} sub={t('support.lead')} demo={false} />

<section class="card stack">
  <textarea bind:value={text} placeholder={t('support.placeholder')} maxlength="4000"></textarea>
  <label class="btn btn-plain file">
    {photo ? t('support.photoAdded') : t('support.addPhoto')}
    <input type="file" accept="image/*" onchange={pick} />
  </label>
  <button class="btn" disabled={sending || (!text.trim() && !photo)} onclick={send}>{t('support.send')}</button>
  {#if note}<p class={note === 'support.sent' ? 'hint' : 'error'}>{t(note)}</p>{/if}
</section>

{#if !tickets}
  <Skeleton rows={2} />
{:else}
  {#each tickets as tk (tk.id)}
    <section class="card stack">
      <p class="tag">#{tk.id} · {t(status[tk.status]!)} · {fmtDate(tk.updated_at)}</p>
      {#each tk.messages ?? [] as m (m.id)}
        <p class="msg" class:admin={m.from_admin}>{m.text || '📷'}</p>
      {/each}
    </section>
  {/each}
{/if}
<p class="hint">{t('support.bot')}</p>

<style>
  .file {
    position: relative;
    overflow: hidden;
    text-align: center;
  }

  .file input {
    position: absolute;
    inset: 0;
    opacity: 0;
  }

  .msg {
    padding: 8px 12px;
    border-radius: 12px;
    background: var(--surface);
    white-space: pre-wrap;
    justify-self: end;
    max-width: 90%;
  }

  .msg.admin {
    justify-self: start;
    background: color-mix(in srgb, var(--accent) 14%, transparent);
  }
</style>
