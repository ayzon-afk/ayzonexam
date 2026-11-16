<script lang="ts">
  // Знак AYZON EXAM: буква A как вершина графика роста на фирменном градиенте темы, рядом — название.
  let { size = 32, word = true }: { size?: number; word?: boolean } = $props()
</script>

<span class="logo">
  <span class="mark" style:width="{size}px" style:height="{size}px" style:border-radius="{size * 0.28}px">
    <svg viewBox="0 0 32 32" aria-hidden="true">
      <path d="M8 24 16 7.5 24 24" />
      <path d="M11.6 17.5h8.8" />
    </svg>
  </span>
  {#if word}<span class="word" style:font-size="{size * 0.56}px">AYZON <b>EXAM</b></span>{/if}
</span>

<style>
  .logo {
    display: inline-flex;
    align-items: center;
    gap: 10px;
  }

  .mark {
    display: grid;
    place-items: center;
    flex: none;
    background: var(--grad);
  }

  svg {
    width: 72%;
    height: 72%;
    fill: none;
    stroke: #fff;
    stroke-width: 3.4;
    stroke-linecap: round;
    stroke-linejoin: round;
  }

  .word {
    font-weight: 800;
    letter-spacing: 0.02em;
    white-space: nowrap;
  }

  b {
    color: var(--accent);
  }
</style>
