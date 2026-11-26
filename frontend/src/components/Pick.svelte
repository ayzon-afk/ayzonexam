<script lang="ts">
  import type { Answer, Group } from '../lib/api'
  import { t } from '../lib/i18n/index.svelte'

  // «Выберите N букв» (IELTS: Questions 21–22, Choose TWO letters): один блок на несколько вопросов, каждая
  // верная буква — балл, порядок не важен. Выбранные буквы раскладываются по вопросам группы по порядку.
  // set — верные буквы после сдачи.
  let {
    nums,
    labels,
    group,
    answers = $bindable(),
    set,
  }: { nums: number[]; labels: string[]; group: Group; answers: (Answer | null)[]; set?: number[] } = $props()

  const checked = $derived(set !== undefined)
  const picked = $derived(nums.map((i) => answers[i]).filter((a): a is number => typeof a === 'number' && a >= 0))
  const limit = $derived(group.pick || nums.length)

  function toggle(i: number) {
    const next = picked.includes(i) ? picked.filter((p) => p !== i) : picked.length < limit ? [...picked, i] : picked
    const sorted = [...next].sort((a, b) => a - b)
    nums.forEach((q, k) => (answers[q] = sorted[k] ?? null))
  }
</script>

<fieldset class="card">
  <legend><b>{(nums[0] ?? 0) + 1}–{(nums[nums.length - 1] ?? 0) + 1}.</b> {t('test.pick', { n: limit })}</legend>
  <div>
    {#each labels as l, i (i)}
      <button
        role="checkbox"
        aria-checked={picked.includes(i)}
        class:on={picked.includes(i)}
        class:right={checked && set?.includes(i)}
        class:wrong={checked && picked.includes(i) && !set?.includes(i)}
        disabled={checked || (!picked.includes(i) && picked.length >= limit)}
        onclick={() => toggle(i)}
      >
        <b>{l}</b>
        {#if group.options?.[i]}<span>{group.options[i]}</span>{/if}
      </button>
    {/each}
  </div>
</fieldset>

<style>
  fieldset {
    margin: 0;
    border: 0;
    display: grid;
    gap: 10px;
  }

  legend {
    float: left;
    width: 100%;
    padding: 0;
    font-weight: 500;
  }

  div {
    display: grid;
    gap: 6px;
  }

  button {
    display: flex;
    gap: 10px;
    align-items: baseline;
    min-height: 44px;
    padding: 8px 14px;
    border: 1px solid var(--separator);
    border-radius: 12px;
    text-align: left;
    overflow-wrap: anywhere;
  }

  button:disabled:not(.right, .wrong, .on) {
    opacity: 0.5;
  }

  .on {
    border-color: var(--accent);
    background: color-mix(in srgb, var(--accent) 12%, transparent);
  }

  .right {
    border-color: #2e9d58;
    background: color-mix(in srgb, #2e9d58 14%, transparent);
  }

  .wrong {
    border-color: var(--danger);
    background: color-mix(in srgb, var(--danger) 12%, transparent);
  }
</style>
