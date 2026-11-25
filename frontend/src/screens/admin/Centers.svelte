<script lang="ts">
  import { onMount } from 'svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { request } from '../../lib/api'
  import { compressPhoto } from '../../lib/photo'
  import { inviteLink } from '../../lib/share'
  import { adm, sum, when, type Row } from './admin'

  // Центры: создать и изменить, ссылка-приглашение, учителя, оплата тарифа (B2B — чек и подтверждение).
  let list = $state<Row[] | null>(null)
  let edit = $state<Row | null>(null)
  let pays = $state<Row[]>([])
  let note = $state('')
  let bot = $state('')

  // Форма оплаты центра.
  let amount = $state(0)
  let seats = $state(20)
  let until = $state('')
  let payNote = $state('')
  let receipt: Blob | null = null

  async function load() {
    list = (await adm.get<{ centers: Row[] }>('centers')).centers
  }

  async function openCenter(c: Row | null) {
    note = ''
    edit = c ? { ...c } : { name: '', city: 'tashkent', about: '', active: true }
    pays = c ? (await adm.get<{ payments: Row[] }>(`centers/${c.id}/payments`)).payments : []
    seats = c?.seats || 20
    amount = seats * 20000
  }

  async function save(newCode = false) {
    if (!edit) return
    try {
      const body = { name: edit.name, city: edit.city, about: edit.about, active: edit.active, new_code: newCode }
      if (edit.id) await adm.put(`centers/${edit.id}`, body)
      else await adm.post('centers', body)
      note = 'Сохранено.'
      await load()
      if (edit.id) edit = list!.find((c) => c.id === edit!.id) ?? null
      else edit = null
    } catch {
      note = 'Не сохранилось: имя пустое или занято.'
    }
  }

  async function pay() {
    if (!edit?.id) return
    const f = new FormData()
    f.append('amount', String(amount))
    f.append('seats', String(seats))
    f.append('paid_until', until)
    f.append('note', payNote)
    if (receipt) f.append('receipt', receipt, 'receipt.jpg')
    try {
      pays = (await request<{ payments: Row[] }>('POST', `/api/admin/centers/${edit.id}/payments`, f)).payments
      note = 'Оплата записана, тариф продлён.'
      await load()
    } catch {
      note = 'Не записалось: проверьте сумму, число учеников и дату.'
    }
  }

  onMount(async () => {
    load()
    try {
      bot = (await request<{ bot: string }>('GET', '/api/me')).bot
    } catch {
      // ссылка будет без бота
    }
  })

  // Цена по SPEC: 20 000 за ученика (от 100 — 15 000), минимум 20 учеников.
  const suggest = $derived(Math.max(seats, 20) * (seats >= 100 ? 15000 : 20000))
</script>

{#if edit}
  <button class="btn btn-plain" onclick={() => (edit = null)}>‹ К списку</button>
  <section class="card stack">
    <h2>{edit.id ? edit.name : 'Новый центр'}</h2>
    <input bind:value={edit.name} placeholder="Название" />
    <select bind:value={edit.city}>
      <option value="tashkent">Ташкент</option>
      <option value="other">Другой город</option>
    </select>
    <textarea bind:value={edit.about} placeholder="О центре: адрес, телефон, курсы"></textarea>
    {#if edit.id}<label class="check"><input type="checkbox" bind:checked={edit.active} /> Активен</label>{/if}
    <button class="btn" onclick={() => save()}>Сохранить</button>
  </section>

  {#if edit.id}
    <section class="card stack">
      <h2>Ссылка-приглашение</h2>
      <p class="mono">{inviteLink(bot, `c-${edit.invite_code}`)}</p>
      <p class="hint">По ней ученик входит в центр сразу. Код: {edit.invite_code}</p>
      <button class="btn btn-small btn-plain" onclick={() => save(true)}>Сменить код (старая ссылка перестанет работать)</button>
      <p class="hint">Учеников: {edit.students} · ждут подтверждения: {edit.pending}</p>
      {#each edit.staff as s (s.id)}<p class="hint">{s.role === 'admin' ? 'Аккаунт центра' : 'Учитель'}: {s.name} ({s.nickname})</p>{/each}
    </section>

    <section class="card stack">
      <h2>Кабинет центра</h2>
      <p class="mono">{location.origin}/cabinet</p>
      <p class="hint">
        Отправьте эту ссылку центру. Войти может только тот, кому вы дали роль «Аккаунт центра» или «Учитель» (раздел
        «Ученики» → ученик этого центра). Аккаунт центра видит учеников с баллами, заявки, ссылку, тариф и оплаты; учитель —
        без денег и без управления. Действия центра пишутся в «Журнал».
      </p>
      {#if !edit.staff.some((s: Row) => s.role === 'admin')}<p class="error">У центра ещё нет аккаунта центра — кабинет никому не откроется.</p>{/if}
    </section>

    <section class="card stack">
      <h2>Тариф: {edit.seats} учеников, оплачено до {edit.paid_until ? new Date(edit.paid_until).toLocaleDateString('ru-RU') : '—'}</h2>
      <label>Учеников <input type="number" min="1" bind:value={seats} /></label>
      <label>Сумма, сум <input type="number" min="1" bind:value={amount} /></label>
      <p class="hint">По прайсу: {sum(suggest)} сум в месяц</p>
      <label>Оплачено до <input type="date" bind:value={until} /></label>
      <input bind:value={payNote} placeholder="Заметка: договор, способ оплаты" />
      <label class="btn btn-plain file">Приложить чек<input type="file" accept="image/*" onchange={async (e) => (receipt = await compressPhoto((e.target as HTMLInputElement).files![0]!))} /></label>
      <button class="btn" disabled={!until || amount <= 0} onclick={pay}>Подтвердить оплату</button>
      {#each pays as p (p.id)}
        <p class="hint">{when(p.created_at)} · {sum(p.amount)} сум · {p.seats} уч. · до {new Date(p.paid_until).toLocaleDateString('ru-RU')} {p.note}</p>
      {/each}
    </section>
  {/if}
  {#if note}<p class="hint">{note}</p>{/if}
{:else}
  <button class="btn" onclick={() => openCenter(null)}>+ Новый центр</button>
  {#if !list}
    <Skeleton rows={3} />
  {:else}
    <ul class="card rows">
      {#each list as c (c.id)}
        <li>
          <button class="row" onclick={() => openCenter(c)}>
            <span class="grow">
              <strong>{c.name}{c.active ? '' : ' (выключен)'}</strong>
              <span class="tag">учеников {c.students} · ждут {c.pending} · оплачено до {c.paid_until ? new Date(c.paid_until).toLocaleDateString('ru-RU') : '—'}</span>
            </span>
            <span class="chev">›</span>
          </button>
        </li>
      {/each}
    </ul>
  {/if}
{/if}

<style>
  h2 {
    font-size: 16px;
  }

  strong {
    display: block;
  }

  input:not([type='checkbox']):not([type='file']),
  select {
    width: 100%;
    min-height: 40px;
    padding: 0 10px;
    border: 1px solid var(--separator);
    border-radius: 10px;
    background: var(--section);
    color: var(--text);
    font: inherit;
  }

  label {
    display: grid;
    gap: 4px;
  }

  .check {
    display: flex;
    gap: 8px;
    align-items: center;
  }

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

  .mono {
    font-family: ui-monospace, monospace;
    word-break: break-all;
  }
</style>
