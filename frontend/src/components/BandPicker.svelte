<script lang="ts">
  import { BANDS, formatBand } from '../lib/exam'
  import { haptic } from '../lib/telegram'

  let { value = $bindable() }: { value: number | null } = $props()
</script>

<div class="grid" role="radiogroup">
  {#each BANDS as band (band)}
    <button
      role="radio"
      aria-checked={value === band}
      class:on={value === band}
      onclick={() => {
        value = band
        haptic.select()
      }}>{formatBand(band)}</button
    >
  {/each}
</div>

<style>
  .grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 8px;
  }

  button {
    min-height: 52px;
    border-radius: 12px;
    background: var(--section);
    font-size: 18px;
    font-weight: 600;
    font-variant-numeric: tabular-nums;
  }

  .on {
    background: var(--accent);
    color: var(--accent-text);
  }
</style>
