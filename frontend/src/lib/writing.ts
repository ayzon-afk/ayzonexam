// Правила Writing на стороне интерфейса. Совпадают с сервером (backend/internal/writing/rules.go).
import type { Task } from './api'

export const MIN_WORDS = 50
export const MAX_WORDS: Record<Task, number> = { task1: 400, task2: 600 }
// Экзаменационный минимум: меньше — ученик теряет баллы, но проверка возможна.
export const EXAM_WORDS: Record<Task, number> = { task1: 150, task2: 250 }

// Слово — кусок между пробелами, в котором есть буква или цифра (как writing.CountWords в Go).
export function countWords(text: string): number {
  let n = 0
  for (const w of text.split(/\s+/)) if (/[\p{L}\p{N}]/u.test(w)) n++
  return n
}

// Черновик — чтобы закрытое посреди эссе приложение ничего не потеряло. Фото задания Task 1 в черновик
// не попадает: после восстановления его снимают снова.
export type Draft = {
  task: Task
  promptId: number | null
  prompt: string
  essay: string
  source: 'text' | 'photo'
}

const DRAFT_KEY = 'ayzon.writing.draft'

export function loadDraft(): Draft | null {
  try {
    const raw = localStorage.getItem(DRAFT_KEY)
    if (!raw) return null
    const d = JSON.parse(raw) as Partial<Draft>
    if (d.task !== 'task1' && d.task !== 'task2') return null
    return {
      task: d.task,
      promptId: typeof d.promptId === 'number' ? d.promptId : null,
      prompt: typeof d.prompt === 'string' ? d.prompt : '',
      essay: typeof d.essay === 'string' ? d.essay : '',
      source: d.source === 'photo' ? 'photo' : 'text',
    }
  } catch {
    return null
  }
}

export function saveDraft(d: Draft) {
  try {
    localStorage.setItem(DRAFT_KEY, JSON.stringify(d))
  } catch {
    // Хранилище недоступно (приватный режим) — работаем без черновика.
  }
}

export function clearDraft() {
  try {
    localStorage.removeItem(DRAFT_KEY)
  } catch {
    // см. saveDraft
  }
}
