<script lang="ts">
  import { t, type Key } from '../lib/i18n/index.svelte'

  let { title, text }: { title: Key; text: Key } = $props()
</script>

<h1>{t(title)}</h1>
<div class="card">
  <p class="badge">{t('soon.title')}</p>
  <p>{t(text)}</p>
</div>

<style>
  h1 {
    font-size: 26px;
  }

  .badge {
    display: inline-block;
    margin-bottom: 8px;
    padding: 2px 10px;
    border-radius: 999px;
    background: var(--accent);
    color: var(--accent-text);
    font-size: 13px;
    font-weight: 600;
  }
</style>
