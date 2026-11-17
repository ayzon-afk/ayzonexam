<script lang="ts">
  import { onMount } from 'svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { adm, privateImage, sum, when, type Row } from './admin'

  // Оплаты учеников переводом: скрин, сумма, код — «Подтвердить» / «Отказать» (то же — кнопками в Telegram).
  let status = $state('pending')
  let list = $state<Row[] | null>(null)
  let shots = $state<Record<number, string>>({})
  let note = $state('')

  const plans: Record<string, string> = { month: '1 месяц', quarter: '3 месяца', mock: 'полный mock' }
  const statuses: Record<string, string> = { pending: 'на проверке', confirmed: 'подтверждена', rejected: 'отказано' }
  const reasons: [string, string][] = [
    ['notfound', 'Перевод не найден'],
    ['amount', 'Не та сумма'],
    ['screenshot', 'Скрин не подходит'],
  ]

  async function load() {
    list = null
    list = (await adm.get<{ payments: Row[] }>(`payments?status=${status}`)).payments
  }

  async function shot(p: Row) {
    try {
      shots[p.id] = await privateImage(p.screenshot)
    } catch {
      note = 'Скрин не открылся.'
    }
  }

  async function decide(p: Row, confirm: boolean, reason = '') {
    try {
      await adm.post(`payments/${p.id}`, { confirm, reason })
      note = confirm ? `Оплата #${p.id} подтверждена.` : `Оплата #${p.id}: отказ.`
      await load()
    } catch (e) {
      note = `Не получилось: ${(e as Error).message}`
    }
  }

  onMount(load)
</script>

<div class="seg">
  {#each ['pending', 'confirmed', 'rejected', ''] as s (s)}
    <button class:on={status === s} onclick={() => ((status = s), load())}>{s ? statuses[s] : 'все'}</button>
  {/each}
</div>
{#if note}<p class="hint">{note}</p>{/if}

{#if !list}
  <Skeleton rows={3} />
{:else if list.length === 0}
  <p class="card hint">Пусто.</p>
{:else}
  {#each list as p (p.id)}
    <section class="card stack">
      <p><b>#{p.id} · {sum(p.amount)} сум · {plans[p.plan]}</b> · {statuses[p.status]}</p>
      <p class="hint">{p.nickname} · Telegram id {p.telegram_id} · код {p.code} · {when(p.created_at)}</p>
      {#if p.status === 'pending'}
        <p class="hint">{p.granted_days || p.granted_mocks ? 'Доступ открыт сразу — «на проверке».' : 'Доступ не открыт: ученику уже отказывали.'}</p>
      {/if}
      {#if p.reason}<p class="hint">Причина: {p.reason}</p>{/if}
      {#if shots[p.id]}<img src={shots[p.id]} alt="скрин оплаты" />{:else}<button class="btn btn-small btn-plain" onclick={() => shot(p)}>Показать скрин</button>{/if}
      {#if p.status === 'pending'}
        <button class="btn" onclick={() => decide(p, true)}>✅ Подтвердить</button>
        <div class="line">
          {#each reasons as [code, text] (code)}
            <button class="btn btn-small btn-plain" onclick={() => decide(p, false, code)}>❌ {text}</button>
          {/each}
        </div>
      {/if}
    </section>
  {/each}
{/if}

<style>
  img {
    max-width: 100%;
    max-height: 420px;
    border-radius: 12px;
    justify-self: start;
  }

  .line {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }
</style>
