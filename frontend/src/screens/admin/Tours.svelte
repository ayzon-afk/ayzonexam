<script lang="ts">
  import { onMount } from 'svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { adm, when, type Row } from './admin'

  // Турнир: итоги прошедших. Тройка лучших подтверждает результат коротким Speaking (созвон или голосовое);
  // не подтвердил — «снять», место переходит следующему. Чемпион встаёт на главную до следующего турнира.
  let list = $state<Row[] | null>(null)
  let note = $state('')

  async function load() {
    list = (await adm.get<{ tournaments: Row[] }>('tournaments')).tournaments
  }

  async function act(t: Row, body: Row, done: string) {
    try {
      list = (await adm.post<{ tournaments: Row[] }>(`tournaments/${t.id}`, body)).tournaments
      note = done
    } catch {
      note = 'Не получилось.'
    }
  }

  onMount(load)
</script>

<p class="hint">Турнир — последнее воскресенье месяца, 10:00–13:00 по Ташкенту, создаётся сам. Задания закрепляются при первом старте.</p>
{#if note}<p class="hint">{note}</p>{/if}

{#if !list}
  <Skeleton rows={3} />
{:else if list.length === 0}
  <p class="card hint">Турниров ещё не было.</p>
{:else}
  {#each list as t (t.id)}
    <section class="card stack">
      <h2>{when(t.starts_at)} · участников {t.players}</h2>
      {#if t.top.length === 0}<p class="hint">Никто не сдал обе части.</p>{/if}
      <ol>
        {#each t.top as p (p.id)}
          <li class:off={p.disqualified}>
            <b>{p.nickname}</b> ({p.first_name}) · {p.band} · {p.center ?? 'без центра'} {p.parts < 2 ? '· одна часть' : ''}
            {#if t.champion_id === p.id}<span class="ok">★ чемпион</span>{/if}
            <span class="acts">
              {#if !p.disqualified && p.parts === 2 && t.champion_id !== p.id}
                <button class="btn btn-small" onclick={() => act(t, { champion: p.id }, `${p.nickname} — чемпион.`)}>Чемпион</button>
              {/if}
              {#if p.disqualified}
                <button class="btn btn-small btn-plain" onclick={() => act(t, { requalify: p.id }, 'Возвращён.')}>Вернуть</button>
              {:else}
                <button class="btn btn-small btn-plain" onclick={() => act(t, { disqualify: p.id }, 'Снят с турнира.')}>Снять</button>
              {/if}
            </span>
          </li>
        {/each}
      </ol>
      {#if t.champion_id}<button class="btn btn-small btn-plain" onclick={() => act(t, { champion: 0 }, 'Чемпион снят.')}>Снять чемпиона</button>{/if}
    </section>
  {/each}
{/if}

<style>
  h2 {
    font-size: 16px;
  }

  ol {
    margin: 0;
    padding-left: 20px;
    display: grid;
    gap: 10px;
  }

  .off {
    opacity: 0.5;
    text-decoration: line-through;
  }

  .ok {
    color: #d69a00;
    font-weight: 700;
  }

  .acts {
    display: flex;
    gap: 6px;
    margin-top: 4px;
  }
</style>
