<script lang="ts">
  import type { Chart } from '../lib/api'

  // График задания Task 1 из данных (DECISIONS 2026-11-27): линейный, столбчатый, круговые диаграммы, таблица и
  // схема процесса — без библиотек. Рисуется в настоящих пикселях ширины блока, чтобы подписи не мельчили на
  // телефоне.
  let { chart }: { chart: Chart } = $props()

  const COLORS = ['#1d8fe1', '#f08c2e', '#2ea56a', '#9b59d0', '#d84a5a', '#6b7a8c', '#c9a227', '#3bb5b0']

  let w = $state(320)
  // Столбцы — категории: подпись нужна у каждой. Не помещаются по ширине — наискосок, место снизу — под самую длинную.
  const longest = $derived(Math.max(...(chart.x ?? ['']).map((x) => x.length)))
  const tilt = $derived(chart.type === 'bar' && longest * 6.5 > (w - 56) / Math.max(chart.x?.length ?? 1, 1))
  const pad = $derived({ l: 44, r: 12, t: 12, b: tilt ? Math.round(16 + longest * 6.5 * 0.64) : 28 })
  const h = $derived(Math.round(Math.min(Math.max(w * 0.58, 200), 320)) + (tilt ? pad.b - 28 : 0))

  const fmt = (v: number) => (Math.abs(v) >= 1000 ? v.toLocaleString('en') : String(Math.round(v * 100) / 100))

  // Шкала: «круглый» верх и 4–5 делений.
  const scale = $derived.by(() => {
    const vals = (chart.series ?? []).flatMap((s) => s.values)
    const top = Math.max(...vals, 0)
    const raw = top / 5 || 1
    const mag = 10 ** Math.floor(Math.log10(raw))
    const step = [1, 2, 2.5, 5, 10].map((m) => m * mag).find((s) => s >= raw) ?? raw
    const max = Math.ceil(top / step) * step || step
    return { max, ticks: Array.from({ length: Math.round(max / step) + 1 }, (_, i) => i * step) }
  })

  const pw = $derived(Math.max(w - pad.l - pad.r, 40))
  const ph = $derived(h - pad.t - pad.b)
  const y = (v: number) => pad.t + ph - (v / scale.max) * ph
  const n = $derived(chart.x?.length ?? 0)
  // Подписи оси x: если тесно — через одну или реже.
  const every = $derived(tilt ? 1 : Math.max(1, Math.ceil((n * 44) / pw)))
  const xLine = (i: number) => pad.l + (n > 1 ? (i / (n - 1)) * pw : pw / 2)
  const band = $derived(pw / Math.max(n, 1))
  const xBand = (i: number) => pad.l + band * i + band / 2

  // Доля круга: дуга от a0 до a1 (доли от 0 до 1).
  function arc(cx: number, cy: number, r: number, a0: number, a1: number): string {
    if (a1 - a0 >= 0.9999) return `M ${cx - r} ${cy} a ${r} ${r} 0 1 0 ${2 * r} 0 a ${r} ${r} 0 1 0 ${-2 * r} 0`
    const p = (a: number) => [cx + r * Math.sin(2 * Math.PI * a), cy - r * Math.cos(2 * Math.PI * a)]
    const [x0, y0] = p(a0)
    const [x1, y1] = p(a1)
    return `M ${cx} ${cy} L ${x0} ${y0} A ${r} ${r} 0 ${a1 - a0 > 0.5 ? 1 : 0} 1 ${x1} ${y1} Z`
  }

  const pieSize = 150
</script>

<figure class="chart">
  <figcaption>
    <b>{chart.title}</b>
    {#if chart.unit}<span class="hint">{chart.unit}</span>{/if}
  </figcaption>

  {#if chart.type === 'line' || chart.type === 'bar'}
    <div class="plot" bind:clientWidth={w}>
      <svg width={w} height={h} role="img" aria-label={chart.title}>
        {#each scale.ticks as v (v)}
          <line class="grid" x1={pad.l} x2={pad.l + pw} y1={y(v)} y2={y(v)} />
          <text class="tick" x={pad.l - 6} y={y(v) + 4} text-anchor="end">{fmt(v)}</text>
        {/each}
        {#each chart.x ?? [] as label, i (i)}
          {#if tilt}
            {@const lx = xBand(i)}
            {@const ly = pad.t + ph + 14}
            <text class="tick" x={lx} y={ly} text-anchor="end" transform="rotate(-40 {lx} {ly})">{label}</text>
          {:else if i % every === 0}
            <text class="tick" x={chart.type === 'line' ? xLine(i) : xBand(i)} y={h - 8} text-anchor="middle">{label}</text>
          {/if}
        {/each}
        {#if chart.type === 'line'}
          {#each chart.series ?? [] as s, k (k)}
            <polyline fill="none" stroke={COLORS[k % COLORS.length]} stroke-width="2.5" points={s.values.map((v, i) => `${xLine(i)},${y(v)}`).join(' ')} />
            {#each s.values as v, i (i)}<circle cx={xLine(i)} cy={y(v)} r="3" fill={COLORS[k % COLORS.length]} />{/each}
          {/each}
        {:else}
          {@const m = (chart.series ?? []).length}
          {@const bw = Math.max(Math.min((band * 0.78) / m, 34), 3)}
          {#each chart.series ?? [] as s, k (k)}
            {#each s.values as v, i (i)}
              <rect x={xBand(i) - (bw * m) / 2 + k * bw} y={y(v)} width={bw - 1} height={Math.max(pad.t + ph - y(v), 0)} fill={COLORS[k % COLORS.length]} />
            {/each}
          {/each}
        {/if}
        <line class="axis" x1={pad.l} x2={pad.l + pw} y1={pad.t + ph} y2={pad.t + ph} />
      </svg>
    </div>
    <ul class="legend">
      {#each chart.series ?? [] as s, k (k)}<li><i style:background={COLORS[k % COLORS.length]}></i>{s.name}</li>{/each}
    </ul>
  {:else if chart.type === 'pie'}
    <div class="pies">
      {#each chart.pies ?? [] as p, j (j)}
        {@const total = p.parts.reduce((a, s) => a + s.value, 0) || 1}
        <div class="pie">
          {#if p.name}<b>{p.name}</b>{/if}
          <svg width={pieSize} height={pieSize} role="img" aria-label={`${chart.title} ${p.name}`}>
            {#each p.parts as s, k (k)}
              {@const a0 = p.parts.slice(0, k).reduce((a, x) => a + x.value, 0) / total}
              <path d={arc(pieSize / 2, pieSize / 2, pieSize / 2 - 2, a0, a0 + s.value / total)} fill={COLORS[k % COLORS.length]} stroke="var(--section)" stroke-width="1.5" />
            {/each}
          </svg>
          <ul class="legend col">
            {#each p.parts as s, k (k)}<li><i style:background={COLORS[k % COLORS.length]}></i>{s.name} — {fmt(s.value)}{chart.unit === '%' ? '%' : ''}</li>{/each}
          </ul>
        </div>
      {/each}
    </div>
  {:else if chart.type === 'table'}
    <div class="scroll">
      <table>
        <thead><tr>{#each chart.columns ?? [] as c, i (i)}<th>{c}</th>{/each}</tr></thead>
        <tbody>
          {#each chart.rows ?? [] as row, i (i)}<tr>{#each row as cell, k (k)}<td class:val={k > 0}>{cell}</td>{/each}</tr>{/each}
        </tbody>
      </table>
    </div>
  {:else if chart.type === 'process'}
    <ol class="steps">
      {#each chart.steps ?? [] as s, i (i)}<li><span class="n">{i + 1}</span><span>{s}</span></li>{/each}
    </ol>
  {/if}
</figure>

<style>
  .chart {
    margin: 0;
    display: grid;
    gap: 10px;
    min-width: 0;
  }

  figcaption {
    display: grid;
    gap: 2px;
  }

  .plot {
    width: 100%;
    min-width: 0;
  }

  svg {
    display: block;
    overflow: visible;
  }

  .grid {
    stroke: var(--separator);
  }

  .axis {
    stroke: var(--hint);
  }

  .tick {
    font-size: 11px;
    fill: var(--hint);
  }

  .legend {
    display: flex;
    flex-wrap: wrap;
    gap: 4px 14px;
    margin: 0;
    padding: 0;
    list-style: none;
    font-size: 14px;
  }

  .legend.col {
    display: grid;
  }

  .legend li {
    display: flex;
    gap: 6px;
    align-items: center;
    min-width: 0;
  }

  .legend i {
    flex: none;
    width: 12px;
    height: 12px;
    border-radius: 3px;
  }

  .pies {
    display: flex;
    flex-wrap: wrap;
    gap: 16px 24px;
  }

  .pie {
    display: grid;
    gap: 8px;
    justify-items: start;
    min-width: 0;
  }

  .scroll {
    overflow-x: auto;
  }

  table {
    border-collapse: collapse;
    width: 100%;
    font-size: 13px;
  }

  th {
    font-weight: 600;
    vertical-align: bottom;
  }

  th,
  td {
    padding: 6px 6px;
    border-bottom: 1px solid var(--separator);
    text-align: left;
  }

  td.val {
    text-align: right;
    font-variant-numeric: tabular-nums;
  }

  .steps {
    display: grid;
    gap: 8px;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .steps li {
    display: grid;
    grid-template-columns: 28px minmax(0, 1fr);
    gap: 10px;
    align-items: start;
    padding: 10px 12px;
    border: 1px solid var(--separator);
    border-radius: 12px;
  }

  .n {
    display: grid;
    place-items: center;
    width: 28px;
    height: 28px;
    border-radius: 50%;
    background: var(--accent);
    color: var(--accent-text);
    font-weight: 600;
  }
</style>
