<script lang="ts">
  import Avatar from '../../components/Avatar.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { fmtDate } from '../../lib/date'
  import { t } from '../../lib/i18n/index.svelte'
  import Invite from './Invite.svelte'
  import { cab, type SectionProps, type Student } from './cabinet'

  // Обзор центра: главные числа, ссылка-приглашение, заявки и лучшие ученики.
  let { me, data, goto }: SectionProps = $props()

  const c = $derived(data.center)
  const admin = $derived(data.role === 'admin')
  let top = $state<Student[] | null>(null)
  cab
    .students()
    .then((r) => (top = r.students.filter((s) => s.band !== null).slice(0, 5)))
    .catch(() => (top = []))
</script>

<div class="tiles">
  <div class="card tile">
    <span class="hint">{t('cab.students')}</span>
    <strong class="num">{c.students}</strong>
    {#if admin && c.seats}<span class="hint">{t('cab.ofSeats', { n: c.seats })}</span>{/if}
  </div>
  <div class="card tile">
    <span class="hint">{t('cab.week')}</span>
    <strong class="num">{c.active_week}</strong>
  </div>
  <div class="card tile">
    <span class="hint">{t('cab.avg')}</span>
    <strong class="num">{c.avg !== null ? c.avg.toFixed(1) : '—'}</strong>
    <span class="hint">{c.avg !== null ? t('cab.scored', { n: c.scored }) : t('cab.noAvg')}</span>
  </div>
  <!-- Учитель без центра в рейтинг центров не входит — места у него нет. -->
  {#if c.kind !== 'teacher'}
    <div class="card tile">
      <span class="hint">{t('cab.place')}</span>
      <strong class="num">
        {c.place ?? '—'}
        {#if c.delta > 0}<span class="up">↑{c.delta}</span>{:else if c.delta < 0}<span class="down">↓{-c.delta}</span>{/if}
      </strong>
      {#if c.place}<span class="hint">{t('cab.of', { n: c.centers })}</span>{/if}
    </div>
  {/if}
  {#if admin}
    <div class="card tile">
      <span class="hint">{t('cab.paidUntil')}</span>
      <strong class="date">{c.paid_until ? fmtDate(c.paid_until) : t('cab.notPaid')}</strong>
    </div>
  {/if}
</div>

{#if admin && c.seats > 0 && c.students > c.seats}
  <p class="card warn">{t('cab.over')}</p>
{/if}

{#if c.pending}
  <button class="card row pending" onclick={() => goto('requests')}>
    <span class="grow">{t('cab.pending')}</span>
    <strong class="num">{c.pending}</strong>
    <span class="chev" aria-hidden="true">›</span>
  </button>
{/if}

<div class="two">
  <section class="card">
    <h2>{t('cab.invite')}</h2>
    <Invite {me} name={c.name} code={c.invite_code} />
  </section>

  <section class="card">
    <div class="sect">
      <h2>{t('cab.best')}</h2>
      <button class="more" onclick={() => goto('students')}>{t('cab.all')}</button>
    </div>
    {#if !top}
      <Skeleton rows={3} />
    {:else if top.length === 0}
      <p class="hint">{c.students ? t('cab.noAvg') : t('cab.empty')}</p>
    {:else}
      <ol class="rows best">
        {#each top as s, i (s.id)}
          <li class="row">
            <span class="place">{i + 1}</span>
            <Avatar name={s.nickname} size={32} />
            <span class="grow nick">{s.nickname}</span>
            <strong class="num">{s.band?.toFixed(1)}</strong>
          </li>
        {/each}
      </ol>
    {/if}
  </section>
</div>

<style>
  .tiles {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(min(100%, 132px), 1fr));
    gap: 10px;
  }

  .tile {
    display: grid;
    align-content: start;
    gap: 4px;
    min-width: 0;
  }

  .tile .num {
    font-size: 28px;
    line-height: 1.1;
  }

  .date {
    font-size: 18px;
  }

  .warn {
    color: var(--danger);
    font-size: 14px;
  }

  .pending {
    min-height: 0;
    padding: 14px 16px;
  }

  .two {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(min(100%, 340px), 1fr));
    gap: 12px;
    align-items: start;
  }

  section {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: 12px;
    min-width: 0;
  }

  h2 {
    font-size: 17px;
  }

  .sect {
    margin: 0;
  }

  .best {
    padding: 0;
  }

  .best .row {
    min-height: 48px;
    padding: 6px 0;
  }

  .place {
    width: 20px;
    color: var(--hint);
    font-variant-numeric: tabular-nums;
    text-align: right;
  }

  .nick {
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }
</style>
