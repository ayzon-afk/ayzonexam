<script lang="ts">
  // Буква ника на цветном круге; цвет считается из ника — у ученика он всегда один.
  let { name, size = 36 }: { name: string; size?: number } = $props()

  const hue = $derived([...name].reduce((h, c) => (h * 31 + c.charCodeAt(0)) % 360, 7))
</script>

<span
  aria-hidden="true"
  style:width="{size}px"
  style:height="{size}px"
  style:font-size="{Math.round(size * 0.42)}px"
  style:background="hsl({hue} 50% 48%)">{name.slice(0, 1).toUpperCase()}</span
>

<style>
  span {
    display: grid;
    place-items: center;
    flex: none;
    border-radius: 50%;
    color: #fff;
    font-weight: 700;
  }
</style>
