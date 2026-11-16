import { describe, expect, it } from 'vitest'
import { countWords } from './writing'

describe('countWords', () => {
  // Те же примеры — в backend/internal/writing/writing_test.go: правило подсчёта одно.
  const cases: [string, number][] = [
    ['', 0],
    ['   \n\t ', 0],
    ['One two  three', 3],
    ['Well-known facts — and 2 numbers.', 5],
    ['line\nbreak\n\nparagraph', 3],
    ["don't stop , ok", 3],
    ['Ўзбекча матн va lotin', 4],
    ['- — – ...', 0],
  ]
  for (const [text, want] of cases) {
    it(JSON.stringify(text), () => expect(countWords(text)).toBe(want))
  }
})
