<script lang="ts">
  import Avatar from '../../components/Avatar.svelte'
  import BandPicker from '../../components/BandPicker.svelte'
  import Head from '../../components/Head.svelte'
  import { groups, groupStats, submissions } from '../../lib/demo'
  import { formatBand } from '../../lib/exam'
  import { t, type Key } from '../../lib/i18n/index.svelte'
  import { haptic } from '../../lib/telegram'

  // Режим учителя: AI делает черновую проверку, учитель правит балл и отправляет одной кнопкой (SPEC).
  type View = 'works' | 'groups' | 'stats'
  let view = $state<View>('works')
  const views: { id: View; label: Key }[] = [
    { id: 'works', label: 'teacher.works' },
    { id: 'groups', label: 'community.groups' },
    { id: 'stats', label: 'teacher.stats' },
  ]

  let open = $state<number | null>(null)
  let works = $state(submissions.map((s) => ({ ...s, band: s.ai as number | null, sent: false })))
  let copied = $state(false)

  const crit: { key: keyof typeof groupStats.crit; name: Key }[] = [
    { key: 'ta', name: 'crit.ta.task2' },
    { key: 'cc', name: 'crit.cc' },
    { key: 'lr', name: 'crit.lr' },
    { key: 'gra', name: 'crit.gra' },
  ]
</script>

<Head title={t('teacher.title')} sub={t('teacher.lead')} />

<div class="seg" role="tablist">
  {#each views as v (v.id)}
    <button role="tab" aria-selected={view === v.id} class:on={view === v.id} onclick={() => (view = v.id)}>{t(v.label)}</button>
  {/each}
</div>

{#if view === 'works'}
  {#each works as s (s.id)}
    <section class="card stack">
      <button class="row head" onclick={() => (open = open === s.id ? null : s.id)}>
        <Avatar name={s.nick} />
        <span class="grow">
          <strong>{s.nick}</strong>
          <span class="tag">{t(s.task === 'task1' ? 'task.task1' : 'task.task2')}</span>
        </span>
        {#if s.sent}
          <span class="sent">{t('teacher.sent')}</span>
        {:else}
          <span class="tag">AI {formatBand(s.ai)}</span>
        {/if}
      </button>
      {#if open === s.id && !s.sent}
        <p class="hint">{t('teacher.draft')}</p>
        <p>{t(s.summary)}</p>
        <p class="hint">{t('teacher.yourBand')}</p>
        <BandPicker bind:value={s.band} />
        <button
          class="btn"
          disabled={s.band === null}
          onclick={() => {
            haptic.success()
            s.sent = true
            open = null
          }}>{t('teacher.send')}</button
        >
      {/if}
    </section>
  {/each}
{:else if view === 'groups'}
  <ul class="card rows">
    {#each groups as g (g.id)}
      <li class="row">
        <Avatar name={g.name} />
        <span class="grow">
          <strong>{g.name}</strong>
          <span class="tag">{t('center.students', { n: g.students })}</span>
        </span>
      </li>
    {/each}
  </ul>
  <button class="btn btn-plain" onclick={() => (copied = true)}>{t('teacher.invite')}</button>
  <button class="btn" onclick={() => (copied = true)}>{t('teacher.homework')}</button>
  {#if copied}<p class="hint">{t('demo.later')}</p>{/if}
{:else}
  <section class="card stack">
    <h2>{t('teacher.avg')}</h2>
    {#each crit as c (c.key)}
      <div class="bar">
        <span class="name">{t(c.name)}</span>
        <span class="track"><span style:width="{(groupStats.crit[c.key] / 9) * 100}%"></span></span>
        <span class="num">{groupStats.crit[c.key].toFixed(1)}</span>
      </div>
    {/each}
  </section>
  <section class="card stack">
    <h2>{t('teacher.errors')}</h2>
    <ol>
      {#each groupStats.errors as e (e)}<li>{t(e)}</li>{/each}
    </ol>
  </section>
  <p class="card saved">{t('teacher.hours', { n: groupStats.hours })}</p>
{/if}

<style>
  h2 {
    font-size: 17px;
  }

  .head {
    min-height: 0;
    padding: 0;
  }

  strong {
    display: block;
  }

  .sent {
    color: #2e9d58;
    font-size: 14px;
    font-weight: 600;
  }

  .bar {
    display: grid;
    grid-template-columns: 1fr 90px 32px;
    align-items: center;
    gap: 10px;
    font-size: 14px;
  }

  .track {
    height: 8px;
    border-radius: 4px;
    background: var(--surface);
    overflow: hidden;
  }

  .track span {
    display: block;
    height: 100%;
    background: var(--accent);
  }

  ol {
    margin: 0;
    padding-left: 20px;
    display: grid;
    gap: 8px;
  }

  .saved {
    text-align: center;
    font-weight: 600;
  }
</style>
