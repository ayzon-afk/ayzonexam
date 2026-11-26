<script lang="ts">
  import type { Answer, Group, Question, Result, TestView } from '../lib/api'
  import Choice from './Choice.svelte'
  import Gap from './Gap.svelte'
  import Letters from './Letters.svelte'
  import Pick from './Pick.svelte'

  // from и to — показать только вопросы с этими номерами (с 0, включительно): одна часть экзамена. У каждого
  // вопроса якорь id="q{номер}" — на него переходит навигатор экзамена.
  // Вопросы задания по группам (тесты канала, DECISIONS 2026-11-26): у группы — инструкция, заметки,
  // картинка и общий список вариантов. answers — ответы ученика (null — нет ответа); key и marks — разбор
  // после сдачи, тогда всё только для чтения.
  let {
    test,
    answers = $bindable(),
    key,
    marks,
    from = 0,
    to = Infinity,
  }: { test: TestView; answers: (Answer | null)[]; key?: Result['key']; marks?: boolean[]; from?: number; to?: number } = $props()

  // labels — подписи вариантов первого вопроса блока (буквы A–H или i–x): у общего списка и «выберите N».
  type Block = { group: Group | null; items: { i: number; q: Question }[]; nums: number[]; multi: boolean; labels: string[] }

  // Подряд идущие вопросы одной группы — один блок.
  const blocks = $derived.by(() => {
    const out: Block[] = []
    test.questions.forEach((q, i) => {
      if (i < from || i > to) return
      const g = q.group ?? 0
      const last = out[out.length - 1]
      if (g && last && last.group === test.groups?.[g - 1]) {
        last.items.push({ i, q })
        last.nums.push(i)
      } else
        out.push({ group: g ? (test.groups?.[g - 1] ?? null) : null, items: [{ i, q }], nums: [i], multi: q.kind === 'multi', labels: q.options })
    })
    return out
  })

  const num = (k: Result['key'][number] | undefined) => (typeof k === 'number' ? k : undefined)
</script>

{#each blocks as b (b.nums[0])}
  {#if b.group}
    <section class="card group">
      <h2>{b.group.title}</h2>
      {#if b.group.text}
        {#each b.group.text.split('\n') as line, i (i)}<p>{line}</p>{/each}
      {/if}
      {#if b.group.image}<img src={b.group.image} alt="" loading="lazy" />{/if}
      {#if b.group.options?.length && !b.multi}
        <ul>
          {#each b.group.options as o, i (i)}
            <li><b>{b.labels[i] ?? ''}</b> {o}</li>
          {/each}
        </ul>
      {/if}
    </section>
  {/if}
  {#if b.multi && b.group}
    <div class="item" id="q{(b.nums[0] ?? 0) + 1}" data-q={b.nums[0]}>
      {#each b.nums.slice(1) as i (i)}<span class="anchor" id="q{i + 1}"></span>{/each}
      <Pick nums={b.nums} labels={b.labels} group={b.group} bind:answers set={key ? (key[b.nums[0] ?? 0] as number[]) : undefined} />
    </div>
  {:else}
    {#each b.items as { i, q } (i)}
      <div class="item" id="q{i + 1}" data-q={i}>
        {#if q.kind === 'text'}
          <Gap n={i + 1} {q} bind:value={answers[i]} accept={key ? (key[i] as string[]) : undefined} mark={marks?.[i]} />
        {:else if b.group?.options?.length}
          <Letters n={i + 1} {q} bind:value={answers[i]} right={num(key?.[i])} />
        {:else}
          <Choice n={i + 1} {q} bind:value={answers[i]} right={num(key?.[i])} />
        {/if}
      </div>
    {/each}
  {/if}
{/each}

<style>
  .item {
    position: relative;
    display: grid;
    scroll-margin-top: 12px;
  }

  .anchor {
    position: absolute;
    top: 0;
  }

  .group {
    display: grid;
    gap: 8px;
    line-height: 1.5;
    overflow-wrap: anywhere;
  }

  h2 {
    font-size: 16px;
  }

  img {
    display: block;
    max-width: 100%;
    height: auto;
    border-radius: 10px;
  }

  ul {
    display: grid;
    gap: 4px;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  li b {
    display: inline-block;
    min-width: 1.6em;
  }
</style>
