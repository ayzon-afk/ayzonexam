<script lang="ts">
  import Head from '../components/Head.svelte'
  import { api, type Me } from '../lib/api'
  import { fmtDate } from '../lib/date'
  import { t, type Key } from '../lib/i18n/index.svelte'
  import { go } from '../lib/nav.svelte'
  import { openExternal } from '../lib/telegram'

  // Premium: что даёт, цены (SPEC, «Деньги»), оплата — на странице сайта переводом на карту со скрином.
  let { me }: { me: Me } = $props()

  const plan = $derived(me.plan)
  const discount = $derived(plan?.discount_until ? new Date(plan.discount_until) : null)
  let opening = $state(false)
  let error = $state(false)

  const sum = (n: number) => n.toLocaleString('ru-RU').replace(/ /g, ' ')
  const source: Record<string, Key> = { trial: 'premium.src.trial', paid: 'premium.src.paid', center: 'premium.src.center' }

  async function pay() {
    opening = true
    error = false
    try {
      const { path } = await api.payLink()
      openExternal(location.origin + path)
    } catch {
      error = true
    } finally {
      opening = false
    }
  }
</script>

<Head title="Premium" sub={t('premium.lead')} demo={false} />

{#if plan}
  <section class="card stack status">
    {#if plan.premium && plan.until}
      <p><b>{t('premium.active', { date: fmtDate(plan.until) })}</b></p>
      <p class="hint">{t(source[plan.source] ?? 'premium.src.paid')}</p>
    {:else}
      <p><b>{t('premium.inactive')}</b></p>
    {/if}
    {#if plan.pending}<p class="warn">{t('premium.pending')}</p>{/if}
    {#if plan.mock_credits > 0}<p>{t('mock.credits', { n: plan.mock_credits })}</p>{/if}
  </section>
{/if}

<div class="plans">
  <section class="card stack">
    <h2>{t('premium.free')}</h2>
    <ul>
      <li>{t('premium.f1')}</li>
      <li>{t('premium.f2')}</li>
      <li>{t('premium.f3')}</li>
    </ul>
  </section>
  <section class="card stack pro">
    <h2>Premium</h2>
    <ul>
      <li>{t('premium.p1')}</li>
      <li>{t('premium.p2')}</li>
      <li>{t('premium.p3')}</li>
    </ul>
  </section>
</div>

{#if plan}
  <ul class="card rows">
    <li class="row">
      <span class="grow">{t('premium.month')}{#if discount}<span class="badge">−30%</span>{/if}</span>
      {#if discount}<s class="hint">{sum(plan.prices.month)}</s>{/if}
      <b>{sum(discount ? plan.prices.month_discount : plan.prices.month)} {t('premium.sum')}</b>
    </li>
    <li class="row"><span class="grow">{t('premium.quarter')}</span><b>{sum(plan.prices.quarter)} {t('premium.sum')}</b></li>
    <li class="row"><span class="grow">{t('premium.mock')}</span><b>{sum(plan.prices.mock)} {t('premium.sum')}</b></li>
  </ul>
  {#if discount}<p class="hint">{t('premium.discount', { date: fmtDate(discount, true) })}</p>{/if}
  <button class="btn" disabled={opening || plan.pending} onclick={pay}>{t('premium.go')}</button>
  <p class="hint">{plan.rejected ? t('premium.howLater') : t('premium.how')}</p>
  {#if error}<p class="error">{t('app.error')}</p>{/if}
{/if}
<p class="hint">{t('premium.center')}</p>

<div class="sect"><h2>{t('profile.friends')}</h2></div>
<button class="card row" onclick={() => go({ s: 'friends' })}>
  <span class="grow">{t('premium.bonus')}</span>
  <span class="chev" aria-hidden="true">›</span>
</button>

<style>
  h2 {
    font-size: 18px;
  }

  .plans {
    display: grid;
    gap: 12px;
  }

  @media (min-width: 600px) {
    .plans {
      grid-template-columns: 1fr 1fr;
    }
  }

  ul:not(.rows) {
    margin: 0;
    padding-left: 20px;
    display: grid;
    gap: 6px;
  }

  .pro {
    outline: 2px solid var(--accent);
  }

  .badge {
    margin-left: 8px;
    padding: 1px 8px;
    border-radius: 999px;
    background: #2e9d58;
    color: #fff;
    font-size: 12px;
    font-weight: 700;
  }

  .warn {
    color: #d69a00;
    font-weight: 600;
  }

  b {
    font-variant-numeric: tabular-nums;
  }
</style>
