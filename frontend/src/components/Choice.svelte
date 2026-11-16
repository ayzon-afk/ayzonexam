<script lang="ts">
  import type { Q } from '../lib/demo-exam'

  // Вопрос с вариантами. checked — показать верный ответ и ошибку ученика.
  let { n, q, value = $bindable(), checked = false }: { n: number; q: Q; value: number | null | undefined; checked?: boolean } = $props()
</script>

<fieldset class="card">
  <legend>{n}. {q.q}</legend>
  <div role="radiogroup">
    {#each q.options as o, i (i)}
      <button
        role="radio"
        aria-checked={value === i}
        class:on={value === i}
        class:right={checked && i === q.answer}
        class:wrong={checked && value === i && i !== q.answer}
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
