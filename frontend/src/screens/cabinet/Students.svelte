<script lang="ts">
  import Avatar from '../../components/Avatar.svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { fmtDate } from '../../lib/date'
  import { t } from '../../lib/i18n/index.svelte'
  import { haptic } from '../../lib/telegram'
  import { PARTS, cab, download, toCSV, view, type Filter, type SectionProps, type Sort, type Student } from './cabinet'

  // Ученики центра: поиск, отбор, сортировка, CSV. Аккаунт центра делает учителем и убирает из центра —
  // не себя и не другой аккаунт центра (их назначает владелец). Видны только ник, аватар и результаты.
  let { data, onchange }: SectionProps = $props()

  const admin = $derived(data.role === 'admin')
  let list = $state<Student[] | null>(null)
  let meID = $state(0)
  let failed = $state(false)
  let q = $state('')
  let filter = $state<Filter>('all')
  let sort = $state<Sort>('band')
  let asking = $state<number | null>(null)
  let busy = $state<number | null>(null)
  let error = $state('')

  async function load() {
    failed = false
    try {
      const r = await cab.students()
      list = r.students
      meID = r.me
    } catch {
      failed = true
    }
  }
  load()

  const shown = $derived(list ? view(list, q, filter, sort) : [])
  const filters: { id: Filter; key: 'cab.fAll' | 'cab.fNoBand' | 'cab.fIdle' }[] = [
    { id: 'all', key: 'cab.fAll' },
    { id: 'noBand', key: 'cab.fNoBand' },
    { id: 'idle', key: 'cab.fIdle' },
  ]

  const manageable = (s: Student) => admin && s.id !== meID && s.role !== 'admin'

  async function act(s: Student, run: () => Promise<void>, after: (l: Student[]) => Student[]) {
    busy = s.id
    error = ''
    try {
      await run()
      haptic.success()
      list = after(list ?? [])
    } catch {
      haptic.error()
      error = t('cab.failed')
    } finally {
      busy = null
      asking = null
    }
  }

  const toggleTeacher = (s: Student) =>
    act(
      s,
      () => cab.teacher(s.id, s.role !== 'teacher'),
      (l) => l.map((x) => (x.id === s.id ? { ...x, role: x.role === 'teacher' ? null : 'teacher' } : x)),
    )

  const remove = (s: Student) =>
    act(
      s,
      () => cab.remove(s.id),
      (l) => {
        onchange({ ...data, center: { ...data.center, students: data.center.students - 1 } })
        return l.filter((x) => x.id !== s.id)
      },
    )

  function csv() {
    const head = [t('cab.colStudent'), t('cab.colBand'), ...PARTS.map((p) => p.name), t('cab.colMonth'), t('cab.colLast')]
    download(`${data.center.name.replace(/[^\p{L}\p{N}]+/gu, '-')}-students.csv`, toCSV(shown, head))
  }

  const last = (s: Student) => (s.last_at ? fmtDate(s.last_at) : t('cab.never'))
</script>

<div class="tools">
  <input type="search" bind:value={q} placeholder={t('cab.search')} aria-label={t('cab.search')} autocomplete="off" />
  <select bind:value={sort} aria-label={t('cab.sortBand')}>
    <option value="band">{t('cab.sortBand')}</option>
    <option value="active">{t('cab.sortActive')}</option>
    <option value="name">{t('cab.sortName')}</option>
  </select>
  <button class="btn btn-small btn-plain" onclick={csv} disabled={!shown.length}>{t('cab.csv')}</button>
</div>
<div class="seg" role="radiogroup">
  {#each filters as f (f.id)}
    <button role="radio" aria-checked={filter === f.id} class:on={filter === f.id} onclick={() => (filter = f.id)}>{t(f.key)}</button>
  {/each}
</div>

{#if error}<p class="error" role="alert">{error}</p>{/if}

{#if failed}
  <p class="error">{t('cab.loadFail')}</p>
  <button class="btn btn-plain" onclick={load}>{t('app.retry')}</button>
{:else if !list}
  <Skeleton rows={6} />
{:else if list.length === 0}
  <p class="card hint">{t('cab.empty')}</p>
{:else if shown.length === 0}
  <p class="card hint">{t('cab.nothing')}</p>
{:else}
  <div class="wrap">
    <div class="table card" class:admin role="table" aria-label={t('cab.sStudents')}>
      <div class="tr head" role="row">
        <span role="columnheader">{t('cab.colStudent')}</span>
        <span role="columnheader" class="c">{t('cab.colBand')}</span>
        {#each PARTS as p (p.kind)}<span role="columnheader" class="c part" title={p.name}>{p.short}</span>{/each}
        <span role="columnheader" class="c">{t('cab.colMonth')}</span>
        <span role="columnheader">{t('cab.colLast')}</span>
        {#if admin}<span role="columnheader" class="sr">{t('cab.actions')}</span>{/if}
      </div>
      {#each shown as s (s.id)}
        <div class="tr" role="row">
          <span class="who" role="cell">
            <Avatar name={s.nickname} size={34} />
            <span class="name">
              <strong>{s.nickname}</strong>
              {#if s.role}<span class="tag">{s.role === 'admin' ? t('cab.roleAdmin') : t('cab.roleTeacher')}</span>{/if}
            </span>
          </span>
          <span class="band" role="cell"><strong class="num">{s.band !== null ? s.band.toFixed(1) : '—'}</strong></span>
          <span class="parts" role="cell">
            {#each PARTS as p (p.kind)}
              <span class="part" title={p.name}><em>{p.short}</em>{s.parts[p.kind]?.toFixed(1) ?? '—'}</span>
            {/each}
          </span>
          <span class="meta">
            <span class="month" role="cell"><span class="label">{t('cab.tasks', { n: s.month })}</span><span class="val">{s.month}</span></span>
            <span class="last" role="cell">{last(s)}</span>
          </span>
          {#if admin}
            <span class="acts" role="cell">
              {#if manageable(s)}
                {#if asking === s.id}
                  <button class="btn btn-small danger" disabled={busy === s.id} onclick={() => remove(s)}>{t('cab.remove')}</button>
                  <button class="btn btn-small btn-plain" onclick={() => (asking = null)}>×</button>
                {:else}
                  <button class="btn btn-small btn-plain" disabled={busy === s.id} onclick={() => toggleTeacher(s)}>
                    {s.role === 'teacher' ? t('cab.unTeacher') : t('cab.makeTeacher')}
                  </button>
                  <button class="btn btn-small btn-plain out" disabled={busy === s.id} aria-label={t('cab.removeAsk', { name: s.nickname })} onclick={() => (asking = s.id)}>
                    {t('cab.remove')}
                  </button>
                {/if}
              {/if}
            </span>
          {/if}
        </div>
      {/each}
    </div>
  </div>
{/if}

<style>
  .tools {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  input,
  select {
    min-height: 40px;
    padding: 0 12px;
    border: 1px solid var(--separator);
    border-radius: 10px;
    background: var(--section);
    color: var(--text);
    font: inherit;
    font-size: 15px;
  }

  input {
    flex: 1 1 200px;
    min-width: 0;
  }

  select {
    flex: 0 1 auto;
    min-width: 0;
  }

  .tools .btn-small {
    min-height: 40px;
  }

  /* Таблица перестраивается по ширине самого списка (не окна): в Mini App и в узкой колонке — карточки. */
  .wrap {
    container-type: inline-size;
  }

  .table {
    padding: 0 16px;
  }

  .tr {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    grid-template-areas:
      'who band'
      'parts parts'
      'meta meta'
      'acts acts';
    align-items: center;
    gap: 8px 12px;
    padding: 12px 0;
  }

  .tr + .tr {
    border-top: 1px solid var(--separator);
  }

  .head {
    display: none;
  }

  .who {
    grid-area: who;
    display: flex;
    align-items: center;
    gap: 10px;
    min-width: 0;
  }

  .name {
    min-width: 0;
  }

  .name strong {
    display: block;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }

  .band {
    grid-area: band;
    font-size: 20px;
  }

  .parts {
    grid-area: parts;
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .parts .part {
    padding: 3px 8px;
    border-radius: 8px;
    background: var(--surface);
    font-size: 13px;
    font-variant-numeric: tabular-nums;
  }

  em {
    margin-right: 4px;
    color: var(--hint);
    font-style: normal;
    font-weight: 600;
  }

  .meta {
    grid-area: meta;
    display: flex;
    flex-wrap: wrap;
    justify-content: space-between;
    gap: 2px 12px;
    color: var(--hint);
    font-size: 13px;
  }

  .month .val {
    display: none;
  }

  .acts {
    grid-area: acts;
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .acts:empty {
    display: none;
  }

  .acts .btn-plain {
    background: var(--surface);
  }

  /* Карточка: кнопки делят строку поровну, не влезли — каждая на свою строку во всю ширину. */
  .acts .btn-small {
    flex: 1 1 auto;
  }

  .out {
    color: var(--danger) !important;
  }

  .danger {
    background: var(--danger);
  }

  .sr {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip-path: inset(50%);
  }

  /* Широкий список — таблица: столбцы частей, заданий и даты. Ширины постоянные — столбцы ровные во всех
     строках, даже где кнопок нет (свой аккаунт, другой аккаунт центра). */
  @container (min-width: 980px) {
    .tr {
      grid-template-columns: minmax(160px, 1.6fr) 60px repeat(4, 44px) 84px minmax(100px, 1fr);
      grid-template-areas: none;
      gap: 8px;
    }

    .admin .tr {
      grid-template-columns: minmax(160px, 1.6fr) 60px repeat(4, 44px) 84px minmax(100px, 1fr) 316px;
    }

    .acts .btn-small {
      flex: none;
    }

    .tr > * {
      grid-area: auto;
    }

    .head {
      display: grid;
      padding: 12px 0 10px;
      color: var(--hint);
      font-size: 12px;
      font-weight: 600;
    }

    .c {
      text-align: center;
    }

    .band {
      font-size: 17px;
      text-align: center;
    }

    .parts {
      display: contents;
    }

    .parts .part {
      padding: 0;
      background: none;
      text-align: center;
      font-size: 14px;
    }

    .parts em {
      display: none;
    }

    .meta {
      display: contents;
    }

    .month {
      color: var(--text);
      font-size: 14px;
      text-align: center;
    }

    .last {
      color: var(--hint);
      font-size: 14px;
    }

    .month .label {
      display: none;
    }

    .month .val {
      display: inline;
    }

    .acts {
      flex-wrap: nowrap;
      justify-content: flex-end;
    }

    .acts:empty {
      display: block;
    }
  }
</style>
