<script lang="ts">
  import { untrack } from 'svelte'
  import { fmtDate } from '../../lib/date'
  import { t } from '../../lib/i18n/index.svelte'
  import { go } from '../../lib/nav.svelte'
  import { haptic, inTelegram } from '../../lib/telegram'
  import { cab, sum, type SectionProps } from './cabinet'

  // Только аккаунт центра: текст «О центре», новая ссылка-приглашение, тариф и оплаты. Название, город
  // и тариф меняет владелец AYZON EXAM.
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
  <button class="more link" onclick={support}>{t('cab.planHint')}</button>
</section>

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
  }
</style>
