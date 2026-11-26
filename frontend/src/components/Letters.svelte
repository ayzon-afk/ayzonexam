<script lang="ts">
  import type { Question } from '../lib/api'

  // Вопрос с буквами из общего списка группы (сопоставление, план, карта): тексты вариантов — в группе выше,
  // здесь только ряд букв. right — верная буква после сдачи.
  let { n, q, value = $bindable(), right }: { n: number; q: Question; value: number | string | null | undefined; right?: number } = $props()

  const checked = $derived(right !== undefined)
</script>

<fieldset class="card">
  <legend><b>{n}.</b> {q.q}</legend>
  <div role="radiogroup">
    {#each q.options as o, i (i)}
      <button
        role="radio"
        aria-checked={value === i}
        aria-label="{n}: {o}"
        class:on={value === i}
        class:right={checked && i === right}
        class:wrong={checked && value === i && i !== right}
        disabled={checked}
        onclick={() => (value = value === i ? null : i)}>{o}</button
      >
    {/each}
  </div>
</fieldset>

<style>
  fieldset {
    margin: 0;
    border: 0;
    display: grid;
    gap: 8px;
  }

  legend {
    float: left;
    width: 100%;
    padding: 0;
    overflow-wrap: anywhere;
  }

  div {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  button {
    min-width: 44px;
    min-height: 44px;
    padding: 0 10px;
    border: 1px solid var(--separator);
    border-radius: 12px;
    font-weight: 600;
  }

  button:disabled {
    cursor: default;
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
