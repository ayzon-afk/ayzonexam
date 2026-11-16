<script lang="ts">
  // Серые заготовки на время короткой загрузки: экран сразу похож на себя, без слова «Загрузка».
  let { rows = 3, title = true }: { rows?: number; title?: boolean } = $props()
</script>

<div class="sk" aria-busy="true">
  {#if title}<span class="line w40 tall"></span>{/if}
  <div class="card">
    {#each { length: rows } as _, i (i)}
      <div class="row">
        <span class="dot"></span>
        <span class="lines"><span class="line w70"></span><span class="line w40"></span></span>
      </div>
    {/each}
  </div>
</div>

<style>
  .sk {
    display: grid;
    gap: 12px;
  }

  .card {
    display: grid;
    gap: 18px;
  }

  .row {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .lines {
    flex: 1;
    display: grid;
    gap: 8px;
  }

  .line,
  .dot {
    display: block;
    border-radius: 6px;
    background: linear-gradient(90deg, var(--shimmer) 25%, var(--surface) 50%, var(--shimmer) 75%);
    background-size: 300% 100%;
    animation: shimmer 1.3s linear infinite;
  }

  .line {
    height: 12px;
  }

  .tall {
    height: 26px;
  }

  .w70 {
    width: 70%;
  }

  .w40 {
    width: 40%;
  }

  .dot {
    width: 36px;
    height: 36px;
    border-radius: 50%;
    flex: none;
  }

  @keyframes shimmer {
    from {
      background-position: 100% 0;
    }
    to {
      background-position: 0 0;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .line,
    .dot {
      animation: none;
    }
  }
</style>
