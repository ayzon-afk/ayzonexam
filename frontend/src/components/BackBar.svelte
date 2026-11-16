<script lang="ts">
  import { t } from '../lib/i18n/index.svelte'
  import { hasBackButton, showBack } from '../lib/telegram'

  // В Telegram «Назад» — системная кнопка в шапке; вне его (браузер при разработке) — своя.
  let { onback }: { onback: () => void } = $props()

  $effect(() => showBack(onback))
</script>

{#if !hasBackButton()}
  <button class="back" onclick={onback}>‹ {t('onb.back')}</button>
{/if}

<style>
  .back {
    align-self: flex-start;
    min-height: 32px;
    color: var(--link);
  }
</style>
