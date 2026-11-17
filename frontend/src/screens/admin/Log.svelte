<script lang="ts">
  import { onMount } from 'svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { adm, when, type Row } from './admin'

  // Журнал: кто из админов что и когда изменил (SPEC, «Админка»).
  let list = $state<Row[] | null>(null)
  onMount(async () => {
    list = (await adm.get<{ log: Row[] }>('log')).log
  })
</script>

{#if !list}
  <Skeleton rows={4} />
{:else if list.length === 0}
  <p class="card hint">Пока пусто.</p>
{:else}
  <ul class="card rows">
    {#each list as l (l.id)}
      <li class="row">
        <span class="grow">
          <strong>{l.action} {l.target}</strong>
          <span class="tag">{when(l.created_at)} · админ {l.admin_tg} · {JSON.stringify(l.details)}</span>
        </span>
      </li>
    {/each}
  </ul>
{/if}

<style>
  strong {
    display: block;
    font-family: ui-monospace, monospace;
    font-size: 13px;
  }

  .tag {
    word-break: break-all;
  }
</style>
