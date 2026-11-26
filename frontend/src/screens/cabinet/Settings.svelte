<script lang="ts">
  import { untrack } from 'svelte'
  import { ApiError } from '../../lib/api'
  import { fmtDate } from '../../lib/date'
  import { t } from '../../lib/i18n/index.svelte'
  import { go } from '../../lib/nav.svelte'
  import { compressPhoto } from '../../lib/photo'
  import { haptic, inTelegram } from '../../lib/telegram'
  import { cab, centerPrice, MAX_SEATS, minSeats, MONTHS, sum, type SectionProps } from './cabinet'

  // Только аккаунт центра: текст «О центре», новая ссылка-приглашение, тариф, оплаты и продление тарифа
  // самим центром (перевод на карту со скрином, владелец подтверждает в боте). Название и город — у владельца.
  let { data, onchange }: SectionProps = $props()

  const c = $derived(data.center)
  // Поле — черновик: после сохранения берёт текст с сервера (run), а не следит за data.
  let about = $state(untrack(() => data.center.about))
  let saving = $state(false)
  let note = $state('')
  let asking = $state(false)
  let error = $state('')

  async function run(fn: () => Promise<typeof data>, done: string) {
    error = ''
    note = ''
    saving = true
    try {
      const next = await fn()
      onchange(next)
      about = next.center.about
      note = done
      haptic.success()
    } catch {
      error = t('cab.failed')
      haptic.error()
    } finally {
      saving = false
      asking = false
    }
  }

  // Продление: места — не меньше, чем учеников сейчас; сумма — как на сервере (centerPrice).
  const min = $derived(minSeats(data.center.kind))
  let seats = $state(untrack(() => Math.max(minSeats(data.center.kind), data.center.seats, data.center.students)))
  let months = $state(1)
  let shot = $state<Blob | null>(null)
  let copied = $state('')
  const price = $derived(centerPrice(seats, months, data.center.kind))
  const pending = $derived(data.pay?.pending ?? null)

  async function pickShot(e: Event) {
    const f = (e.target as HTMLInputElement).files?.[0]
    shot = f ? await compressPhoto(f) : null
  }

  async function copy(text: string) {
    try {
      await navigator.clipboard.writeText(text)
      copied = text
      haptic.success()
    } catch {
      // буфер недоступен — номер виден на экране, его можно выделить
    }
  }

  async function sendOrder() {
    if (!shot || price === null) return
    error = ''
    note = ''
    saving = true
    try {
      onchange(await cab.order(seats, months, shot))
      shot = null
      haptic.success()
    } catch (e) {
      error = e instanceof ApiError && e.code === 'bad_image' ? t('cab.badShot') : t('cab.failed')
      haptic.error()
    } finally {
      saving = false
    }
  }

  // В Mini App «поддержка» — свой экран; на сайте — то же приложение по адресу /.
  function support() {
    if (inTelegram()) go({ s: 'support' })
    else location.href = '/?open=support'
  }
</script>

<section class="card">
  <h2>{c.name}</h2>
  <label for="about" class="hint">{t('cab.aboutHint')}</label>
  <textarea id="about" bind:value={about} maxlength="2000" rows="5" placeholder={t('cab.about')}></textarea>
  <div class="line">
    <button class="btn btn-small" disabled={saving || about.trim() === c.about} onclick={() => run(() => cab.about(about), t('cab.saved'))}>{t('cab.save')}</button>
    <span class="hint">{about.length} / 2000</span>
  </div>
</section>

<section class="card">
  <h2>{t('cab.invite')}</h2>
  <p class="hint">{t('cab.newCodeAsk')}</p>
  {#if asking}
    <div class="line">
      <button class="btn btn-small danger" disabled={saving} onclick={() => run(cab.newCode, t('cab.saved'))}>{t('cab.newCode')}</button>
      <button class="btn btn-small btn-plain" onclick={() => (asking = false)}>×</button>
    </div>
  {:else}
    <div class="line"><button class="btn btn-small btn-plain" onclick={() => (asking = true)}>{t('cab.newCode')}</button></div>
  {/if}
</section>

{#if note}<p class="ok" role="status">{note}</p>{/if}
{#if error}<p class="error" role="alert">{error}</p>{/if}

<section class="card">
  <h2>{t('cab.plan')}</h2>
  <dl>
    <div><dt class="hint">{t('cab.seats')}</dt><dd class="num">{c.seats}</dd></div>
    <div><dt class="hint">{t('cab.students')}</dt><dd class="num">{c.students}</dd></div>
    <div><dt class="hint">{t('cab.paidUntil')}</dt><dd>{c.paid_until ? fmtDate(c.paid_until) : t('cab.notPaid')}</dd></div>
  </dl>
  <h3>{t('cab.payments')}</h3>
  {#if !data.payments?.length}
    <p class="hint">{t('cab.noPayments')}</p>
  {:else}
    <ul class="rows pays">
      {#each data.payments as p (p.at)}
        <li class="row">
          <span class="grow">
            <strong>{t('cab.sum', { n: sum(p.amount) })}</strong>
            <span class="tag">{t('cab.paySeats', { n: p.seats, date: fmtDate(p.paid_until) })}</span>
          </span>
          <span class="hint">{fmtDate(p.at)}</span>
        </li>
      {/each}
    </ul>
  {/if}
</section>

{#if data.pay}
  <section class="card">
    <h2>{t('cab.renew')}</h2>
    {#if pending}
      <p>{t('cab.orderPending', { seats: pending.seats, months: t('cab.months', { n: pending.months }), sum: sum(pending.amount) })}</p>
      <p class="hint">{t('cab.orderPendingHint')}</p>
    {:else if !data.pay.card}
      <p class="hint">{t('cab.noCard')}</p>
      <button class="more link" onclick={support}>{t('cab.planHint')}</button>
    {:else}
      <div class="pick">
        <label>
          <span class="hint">{t('cab.renewSeats')}</span>
          <input type="number" inputmode="numeric" {min} max={MAX_SEATS} bind:value={seats} />
        </label>
        <label>
          <span class="hint">{t('cab.renewMonths')}</span>
          <select bind:value={months}>
            {#each MONTHS as m (m)}<option value={m}>{t('cab.months', { n: m })}</option>{/each}
          </select>
        </label>
      </div>
      <p class="hint">{t('cab.priceRule', { n: min })}</p>
      <p class="total num">{price === null ? t('cab.minSeats', { n: min }) : t('cab.sum', { n: sum(price) })}</p>
      <ol class="steps">
        <li>
          {t('cab.step1')}
          <span class="copy">
            <code>{data.pay.card}</code>
            <button class="link" onclick={() => copy(data.pay!.card)}>{copied === data.pay.card ? t('cab.copied') : t('cab.copy')}</button>
          </span>
          {#if data.pay.holder}<span class="hint">{data.pay.holder}</span>{/if}
        </li>
        <li>
          {t('cab.step2')}
          <span class="copy">
            <code>{data.pay.code}</code>
            <button class="link" onclick={() => copy(data.pay!.code)}>{copied === data.pay.code ? t('cab.copied') : t('cab.copy')}</button>
          </span>
        </li>
        <li>{t('cab.step3')}</li>
      </ol>
      <div class="line">
        <label class="btn btn-small btn-plain file">
          {shot ? t('cab.shotReady') : t('cab.shot')}
          <input type="file" accept="image/*" onchange={pickShot} />
        </label>
        <button class="btn btn-small" disabled={saving || !shot || price === null} onclick={sendOrder}>{t('cab.paid')}</button>
      </div>
      <p class="hint">{t('cab.orderHint')}</p>
    {/if}
  </section>
{/if}

<style>
  section {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: 10px;
    min-width: 0;
  }

  h2 {
    font-size: 17px;
    overflow-wrap: anywhere;
  }

  h3 {
    margin: 6px 0 0;
    font-size: 15px;
  }

  .line {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 8px 12px;
  }

  .line .btn-plain {
    background: var(--surface);
  }

  .danger {
    background: var(--danger);
  }

  .ok {
    color: var(--ok);
    font-size: 14px;
  }

  dl {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(130px, 1fr));
    gap: 10px;
    margin: 0;
  }

  dd {
    margin: 2px 0 0;
    font-size: 18px;
  }

  .pays {
    padding: 0;
  }

  .pays .row {
    flex-wrap: wrap;
    min-height: 48px;
  }

  .link {
    text-align: left;
    color: var(--link);
  }

  .pick {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(min(160px, 100%), 1fr));
    gap: 10px;
  }

  .pick label {
    display: grid;
    gap: 4px;
    min-width: 0;
  }

  .pick input,
  .pick select {
    width: 100%;
    min-width: 0;
    min-height: 44px;
    padding: 0 12px;
    border: 1px solid var(--separator);
    border-radius: 10px;
    background: var(--section);
    color: var(--text);
    font: inherit;
  }

  .total {
    font-size: 24px;
    font-weight: 700;
  }

  .steps {
    display: grid;
    gap: 10px;
    margin: 0;
    padding-left: 20px;
  }

  .steps li {
    display: grid;
    gap: 4px;
    min-width: 0;
  }

  .copy {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 4px 12px;
    min-width: 0;
  }

  code {
    font-size: 16px;
    overflow-wrap: anywhere;
  }

  .file {
    position: relative;
    overflow: hidden;
  }

  .file input {
    position: absolute;
    inset: 0;
    opacity: 0;
  }
</style>
