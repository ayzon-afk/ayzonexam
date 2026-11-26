<script lang="ts">
  import { onMount } from 'svelte'
  import Delta from '../../components/Delta.svelte'
  import Head from '../../components/Head.svelte'
  import ShareCard from '../../components/ShareCard.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { api, type MockPart, type MockView } from '../../lib/api'
  import { fmtDate } from '../../lib/date'
  import { formatBand } from '../../lib/exam'
  import { t, type Key } from '../../lib/i18n/index.svelte'
  import { go } from '../../lib/nav.svelte'

  // Отчёт mock: Overall Band, баллы частей со стрелками к прошлому mock, слабые места, план на 2 недели.
  let { r }: { r: { id: number } } = $props()

  let m = $state<MockView | null>(null)
  let failed = $state(false)

  const names: Record<MockPart, string> = { listening: 'Listening', reading: 'Reading', writing: 'Writing', speaking: 'Speaking' }
  const order: MockPart[] = ['listening', 'reading', 'writing', 'speaking']
  // План: три самые слабые части по 3–4 дня, в конце — снова mock.
  const days = ['1–3', '4–6', '7–10', '11–14']

  onMount(async () => {
    try {
      m = await api.mock(r.id)
    } catch {
      failed = true
    }
  })

  const plan = $derived(m ? [...m.plan.slice(0, 3).map((p) => `plan.${p}` as Key), 'plan.mock' as Key] : [])
</script>

<Head title={t('report.title')} sub={m ? fmtDate(m.finished_at ?? m.started_at) : ''} demo={false} />

{#if failed}
  <p class="error">{t('app.error')}</p>
{:else if !m}
  <Skeleton rows={4} />
{:else if m.status !== 'done' || m.overall === null}
  <p class="card">{t('report.notYet')}</p>
  <button class="btn" onclick={() => go({ s: 'mock' })}>{t('mock.open')}</button>
{:else}
  <section class="card overall">
    <p class="hint">Overall Band</p>
    <p class="num band">{formatBand(m.overall)}</p>
    <div class="parts">
      {#each order as p (p)}
        {@const b = m.parts[p].band}
        <div>
          <p class="num">{b === null ? '—' : formatBand(b)}</p>
          <p class="hint">{names[p]}</p>
          {#if b !== null && m.prev[p] !== undefined}<Delta n={b - m.prev[p]!} band />{/if}
        </div>
      {/each}
    </div>
  </section>

  <!-- Честность по частям (DECISIONS 2026-11-25): с 3 нарушений часть не идёт в рейтинг. -->
  <section class="card stack">
    <h2>{t('mock.honesty')}</h2>
    <ul class="rows honesty">
      {#each order as p (p)}
        {@const v = m.parts[p].violations ?? 0}
        <li class="row">
          <span class="grow">
            {names[p]}
            <span class:bad={v >= 3} class="tag">{v ? t('mock.viol', { n: v }) : t('mock.clean')}{v >= 3 ? ` · ${t('mock.unranked')}` : ''}</span>
          </span>
        </li>
      {/each}
    </ul>
  </section>

  {#if m.weak.length}
    <section class="card stack">
      <h2>{t('report.weak')}</h2>
      <ul>
        <!-- Тип вопроса без текста (новый тип из админки) — не пустой пункт, а пропуск. -->
        {#each m.weak.filter((w) => t(`weak.${w.part}.${w.type}` as Key)) as w (w.part + w.type)}
          <li>{t(`weak.${w.part}.${w.type}` as Key, { pct: w.pct })}</li>
        {/each}
      </ul>
    </section>
  {/if}

  <section class="card stack">
    <h2>{t('report.plan')}</h2>
    <ol>
      {#each plan as p, i (i)}<li><b>{t('plan.days', { d: days[i]! })}</b> {t(p)}</li>{/each}
    </ol>
  </section>

  <div class="sect"><h2>{t('report.story')}</h2></div>
  <ShareCard
    big={formatBand(m.overall)}
    title="Overall Band · IELTS-style mock"
    sub={order.map((p) => `${names[p].slice(0, 1)} ${formatBand(m!.parts[p].band ?? 0)}`).join('  ')}
    text={t('share.report', { band: formatBand(m.overall) })}
  />
  <p class="hint">{t('result.disclaimer')}</p>
{/if}

<style>
  h2 {
    font-size: 17px;
  }

  .overall {
    text-align: center;
    display: grid;
    gap: 4px;
  }

  .band {
    font-size: 52px;
    line-height: 1.1;
  }

  .parts {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 4px;
    margin-top: 8px;
  }

  .parts .num {
    font-size: 20px;
  }

  ul,
  ol {
    margin: 0;
    padding-left: 20px;
    display: grid;
    gap: 8px;
  }

  .honesty {
    padding: 0;
  }

  .honesty .tag {
    display: block;
  }

  .bad {
    color: var(--danger);
    font-weight: 600;
  }
</style>
