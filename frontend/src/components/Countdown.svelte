<script lang="ts">
  import { onDestroy } from 'svelte'
  import { t } from '../lib/i18n/index.svelte'

  let { to }: { to: Date } = $props()

  let now = $state(Date.now())
  const timer = setInterval(() => (now = Date.now()), 30_000)
  onDestroy(() => clearInterval(timer))

  const left = $derived(Math.max(0, to.getTime() - now) / 60_000)
</script>

<span>{t('time.dhm', { d: Math.floor(left / 1440), h: Math.floor((left % 1440) / 60), m: Math.floor(left % 60) })}</span>
