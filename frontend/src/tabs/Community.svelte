<script lang="ts">
  import { t, type Key } from '../lib/i18n/index.svelte'
  import { go, type Route } from '../lib/nav.svelte'

  // Сообщество (DECISIONS 2026-11-24): только то, что уже работает, — друзья, дуэли, турнир, новости, центр.
  // Лента, группы и взаимная проверка — шаги 2 и 4 (ROADMAP); их демо-экраны лежат в screens/demo.
  const items: { route: Route; title: Key; sub: Key; icon: string }[] = [
    { route: { s: 'friends' }, title: 'community.invite', sub: 'community.inviteSub', icon: '＋' },
    { route: { s: 'duel' }, title: 'community.duel', sub: 'community.duelSub', icon: '⚔' },
    { route: { s: 'tournament' }, title: 'community.tour', sub: 'community.tourSub', icon: '🏆' },
    { route: { s: 'news' }, title: 'community.news', sub: 'community.newsSub', icon: '✎' },
    { route: { s: 'centers' }, title: 'community.center', sub: 'community.centerSub', icon: '⌂' },
  ]
</script>

<h1>{t('tab.community')}</h1>
<p class="hint">{t('community.lead')}</p>

<ul class="card rows">
  {#each items as it (it.title)}
    <li>
      <button class="row" onclick={() => go(it.route)}>
        <span class="icon" aria-hidden="true">{it.icon}</span>
        <span class="grow">
          <strong>{t(it.title)}</strong>
          <span class="hint">{t(it.sub)}</span>
        </span>
        <span class="chev" aria-hidden="true">›</span>
      </button>
    </li>
  {/each}
</ul>

<p class="hint soon">{t('community.soon')}</p>

<style>
  h1 {
    font-size: 24px;
  }

  .rows {
    padding: 0;
  }

  .icon {
    width: 36px;
    height: 36px;
    border-radius: 10px;
    display: grid;
    place-items: center;
    background: var(--surface);
    color: var(--accent);
    font-size: 17px;
    flex-shrink: 0;
  }

  strong {
    display: block;
    font-weight: 600;
  }

  .soon {
    text-align: center;
    padding: 8px 16px;
  }
</style>
