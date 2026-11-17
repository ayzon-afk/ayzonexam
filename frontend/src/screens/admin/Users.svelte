<script lang="ts">
  import { onMount } from 'svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { adm, when, type Row } from './admin'

  // Ученики: поиск, профиль, попытки, Premium, блокировка, роль в центре, «Проверенный учитель».
  let q = $state('')
  let list = $state<Row[] | null>(null)
  let open = $state<Row | null>(null)
  let centers = $state<Row[]>([])
  let note = $state('')
  let days = $state(30)

  const lists: [string, string][] = [
    ['attempts', 'Попытки'],
    ['mocks', 'Mock'],
    ['writing', 'Writing'],
    ['speaking', 'Speaking'],
    ['payments', 'Оплаты'],
    ['tickets', 'Обращения'],
  ]

  async function search() {
    list = null
    list = (await adm.get<{ users: Row[] }>(`users?q=${encodeURIComponent(q)}`)).users
  }

  async function show(id: number) {
    note = ''
    open = await adm.get(`users/${id}`)
  }

  async function patch(body: Row, done: string) {
    if (!open) return
    try {
      open = await adm.patch(`users/${open.user.id}`, body)
      note = done
    } catch {
      note = 'Не сохранилось.'
    }
  }

  onMount(async () => {
    search()
    centers = (await adm.get<{ centers: Row[] }>('centers')).centers
  })
</script>

{#if open}
  {@const u = open.user}
  <button class="btn btn-plain" onclick={() => (open = null)}>‹ К списку</button>
  <section class="card stack">
    <h2>{u.first_name} {u.last_name} <span class="hint">@{u.username || '—'}</span></h2>
    <p class="hint">Ник: {u.nickname} · Telegram id: {u.telegram_id} · язык: {u.lang} · город: {u.city || '—'}</p>
    <p class="hint">С нами с {when(u.created_at)} · был {when(u.seen_at)} · цель {u.goal_band ?? '—'} · сейчас {u.current ?? '—'}</p>
    <p>
      Premium: {open.plan.Premium ? `до ${when(open.plan.Until)} (${open.plan.Source})` : 'нет'}
      {#if u.pay_trust === 'rejected'}<span class="error"> · были отказы в оплате</span>{/if}
    </p>
    <p class="hint">Купленных mock: {u.mock_credits} · бонусных проверок: {u.bonus_checks}</p>
    {#if u.blocked_at}<p class="error">Заблокирован {when(u.blocked_at)}</p>{/if}
  </section>

  <section class="card stack">
    <h2>Premium</h2>
    <div class="line">
      <input type="number" min="1" max="365" bind:value={days} />
      <button class="btn btn-small" onclick={() => patch({ premium_days: days }, `Premium продлён на ${days} дн.`)}>Выдать / продлить</button>
      <button class="btn btn-small btn-plain" onclick={() => patch({ premium_days: -1 }, 'Premium забран.')}>Забрать</button>
    </div>
    <button class="btn btn-small btn-plain" onclick={() => patch({ mock_credits: u.mock_credits + 1 }, '+1 mock.')}>+1 полный mock</button>
  </section>

  <section class="card stack">
    <h2>Центр и роль</h2>
    <select onchange={(e) => patch({ center_id: Number((e.target as HTMLSelectElement).value) }, 'Центр изменён.')}>
      <option value="0" selected={!u.center_id}>— без центра —</option>
      {#each centers as c (c.id)}<option value={c.id} selected={u.center_id === c.id}>{c.name}</option>{/each}
    </select>
    <div class="line">
      <button class="btn btn-small" class:btn-plain={u.center_role !== 'admin'} onclick={() => patch({ role: 'admin' }, 'Аккаунт центра.')}>Аккаунт центра</button>
      <button class="btn btn-small" class:btn-plain={u.center_role !== 'teacher'} onclick={() => patch({ role: 'teacher' }, 'Учитель центра.')}>Учитель</button>
      <button class="btn btn-small btn-plain" onclick={() => patch({ role: '' }, 'Роль снята.')}>Без роли</button>
    </div>
    <label class="check"><input type="checkbox" checked={u.verified_teacher} onchange={(e) => patch({ verified: (e.target as HTMLInputElement).checked }, 'Сохранено.')} /> Проверенный учитель</label>
  </section>

  <button class="btn btn-plain danger" onclick={() => patch({ blocked: !u.blocked_at }, u.blocked_at ? 'Разблокирован.' : 'Заблокирован.')}>
    {u.blocked_at ? 'Разблокировать' : 'Заблокировать'}
  </button>
  {#if note}<p class="hint">{note}</p>{/if}

  {#each lists as [key, name] (key)}
    {#if open[key]?.length}
      <section class="card stack">
        <h2>{name}</h2>
        {#each open[key] as x (x.id)}
          <p class="hint">
            #{x.id} · {x.kind ?? x.task ?? x.plan ?? x.source ?? x.status} {x.title ?? ''} · {x.band ?? x.overall ?? x.amount ?? ''} · {x.status ?? (x.ranked ? 'в рейтинге' : '')} · {when(x.finished_at ?? x.created_at ?? x.started_at ?? x.updated_at)}
          </p>
        {/each}
      </section>
    {/if}
  {/each}
{:else}
  <form class="line" onsubmit={(e) => (e.preventDefault(), search())}>
    <input bind:value={q} placeholder="Ник, имя, @username или Telegram id" />
    <button class="btn btn-small">Найти</button>
  </form>
  {#if !list}
    <Skeleton rows={4} />
  {:else}
    <ul class="card rows">
      {#each list as u (u.id)}
        <li>
          <button class="row" onclick={() => show(u.id)}>
            <span class="grow">
              <strong>{u.first_name} {u.last_name}</strong>
              <span class="tag">{u.nickname} · @{u.username || '—'} · {u.center ?? 'без центра'}{u.center_role ? ` (${u.center_role})` : ''}</span>
            </span>
            {#if u.blocked_at}<span class="error">блок</span>{:else if u.premium_until && new Date(u.premium_until) > new Date()}<span class="tag">Premium</span>{/if}
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

  .line {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    align-items: center;
  }

  input:not([type='checkbox']),
  select {
    flex: 1;
    min-width: 0;
    min-height: 40px;
    padding: 0 10px;
    border: 1px solid var(--separator);
    border-radius: 10px;
    background: var(--section);
    color: var(--text);
    font: inherit;
  }

  input[type='number'] {
    max-width: 90px;
  }

  .check {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .danger {
    color: var(--danger);
  }
</style>
