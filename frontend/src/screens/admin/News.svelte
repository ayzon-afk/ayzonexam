<script lang="ts">
  import { onMount } from 'svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import type { Post } from '../../lib/api'
  import { adm, when } from './admin'

  // Новости: написать, изменить, скрыть. С телефона быстрее — /post боту (первая строка — заголовок).
  let list = $state<Post[] | null>(null)
  let edit = $state<Partial<Post> | null>(null)
  let note = $state('')

  const tags: Record<string, string> = { tip: 'Совет', result: 'Результат', tournament: 'Турнир', news: 'Новость' }

  async function load() {
    list = (await adm.get<{ posts: Post[] }>('news')).posts
  }

  async function save() {
    if (!edit) return
    try {
      if (edit.id) await adm.put(`news/${edit.id}`, edit)
      else await adm.post('news', { tag: edit.tag, title: edit.title, body: edit.body })
      edit = null
      note = 'Сохранено.'
      await load()
    } catch {
      note = 'Не сохранилось: нужен заголовок и текст.'
    }
  }

  onMount(load)
</script>

<p class="hint">Чужой результат («получил 8.5») — только с согласия ученика и фото официального результата (SPEC).</p>

{#if edit}
  <section class="card stack">
    <select bind:value={edit.tag}>
      {#each Object.entries(tags) as [k, v] (k)}<option value={k}>{v}</option>{/each}
    </select>
    <input bind:value={edit.title} placeholder="Заголовок" />
    <textarea bind:value={edit.body} placeholder="Текст. Абзацы — через пустую строку." rows="10"></textarea>
    {#if edit.id}<label class="check"><input type="checkbox" bind:checked={edit.hidden} /> Скрыт</label>{/if}
    <button class="btn" onclick={save}>Сохранить</button>
    <button class="btn btn-plain" onclick={() => (edit = null)}>Отмена</button>
  </section>
{:else}
  <button class="btn" onclick={() => (edit = { tag: 'tip', title: '', body: '' })}>+ Новый пост</button>
{/if}
{#if note}<p class="hint">{note}</p>{/if}

{#if !list}
  <Skeleton rows={3} />
{:else}
  <ul class="card rows">
    {#each list as p (p.id)}
      <li>
        <button class="row" onclick={() => (edit = { ...p })}>
          <span class="grow">
            <strong>{p.title}</strong>
            <span class="tag">#{p.id} · {tags[p.tag]} · {when(p.created_at)}{p.hidden ? ' · скрыт' : ''}</span>
          </span>
        </button>
      </li>
    {/each}
  </ul>
{/if}

<style>
  strong {
    display: block;
  }

  input:not([type='checkbox']),
  select {
    width: 100%;
    min-height: 40px;
    padding: 0 10px;
    border: 1px solid var(--separator);
    border-radius: 10px;
    background: var(--section);
    color: var(--text);
    font: inherit;
  }

  .check {
    display: flex;
    gap: 8px;
    align-items: center;
  }
</style>
