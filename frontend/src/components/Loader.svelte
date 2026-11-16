<script lang="ts" module>
  // Долгое ожидание (запуск, проверка эссе) — персонаж и мотивирующие фразы; короткое — Skeleton.
  // penguin — пингвин жмёт штангу, на блинах растёт балл (идея владельца); chart — растущий график.
  // Какой показывать — решает владелец (ROADMAP, этап 12); второй уйдёт.
  export type LoaderKind = 'penguin' | 'chart'
  export const LOADER: LoaderKind = 'penguin'
</script>

<script lang="ts">
  import { onDestroy } from 'svelte'
  import { t, type Key } from '../lib/i18n/index.svelte'

  let { kind = LOADER, phrases = true }: { kind?: LoaderKind; phrases?: boolean } = $props()

  const PHRASES: Key[] = ['loader.m1', 'loader.m2', 'loader.m3', 'loader.m4', 'loader.m5', 'loader.m6']

  // Один подход — 1.6 с: балл растёт с каждым подъёмом штанги, фраза меняется каждые два.
  let rep = $state(0)
  const timer = setInterval(() => (rep += 1), 1600)
  onDestroy(() => clearInterval(timer))

  const band = $derived((5.5 + (rep % 8) * 0.5).toFixed(1))
  const phrase = $derived(PHRASES[Math.floor(rep / 2) % PHRASES.length]!)
</script>

<div class="loader" role="status">
  {#if kind === 'penguin'}
    <svg viewBox="0 0 120 124" aria-hidden="true">
      <ellipse class="shadow" cx="60" cy="116" rx="28" ry="4" />
      <g class="body">
        <ellipse class="foot" cx="49" cy="112" rx="9" ry="4" />
        <ellipse class="foot" cx="71" cy="112" rx="9" ry="4" />
        <ellipse class="fur" cx="60" cy="84" rx="25" ry="29" />
        <ellipse class="belly" cx="60" cy="90" rx="17" ry="21" />
        <circle class="fur" cx="60" cy="54" r="19" />
        <ellipse class="belly" cx="60" cy="58" rx="12" ry="10" />
        <circle class="eye" cx="54" cy="52" r="3.6" />
        <circle class="eye" cx="66" cy="52" r="3.6" />
        <circle class="pupil" cx="55" cy="52.5" r="1.8" />
        <circle class="pupil" cx="67" cy="52.5" r="1.8" />
        <path class="beak" d="M55.5 58h9L60 64z" />
        <circle class="cheek" cx="49" cy="60" r="2.6" />
        <circle class="cheek" cx="71" cy="60" r="2.6" />
      </g>
      <g class="lift">
        <ellipse class="fur" cx="38" cy="50" rx="5.5" ry="15" transform="rotate(-14 38 50)" />
        <ellipse class="fur" cx="82" cy="50" rx="5.5" ry="15" transform="rotate(14 82 50)" />
        <rect class="bar" x="10" y="31" width="100" height="5" rx="2.5" />
        <rect class="plate" x="12" y="20" width="10" height="27" rx="3" />
        <rect class="plate" x="98" y="20" width="10" height="27" rx="3" />
        <rect class="plate small" x="23" y="24" width="6" height="19" rx="2" />
        <rect class="plate small" x="91" y="24" width="6" height="19" rx="2" />
        <text x="60" y="26" text-anchor="middle">{band}</text>
      </g>
    </svg>
  {:else}
    <svg viewBox="0 0 120 100" aria-hidden="true">
      {#each [22, 34, 48, 62, 80] as h, i (i)}
        <rect class="col" x={12 + i * 20} y={90 - h} width="14" height={h} rx="4" style:animation-delay="{i * 0.16}s" />
      {/each}
      <path class="axis" d="M6 90h108" />
      <text class="big" x="60" y="18" text-anchor="middle">{band}</text>
    </svg>
  {/if}
  {#if phrases}
    {#key phrase}<p class="phrase">{t(phrase)}</p>{/key}
  {/if}
</div>

<style>
  .loader {
    display: grid;
    justify-items: center;
    gap: 14px;
    text-align: center;
  }

  svg {
    width: 150px;
    height: auto;
    overflow: visible;
  }

  .shadow {
    fill: var(--hint);
    opacity: 0.25;
  }

  /* Обводка цвета разделителя — чтобы тёмный пингвин был виден и на чёрном фоне. */
  .fur {
    fill: #2b3547;
    stroke: var(--separator);
    stroke-width: 1;
  }

  .belly,
  .eye {
    fill: #ffffff;
  }

  .pupil {
    fill: #111;
  }

  .beak,
  .foot {
    fill: #f5a524;
  }

  .cheek {
    fill: #ff8fa3;
    opacity: 0.7;
  }

  .bar {
    fill: #9aa4b2;
  }

  .plate {
    fill: var(--accent);
  }

  .plate.small {
    opacity: 0.75;
  }

  text {
    fill: var(--score);
    font-size: 13px;
    font-weight: 800;
    font-variant-numeric: tabular-nums;
  }

  /* Подъём: штанга с лапами вверх-вниз, тело чуть приседает внизу. */
  .lift {
    animation: press 1.6s ease-in-out infinite;
  }

  .body {
    transform-origin: 60px 116px;
    animation: squat 1.6s ease-in-out infinite;
  }

  @keyframes press {
    0%,
    100% {
      transform: translateY(14px);
    }
    50% {
      transform: translateY(0);
    }
  }

  @keyframes squat {
    0%,
    100% {
      transform: scaleY(0.96);
    }
    50% {
      transform: scaleY(1);
    }
  }

  .col {
    fill: var(--accent);
    transform-box: fill-box;
    transform-origin: bottom;
    animation: grow 1.6s ease-in-out infinite;
  }

  .axis {
    stroke: var(--separator);
    stroke-width: 2;
  }

  .big {
    font-size: 18px;
  }

  @keyframes grow {
    0%,
    100% {
      transform: scaleY(0.25);
      opacity: 0.5;
    }
    50% {
      transform: scaleY(1);
      opacity: 1;
    }
  }

  .phrase {
    min-height: 2.8em;
    max-width: 280px;
    color: var(--hint);
    font-weight: 500;
    animation: fade 0.5s ease-out;
  }

  @keyframes fade {
    from {
      opacity: 0;
      transform: translateY(4px);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .lift,
    .body,
    .col,
    .phrase {
      animation: none;
    }
  }
</style>
