<script lang="ts">
  import type { Me } from '../../lib/api'
  import { t } from '../../lib/i18n/index.svelte'
  import { inviteLink, share } from '../../lib/share'
  import { haptic } from '../../lib/telegram'

  // Ссылка-приглашение центра: видна целиком, «Копировать» и «Поделиться». Кто придёт по ней — сразу в центре.
  let { me, name, code }: { me: Me; name: string; code: string } = $props()

  const link = $derived(inviteLink(me.bot, `c-${code}`))
  let copied = $state(false)

  async function copy() {
    try {
      await navigator.clipboard.writeText(link)
      copied = true
      haptic.success()
      setTimeout(() => (copied = false), 2000)
    } catch {
      haptic.error()
    }
  }
</script>

<div class="invite">
  <code>{link}</code>
  <div class="acts">
    <button class="btn btn-small" onclick={copy}>{copied ? t('cab.copied') : t('cab.copy')}</button>
    <button class="btn btn-small btn-plain" onclick={() => share(t('center.inviteText', { name }), link)}>{t('cab.share')}</button>
  </div>
  <p class="hint">{t('center.inviteHint')}</p>
</div>

<style>
  .invite {
    display: grid;
    gap: 10px;
  }

  code {
    display: block;
    padding: 10px 12px;
    border-radius: 10px;
    background: var(--surface);
    font-size: 14px;
    overflow-wrap: anywhere;
    user-select: all;
  }

  .acts {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  .acts .btn-plain {
    background: var(--surface);
  }
</style>
