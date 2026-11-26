<script lang="ts">
  import type { Question } from '../lib/api'
  import { t } from '../lib/i18n/index.svelte'

  // Вписать слово или число на месте «____» (тесты канала, DECISIONS 2026-11-26). Проверяет сервер: регистр,
  // пробелы и точка по краям не важны. accept и mark — разбор после сдачи.
  let { n, q, value = $bindable(), accept, mark }: { n: number; q: Question; value: string | number | null | undefined; accept?: string[]; mark?: boolean } =
    $props()

  const parts = $derived(q.q.includes('____') ? q.q.split('____') : [q.q, ''])
  const checked = $derived(accept !== undefined)
  const text = $derived(typeof value === 'string' ? value : '')
</script>

<div class="card gap" class:right={checked && mark} class:wrong={checked && !mark}>
  <p>
    <b>{n}.</b>
    {parts[0]}
    {#if checked}
      <span class="got">{text || t('test.none')}</span>
    {:else}
      <input
        aria-label={t('test.gap', { n })}
        value={text}
        maxlength="60"
        autocomplete="off"
        autocapitalize="off"
        spellcheck="false"
        oninput={(e) => (value = e.currentTarget.value)}
      />
    {/if}
    {parts.slice(1).join('____')}
  </p>
  {#if checked && !mark && accept?.length}<p class="hint">{t('test.right', { a: accept.join(' / ') })}</p>{/if}
</div>

<style>
  .gap {
    display: grid;
    gap: 6px;
    line-height: 1.9;
    overflow-wrap: anywhere;
  }

  input {
    width: 9em;
    max-width: 100%;
    min-height: 36px;
    padding: 4px 10px;
    border: 1px solid var(--separator);
    border-radius: 10px;
    background: var(--surface);
    color: inherit;
    font: inherit;
  }

  input:focus {
    border-color: var(--accent);
    outline: none;
  }

  .got {
    font-weight: 600;
  }

  .right {
    box-shadow: inset 3px 0 0 #2e9d58;
  }

  .wrong {
    box-shadow: inset 3px 0 0 var(--danger);
  }
</style>
