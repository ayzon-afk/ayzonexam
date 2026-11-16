<script lang="ts" module>
  // Иконки — контуры 24×24 линией текущего цвета, как во вкладках.
  const paths = {
    pen: 'M4 20h4L19 9a2.8 2.8 0 0 0-4-4L4 16zM13.5 6.5l4 4',
    book: 'M4 19.5V5a2 2 0 0 1 2-2h13v15H6a2 2 0 0 0-2 2 2 2 0 0 0 2 2h13M8 7h7',
    headphones: 'M4 15v-3a8 8 0 0 1 16 0v3M4 15h3v6H5a1 1 0 0 1-1-1zM20 15h-3v6h2a1 1 0 0 0 1-1z',
    mic: 'M12 3a3 3 0 0 1 3 3v6a3 3 0 0 1-6 0V6a3 3 0 0 1 3-3zM5 11a7 7 0 0 0 14 0M12 18v3',
    timer: 'M12 9v4l2.5 2M9 2h6M12 22a8 8 0 1 0 0-16 8 8 0 0 0 0 16z',
    target: 'M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18zM12 16a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM12 12h.01',
    share: 'M4 13v6a1 1 0 0 0 1 1h14a1 1 0 0 0 1-1v-6M12 3v12M7.5 7.5 12 3l4.5 4.5',
    history: 'M3 12a9 9 0 1 0 3-6.7L3 8M3 3v5h5M12 7v5l3 2',
  }
  export type IconName = keyof typeof paths
</script>

<script lang="ts">
  let { name, size = 24 }: { name: IconName; size?: number } = $props()
</script>

<svg viewBox="0 0 24 24" width={size} height={size} aria-hidden="true"><path d={paths[name]} /></svg>

<style>
  svg {
    flex: none;
    fill: none;
    stroke: currentColor;
    stroke-width: 1.8;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
</style>
