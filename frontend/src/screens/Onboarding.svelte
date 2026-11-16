<script lang="ts">
  import BandPicker from '../components/BandPicker.svelte'
  import { api, ApiError, type Me } from '../lib/api'
  import { examDateRange } from '../lib/exam'
  import { t } from '../lib/i18n/index.svelte'
  import { haptic } from '../lib/telegram'

  // Третий экран предлагает тест уровня; сам тест пока демо (этап 12), настоящий — в этапе 3.
  const TOTAL = 3

  let { ondone }: { ondone: (me: Me, levelTest: boolean) => void } = $props()

  let step = $state(1)
  let goal = $state<number | null>(null)
  let examDate = $state('')
  let unknownDate = $state(false)
  let saving = $state(false)
  let error = $state('')

  const range = examDateRange(new Date())

  async function finish(levelTest: boolean) {
    saving = true
    error = ''
    try {
      const me = await api.updateMe({ goal_band: goal, exam_date: unknownDate ? null : examDate, onboarded: true })
      haptic.success()
      ondone(me, levelTest)
    } catch (e) {
      haptic.error()
      error = e instanceof ApiError && e.code === 'bad_exam_date' ? t('error.date') : t('app.error')
    } finally {
      saving = false
    }
  }
</script>

<main>
  <header>
    {#if step > 1}
      <button class="back" onclick={() => (step -= 1)}>‹ {t('onb.back')}</button>
    {/if}
    <span class="hint">{t('onb.step', { n: step, total: TOTAL })}</span>
  </header>

  {#if step === 1}
    <section>
      <h1>{t('onb.goal.title')}</h1>
      <p class="hint">{t('onb.goal.hint')}</p>
      <BandPicker bind:value={goal} />
    </section>
    <footer>
      <button class="btn" disabled={goal === null} onclick={() => (step = 2)}>{t('onb.next')}</button>
    </footer>
  {:else if step === 2}
    <section>
      <h1>{t('onb.date.title')}</h1>
      <p class="hint">{t('onb.date.hint')}</p>
      <input type="date" min={range.min} max={range.max} bind:value={examDate} />
    </section>
    <footer>
      <button class="btn" disabled={!examDate} onclick={() => ((unknownDate = false), (step = 3))}>{t('onb.next')}</button>
      <button class="btn btn-plain" onclick={() => ((unknownDate = true), (step = 3))}>{t('onb.date.unknown')}</button>
    </footer>
  {:else}
    <section>
      <h1>{t('onb.level.title')}</h1>
      <p class="hint">{t('onb.level.hint')}</p>
      {#if error}<p class="error">{error}</p>{/if}
    </section>
    <footer>
      <button class="btn" disabled={saving} onclick={() => finish(true)}>{t('onb.level.start')}</button>
      <button class="btn btn-plain" disabled={saving} onclick={() => finish(false)}>{t('onb.level.later')}</button>
    </footer>
  {/if}
</main>

<style>
  main {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    padding: calc(var(--safe-top) + 12px) 16px calc(var(--safe-bottom) + 16px);
    background: var(--bg);
  }

  header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    min-height: 32px;
  }

  .back {
    color: var(--link);
  }

  header .hint {
    margin-left: auto;
  }

  section {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding-top: 24px;
  }

  h1 {
    font-size: 26px;
    line-height: 1.2;
  }

  section .hint {
    margin-bottom: 12px;
  }

  footer {
    display: grid;
    gap: 8px;
  }
</style>
