<script lang="ts">
  // Рост или падение: «↑3», «↓1», для баллов — «+0.5». Ноль не показывается.
  let { n, band = false }: { n: number; band?: boolean } = $props()

  const text = $derived(band ? (n > 0 ? '+' : '−') + Math.abs(n).toFixed(1) : (n > 0 ? '↑' : '↓') + Math.abs(n))
</script>

{#if n !== 0}<span class={n > 0 ? 'up' : 'down'}>{text}</span>{/if}
