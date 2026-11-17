<script lang="ts">
  import { onMount } from 'svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { adm, type Row } from './admin'

  // Учителя и аккаунты центров. Назначить — в «Ученики»: найти человека, выбрать центр и роль.
  let { goto }: { goto: (id: string) => void } = $props()

  let list = $state<Row[] | null>(null)
  onMount(async () => {
    list = (await adm.get<{ teachers: Row[] }>('teachers')).teachers
  })
</script>

<p class="hint">Учитель или аккаунт центра — это ученик с ролью. Назначить: «Ученики» → найти → центр и роль, «Проверенный учитель» — галочкой.</p>
<button class="btn btn-plain" onclick={() => goto('users')}>Открыть «Ученики»</button>

{#if !list}
  <Skeleton rows={3} />
{:else if list.length === 0}
  <p class="card hint">Пока никого.</p>
{:else}
  <ul class="card rows">
    {#each list as u (u.id)}
      <li class="row">
        <span class="grow">
          <strong>{u.first_name} · {u.nickname}</strong>
          <span class="tag">@{u.username || '—'} · {u.center ?? 'без центра'} · {u.center_role === 'admin' ? 'аккаунт центра' : u.center_role === 'teacher' ? 'учитель' : '—'}</span>
        </span>
        {#if u.verified_teacher}<span class="tag ok">✓ проверенный</span>{/if}
      </li>
    {/each}
  </ul>
{/if}

<style>
  strong {
    display: block;
  }

  .ok {
    color: #2e9d58;
  }
</style>
