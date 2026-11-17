<script lang="ts">
  import { onMount } from 'svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { adm, privateImage, when, type Row } from './admin'

  // Поддержка: обращения из приложения и /help; ответ уходит ученику ботом (так же — reply в Telegram).
  let status = $state('open')
  let list = $state<Row[] | null>(null)
  let open = $state<Row | null>(null)
  let reply = $state('')
  let photos = $state<Record<number, string>>({})
  let note = $state('')

  const statuses: Record<string, string> = { open: 'открыто', answered: 'отвечено', closed: 'закрыто' }

  async function load() {
    list = null
    list = (await adm.get<{ tickets: Row[] }>(`tickets?status=${status}`)).tickets
  }

  async function show(id: number) {
    note = ''
    open = await adm.get(`tickets/${id}`)
    for (const m of open!.messages) if (m.photo) privateImage(m.photo).then((u) => (photos[m.id] = u), () => {})
  }

  async function act(body: Row) {
    if (!open) return
    try {
      open = await adm.post(`tickets/${open.id}`, body)
      reply = ''
      note = 'Готово.'
    } catch {
      note = 'Не получилось.'
    }
  }

  onMount(load)
</script>

{#if open}
  <button class="btn btn-plain" onclick={() => ((open = null), load())}>‹ К списку</button>
  <section class="card stack">
    <p><b>#{open.id} · {open.nickname}</b> · {statuses[open.status]}</p>
    <p class="hint">Telegram id {open.telegram_id} · {open.premium ? 'Premium' : 'бесплатно'} · центр: {open.center || '—'} · язык: {open.lang}</p>
    {#each open.messages as m (m.id)}
      <div class="msg" class:admin={m.from_admin}>
        {#if m.text}<p>{m.text}</p>{/if}
        {#if photos[m.id]}<img src={photos[m.id]} alt="" />{/if}
        <p class="hint">{when(m.created_at)}</p>
      </div>
    {/each}
    <textarea bind:value={reply} placeholder="Ответ ученику (уйдёт ботом на его языке приложения)"></textarea>
    <button class="btn" disabled={!reply.trim()} onclick={() => act({ reply })}>Ответить</button>
    <div class="line">
      {#each ['open', 'answered', 'closed'] as s (s)}
        <button class="btn btn-small btn-plain" disabled={open.status === s} onclick={() => act({ status: s })}>{statuses[s]}</button>
      {/each}
    </div>
    {#if note}<p class="hint">{note}</p>{/if}
  </section>
{:else}
  <div class="seg">
    {#each ['open', 'answered', 'closed', ''] as s (s)}
      <button class:on={status === s} onclick={() => ((status = s), load())}>{s ? statuses[s] : 'все'}</button>
    {/each}
  </div>
  {#if !list}
    <Skeleton rows={3} />
  {:else if list.length === 0}
    <p class="card hint">Обращений нет.</p>
  {:else}
    <ul class="card rows">
      {#each list as tk (tk.id)}
        <li>
          <button class="row" onclick={() => show(tk.id)}>
            <span class="grow">
              <strong>#{tk.id} · {tk.nickname}</strong>
              <span class="tag">{tk.last || '📷'}</span>
            </span>
            <span class="tag">{when(tk.updated_at)}</span>
          </button>
        </li>
      {/each}
    </ul>
  {/if}
{/if}

<style>
  strong {
    display: block;
  }

  .tag {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .msg {
    padding: 8px 12px;
    border-radius: 12px;
    background: var(--surface);
    white-space: pre-wrap;
  }

  .msg.admin {
    background: color-mix(in srgb, var(--accent) 14%, transparent);
  }

  img {
    max-width: 100%;
    border-radius: 10px;
  }

  .line {
    display: flex;
    gap: 6px;
  }
</style>
