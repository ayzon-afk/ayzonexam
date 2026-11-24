<script lang="ts">
  import { onMount } from 'svelte'
  import Skeleton from '../../components/Skeleton.svelte'
  import { authHeader } from '../../lib/api'
  import { inTelegram } from '../../lib/telegram'
  import { adm, type Row } from './admin'

  // Реальное использование для жюри хакатона (DECISIONS 2026-11-24): без владельца и помощников (ADMIN_IDS).
  let o = $state<Row | null>(null)
  let failed = $state(false)
  let note = $state('')
  let busy = $state(false)

  onMount(async () => {
    try {
      o = await adm.get('usage')
    } catch {
      failed = true
    }
  })

  // В Telegram на телефоне файл из приложения не скачать — его присылает бот; в браузере — обычное скачивание.
  async function csv() {
    busy = true
    note = ''
    try {
      if (inTelegram()) {
        await adm.post('usage/send', {})
        note = 'Файл отправлен в чат с ботом.'
      } else {
        const res = await fetch('/api/admin/usage.csv', { headers: { Authorization: authHeader() } })
        if (!res.ok) throw new Error('csv')
        const a = document.createElement('a')
        a.href = URL.createObjectURL(await res.blob())
        a.download = 'ayzon-usage.csv'
        a.click()
        setTimeout(() => URL.revokeObjectURL(a.href), 10_000) // сразу после click часть браузеров обрывает скачивание
      }
    } catch {
      note = 'Не получилось. Проверьте интернет и попробуйте ещё раз.'
    }
    busy = false
  }

  const avg = (minutes: number, users: number) => (users ? (minutes / users).toFixed(1) : '0')
</script>

{#if failed}
  <p class="error">Не загрузилось.</p>
{:else if !o}
  <Skeleton rows={4} />
{:else}
  <p class="hint">Без владельца и помощников. Минуты — только пока приложение открыто и ученик что-то делает.</p>
  <div class="grid">
    <div class="card"><p class="num">{o.active}</p><p class="hint">уникальных пользователей (из {o.users} вошедших)</p></div>
    <div class="card"><p class="num">{o.minutes}</p><p class="hint">активных минут, {avg(o.minutes, o.active)} на человека</p></div>
    <div class="card"><p class="num">{o.sessions}</p><p class="hint">сессий (открытий приложения)</p></div>
    <div class="card"><p class="num">{o.tests}</p><p class="hint">сданных тестов (уровень, Reading, Listening)</p></div>
    <div class="card"><p class="num">{o.writing} / {o.speaking}</p><p class="hint">проверок Writing / Speaking</p></div>
    <div class="card"><p class="num">{o.mocks}</p><p class="hint">завершённых mock</p></div>
    <div class="card"><p class="num">{o.returning}</p><p class="hint">вернулись в другой день</p></div>
    <div class="card"><p class="num">{o.repeat}</p><p class="hint">открывали 2+ раза</p></div>
  </div>

  <button class="btn csv" disabled={busy} onclick={csv}>Скачать CSV</button>
  {#if note}<p class="hint">{note}</p>{/if}

  <div class="table">
    <table>
      <thead>
        <tr><th>День</th><th>Польз.</th><th>Новые</th><th>Сессии</th><th>Минуты</th><th>Тесты</th><th>W</th><th>S</th></tr>
      </thead>
      <tbody>
        {#each o.days as d (d.day)}
          <tr>
            <td>{d.day.slice(5)}</td><td>{d.users}</td><td>{d.new}</td><td>{d.sessions}</td><td>{d.minutes}</td>
            <td>{d.tests}</td><td>{d.writing}</td><td>{d.speaking}</td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
{/if}

<style>
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
    gap: 10px;
    margin: 10px 0;
  }

  .num {
    font-size: 22px;
  }

  .csv {
    max-width: 320px;
  }

  .table {
    overflow-x: auto;
    margin-top: 12px;
  }

  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 14px;
  }

  th,
  td {
    padding: 6px 4px;
    text-align: right;
    white-space: nowrap;
  }

  th:first-child,
  td:first-child {
    text-align: left;
  }
</style>
