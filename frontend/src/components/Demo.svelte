<script lang="ts">
  import { t } from '../lib/i18n/index.svelte'
</script>

<!-- Пометка этапа 12: данные на экране придуманные (ROADMAP). Уходит, когда экран подключают к серверу. -->
<span class="demo">{t('demo.badge')}</span>

<style>
  .demo {
    display: inline-block;
    padding: 1px 8px;
    border: 1px dashed var(--hint);
    border-radius: 999px;
    color: var(--hint);
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    vertical-align: middle;
  }
</style>
