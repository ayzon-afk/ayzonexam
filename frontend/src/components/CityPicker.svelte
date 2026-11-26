<script lang="ts">
  import { CITIES, cityKey, REGIONS, type City } from '../lib/cities'
  import { t } from '../lib/i18n/index.svelte'

  // Регион ученика: системный список — на телефоне это родное колесо выбора, без своей вёрстки.
  // regions — только регионы Узбекистана (центр, учитель), без «другой страны».
  let { value, onpick, regions = false }: { value: '' | City; onpick: (c: City) => void; regions?: boolean } = $props()
</script>

<select aria-label={t('profile.city')} value={value} onchange={(e) => onpick((e.target as HTMLSelectElement).value as City)}>
  {#if !value}<option value="" disabled selected>{t('city.pick')}</option>{/if}
  {#each regions ? REGIONS : CITIES as c (c)}<option value={c}>{t(cityKey(c))}</option>{/each}
</select>

<style>
  select {
    width: 100%;
    min-width: 0;
    min-height: 50px;
    padding: 0 14px;
    border: 1px solid var(--separator);
    border-radius: var(--radius);
    background: var(--section);
    color: var(--text);
    font: inherit;
  }
</style>
