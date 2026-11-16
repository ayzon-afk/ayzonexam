<script lang="ts">
  // Линия баллов 4.0–9.0 по попыткам, последняя точка выделена.
  let { points }: { points: number[] } = $props()

  const W = 300
  const H = 110
  const x = (i: number) => 8 + (i * (W - 16)) / Math.max(points.length - 1, 1)
  const y = (b: number) => 8 + ((9 - b) / 5) * (H - 16)
  const line = $derived(points.map((b, i) => `${x(i)},${y(b)}`).join(' '))
  const last = $derived(points.length - 1)
</script>

<svg viewBox="0 0 {W} {H}" role="img" aria-label={points.join(', ')}>
  {#each [5, 6, 7, 8] as b (b)}
    <line x1="0" x2={W} y1={y(b)} y2={y(b)} />
    <text x="0" y={y(b) - 3}>{b}</text>
  {/each}
  <polyline points={line} />
  <circle cx={x(last)} cy={y(points[last] ?? 4)} r="5" />
</svg>

<style>
  svg {
    width: 100%;
    height: auto;
    display: block;
  }

  line {
    stroke: var(--separator);
  }

  text {
    fill: var(--hint);
    font-size: 10px;
  }

  polyline {
    fill: none;
    stroke: var(--accent);
    stroke-width: 2.5;
    stroke-linejoin: round;
    stroke-linecap: round;
  }

  circle {
    fill: var(--accent);
  }
</style>
