<script lang="ts">
  import Icon from './Icon.svelte'
  import Logo from './Logo.svelte'
  import { t } from '../lib/i18n/index.svelte'
  import { share, type Shared } from '../lib/share'
  import { haptic } from '../lib/telegram'

  // Карточка для сторис и чатов: большой результат на фирменном градиенте и кнопка «Поделиться».
  // text — что уйдёт в сообщение вместе со ссылкой (url — по умолчанию адрес приложения).
  let { big, title, sub = '', text, url }: { big: string; title: string; sub?: string; text: string; url?: string } = $props()

  let result = $state<Shared | null>(null)

  async function onshare() {
    haptic.select()
    result = await share(text, url)
  }
</script>

<div class="story">
  <span class="brand"><Logo size={26} /></span>
  <p class="title">{title}</p>
  <p class="big">{big}</p>
  {#if sub}<p class="sub">{sub}</p>{/if}
</div>
<button class="btn share" onclick={onshare}><Icon name="share" size={20} /> {t('share')}</button>
{#if result === 'copied'}<p class="hint center">{t('share.copied')}</p>{/if}

<style>
  .story {
    aspect-ratio: 4 / 5;
    max-width: 320px;
    width: 100%;
    margin: 0 auto;
    padding: 24px;
    border-radius: 22px;
    display: grid;
    align-content: center;
    justify-items: center;
    gap: 8px;
    text-align: center;
    color: #fff;
    background: var(--grad);
  }

  .brand {
    margin-bottom: 8px;
  }

  .brand :global(.mark) {
    background: rgb(255 255 255 / 0.2);
  }

  .brand :global(b) {
    color: inherit;
    opacity: 0.8;
  }

  .title {
    font-weight: 600;
    opacity: 0.9;
  }

  .big {
    font-size: 76px;
    font-weight: 800;
    line-height: 1;
    font-variant-numeric: tabular-nums;
  }

  .sub {
    font-weight: 600;
    opacity: 0.9;
  }

  .share {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    max-width: 320px;
    margin: 0 auto;
  }

  .center {
    text-align: center;
  }
</style>
