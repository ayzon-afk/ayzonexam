<script lang="ts">
  import Head from '../../components/Head.svelte'
  import { mine } from '../../lib/demo'
  import { t } from '../../lib/i18n/index.svelte'

  // Что бесплатно и что платно — решается после пилота (SPEC, «Деньги»): цен на экране нет.
  let asked = $state(false)
</script>

<Head title="Premium" sub={t('premium.lead')} />

<div class="plans">
  <section class="card stack">
    <h2>{t('premium.free')}</h2>
    <ul>
      <li>{t('premium.f1')}</li>
      <li>{t('premium.f2')}</li>
      <li>{t('premium.f3')}</li>
    </ul>
  </section>
  <section class="card stack pro">
    <h2>Premium</h2>
    <ul>
      <li>{t('premium.p1')}</li>
      <li>{t('premium.p2')}</li>
      <li>{t('premium.p3')}</li>
    </ul>
    <button class="btn" onclick={() => (asked = true)}>{t('premium.soon')}</button>
    {#if asked}<p class="hint">{t('demo.later')}</p>{/if}
  </section>
</div>

<div class="sect"><h2>{t('profile.friends')}</h2></div>
<section class="card stack">
  <p>{t('premium.bonus')}</p>
  <p class="hint">{t('premium.invited', { n: mine.invited })}</p>
  <button class="btn btn-plain" onclick={() => (asked = true)}>{t('premium.invite')}</button>
</section>

<style>
  h2 {
    font-size: 18px;
  }

  .plans {
    display: grid;
    gap: 12px;
  }

  @media (min-width: 600px) {
    .plans {
      grid-template-columns: 1fr 1fr;
    }
  }

  ul {
    margin: 0;
    padding-left: 20px;
    display: grid;
    gap: 6px;
  }

  .pro {
    outline: 2px solid var(--accent);
  }
</style>
