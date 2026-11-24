<script lang="ts">
  import { onMount } from 'svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { adm, sum, type Row } from './admin'

  let { goto }: { goto: (id: string) => void } = $props()

  let o = $state<Row | null>(null)
  let failed = $state(false)

  onMount(async () => {
    try {
      o = await adm.get('overview')
    } catch {
      failed = true
    }
  })
</script>

{#if failed}
  <p class="error">Не загрузилось.</p>
{:else if !o}
  <Skeleton rows={4} />
{:else}
  <div class="grid">
    <div class="card"><p class="num">{o.users}</p><p class="hint">учеников всего</p></div>
    <div class="card"><p class="num">{o.active_day} / {o.active_week}</p><p class="hint">активных день / неделя</p></div>
    <div class="card"><p class="num">{o.new_day} / {o.new_week}</p><p class="hint">новых день / неделя</p></div>
    <div class="card"><p class="num">{o.paying}</p><p class="hint">платящих сейчас</p></div>
    <div class="card"><p class="num">{sum(o.money_month)}</p><p class="hint">сум за месяц (ученики и центры)</p></div>
    <div class="card"><p class="num">{sum(o.money_total)}</p><p class="hint">сум за всё время</p></div>
    <div class="card"><p class="num">{o.centers}</p><p class="hint">центров</p></div>
    <div class="card"><p class="num">{o.writing_day} / {o.writing_month}</p><p class="hint">проверок Writing день / месяц</p></div>
    <div class="card"><p class="num">{o.speaking_day} / {o.speaking_month}</p><p class="hint">проверок Speaking день / месяц</p></div>
    <div class="card"><p class="num">${o.ai_cost_month.toFixed(2)}</p><p class="hint">AI за месяц (примерно)</p></div>
  </div>
  <button class="card row" onclick={() => goto('payments')}>
    <span class="grow">Оплаты на проверке</span><b>{o.pending_payments}</b><span class="chev">›</span>
  </button>
  <button class="card row" onclick={() => goto('tickets')}>
    <span class="grow">Открытые обращения</span><b>{o.open_tickets}</b><span class="chev">›</span>
  </button>
{/if}

<style>
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
    gap: 10px;
  }

  .num {
    font-size: 22px;
  }

  .row {
    padding: 0 16px;
  }
</style>
