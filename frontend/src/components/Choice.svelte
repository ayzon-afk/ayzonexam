<script lang="ts">
  import type { Answer, Question } from '../lib/api'

  // Вопрос с вариантами. right — верный ответ (приходит после сдачи): показать его и ошибку ученика.
  let { n, q, value = $bindable(), right }: { n: number; q: Question; value: Answer | null | undefined; right?: number } = $props()

  const checked = $derived(right !== undefined)
</script>

<fieldset class="card">
  <legend>{n}. {q.q}</legend>
  <div role="radiogroup">
    {#each q.options as o, i (i)}
      <button
        role="radio"
        aria-checked={value === i}
        class:on={value === i}
        class:right={checked && i === right}
        class:wrong={checked && value === i && i !== right}
        disabled={checked}
        onclick={() => (value = i)}>{o}</button
      >
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
    padding: 0 0 10px;
    font-weight: 500;
  }

  div {
    display: grid;
    gap: 6px;
  }

  button {
    min-height: 44px;
    padding: 8px 14px;
    border: 1px solid var(--separator);
    border-radius: 12px;
    text-align: left;
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
