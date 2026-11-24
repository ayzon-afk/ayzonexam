import { describe, expect, it } from 'vitest'
import { look, type Frame, type Pt } from './camera'

// Лицо из 478 точек MediaPipe; важны только уголки глаз (33, 263) и кончик носа (1).
function face(noseX: number): Pt[] {
  const p: Pt[] = Array.from({ length: 478 }, () => ({ x: 0.5, y: 0.5 }))
  p[33] = { x: 0.4, y: 0.45 }
  p[263] = { x: 0.6, y: 0.45 }
  p[1] = { x: noseX, y: 0.55 }
  return p
}

const eyes = (scores: Record<string, number>) => [
  { categories: Object.entries(scores).map(([categoryName, score]) => ({ categoryName, score })) },
]

describe('look', () => {
  it('прямо на экран — ok', () => {
    expect(look({ faceLandmarks: [face(0.5)] })).toBe('ok')
  })
  it('нет лица и два лица', () => {
    expect(look({ faceLandmarks: [] })).toBe('none')
    expect(look({ faceLandmarks: [face(0.5), face(0.5)] })).toBe('many')
  })
  it('голова повёрнута', () => {
    expect(look({ faceLandmarks: [face(0.42)] })).toBe('away')
    expect(look({ faceLandmarks: [face(0.58)] })).toBe('away')
  })
  it('глаза в сторону, но не вниз', () => {
    const side: Frame = { faceLandmarks: [face(0.5)], faceBlendshapes: eyes({ eyeLookOutLeft: 0.8, eyeLookInRight: 0.75 }) }
    expect(look(side)).toBe('away')
    const down: Frame = { faceLandmarks: [face(0.5)], faceBlendshapes: eyes({ eyeLookDownLeft: 0.9, eyeLookDownRight: 0.9 }) }
    expect(look(down)).toBe('ok')
  })
})
