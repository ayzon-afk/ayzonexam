<script lang="ts">
  import Avatar from '../components/Avatar.svelte'
  import Demo from '../components/Demo.svelte'
  import { feed, groups, mine } from '../lib/demo'
  import { t, type Key } from '../lib/i18n/index.svelte'
  import { go } from '../lib/nav.svelte'
  import { haptic } from '../lib/telegram'

  type View = 'feed' | 'groups' | 'review'
  let view = $state<View>('feed')
  const views: { id: View; label: Key }[] = [
    { id: 'feed', label: 'community.feed' },
    { id: 'groups', label: 'community.groups' },
    { id: 'review', label: 'community.review' },
  ]

  // Правила сообщества — при первом входе (SPEC, «Безопасность»). Помнит браузер: это удобство, не данные.
  const RULES = 'ayzon.rules'
  let rulesSeen = $state(read())
  function read(): boolean {
    try {
      return localStorage.getItem(RULES) === '1'
    } catch {
      return false
    }
  }
  function acceptRules() {
    rulesSeen = true
    try {
      localStorage.setItem(RULES, '1')
    } catch {
      // приватный режим: правила покажутся снова — не беда
    }
  }

  let liked = $state<Record<number, boolean>>({})
</script>

<h1>{t('tab.community')} <Demo /></h1>

{#if !rulesSeen}
  <section class="card stack">
    <h2>{t('rules.title')}</h2>
    <ul>
      <li>{t('rules.1')}</li>
      <li>{t('rules.2')}</li>
      <li>{t('rules.3')}</li>
    </ul>
    <button class="btn" onclick={acceptRules}>{t('rules.ok')}</button>
  </section>
{/if}

<div class="seg" role="tablist">
  {#each views as v (v.id)}
    <button role="tab" aria-selected={view === v.id} class:on={view === v.id} onclick={() => (view = v.id)}>{t(v.label)}</button>
  {/each}
</div>

{#if view === 'feed'}
  {#each feed as p (p.id)}
    <article class="card post">
      <header>
        <Avatar name={p.nick} />
        <span class="grow">
          <strong>{p.nick}</strong>
          {#if p.teacher}<span class="teacher">{t('community.teacher')}</span>{/if}
          <span class="tag">{t('time.hoursAgo', { n: p.hours })}</span>
        </span>
      </header>
      <p>{t(p.text)}</p>
      <footer>
        <button
          class:liked={liked[p.id]}
          aria-pressed={!!liked[p.id]}
          onclick={() => {
            haptic.select()
            liked[p.id] = !liked[p.id]
          }}>♥ {p.likes + (liked[p.id] ? 1 : 0)}</button
        >
        <span class="hint">{t('community.comments', { n: p.comments })}</span>
        <button class="hint report">{t('community.report')}</button>
      </footer>
    </article>
  {/each}
{:else if view === 'groups'}
  <ul class="card rows">
    {#each groups as g (g.id)}
      <li>
        <button class="row" onclick={() => go({ s: 'group', id: g.id })}>
          <Avatar name={g.name} />
          <span class="grow">
            <strong>{g.name}</strong>
            <span class="tag">{g.teacher} · {t('center.students', { n: g.students })}</span>
          </span>
          <span class="chev" aria-hidden="true">›</span>
        </button>
      </li>
    {/each}
  </ul>
  <p class="hint">{t('community.joinHint')}</p>
  <button class="card row teacherMode" onclick={() => go({ s: 'teacher' })}>
    <span class="grow">
      <strong>{t('teacher.title')}</strong>
      <span class="tag">{t('teacher.lead')}</span>
    </span>
    <span class="chev" aria-hidden="true">›</span>
  </button>
{:else}
  <section class="card stack">
    <h2>{t('review.title')}</h2>
    <p class="hint">{t('review.lead')}</p>
    <p>{t('review.stats', { n: mine.checked, b: mine.bonus })}</p>
    <button class="btn" onclick={() => go({ s: 'review' })}>{t('review.start')}</button>
  </section>
{/if}

<style>
  h1 {
    font-size: 26px;
  }

  h2 {
    font-size: 18px;
  }

  ul:not(.rows) {
    margin: 0;
    padding-left: 20px;
    display: grid;
    gap: 6px;
  }

  .post {
    display: grid;
    gap: 10px;
  }

  .post header {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .grow strong {
    display: block;
  }

  .teacher {
    display: inline-block;
    padding: 0 6px;
    border-radius: 6px;
    background: color-mix(in srgb, var(--accent) 15%, transparent);
    color: var(--accent);
    font-size: 12px;
    font-weight: 600;
  }

  .post footer {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .post footer button {
    min-height: 32px;
    font-size: 14px;
  }

  .liked {
    color: #e0245e;
  }

  .report {
    margin-left: auto;
  }

  .teacherMode {
    padding: 0 16px;
  }
</style>
