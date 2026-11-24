// Камера на полном mock с ноутбука (DECISIONS 2026-11-24). Лицо и взгляд распознаёт MediaPipe Face
// Landmarker прямо в браузере: библиотека (~10 МБ) и модель (~4 МБ) грузятся с CDN только здесь и только
// на ноутбуке, телефоны их не качают. Видео не уходит с ноутбука — на сервер идёт только число нарушений.

const MP = 'https://cdn.jsdelivr.net/npm/@mediapipe/tasks-vision@0.10.14'
const MODEL = 'https://storage.googleapis.com/mediapipe-models/face_landmarker/face_landmarker/float16/1/face_landmarker.task'

// Столько подряд можно не смотреть на экран, дальше — нарушение.
export const AWAY_MS = 5000

export type Pt = { x: number; y: number }
export type Frame = {
  faceLandmarks: Pt[][]
  faceBlendshapes?: { categories: { categoryName: string; score: number }[] }[]
}
type Landmarker = { detectForVideo(video: HTMLVideoElement, time: number): Frame }

// Полный mock — только с ноутбука: мышь или тачпад, большой экран и камера в браузере.
export function isLaptop(): boolean {
  return (
    matchMedia('(pointer: fine) and (hover: hover)').matches &&
    Math.min(screen.width, screen.height) >= 600 &&
    !!navigator.mediaDevices?.getUserMedia
  )
}

let loading: Promise<Landmarker> | undefined

// landmarker грузится один раз на открытие сайта: проверка камеры перед mock прогревает его для частей.
export function landmarker(): Promise<Landmarker> {
  loading ??= (async () => {
    const vision = await import(/* @vite-ignore */ `${MP}/vision_bundle.mjs`)
    const files = await vision.FilesetResolver.forVisionTasks(`${MP}/wasm`)
    return (await vision.FaceLandmarker.createFromOptions(files, {
      baseOptions: { modelAssetPath: MODEL, delegate: 'GPU' },
      runningMode: 'VIDEO',
      numFaces: 2,
      outputFaceBlendshapes: true,
    })) as Landmarker
  })()
  loading.catch(() => (loading = undefined)) // сеть упала — следующая попытка грузит заново
  return loading
}

export type Look = 'ok' | 'none' | 'many' | 'away'

// look — куда смотрит ученик на кадре: нет лица, два лица, голова или глаза в сторону, на экран.
// Вниз не считается: на ноутбуке смотрят на клавиатуру, когда печатают.
export function look(f: Frame): Look {
  const faces = f.faceLandmarks?.length ?? 0
  if (faces === 0) return 'none'
  if (faces > 1) return 'many'
  const p = f.faceLandmarks[0]!
  // Поворот головы: кончик носа (1) между внешними уголками глаз (33 и 263); прямо — около середины.
  const span = p[263]!.x - p[33]!.x
  const turn = span ? (p[1]!.x - p[33]!.x) / span : 0.5
  if (turn < 0.25 || turn > 0.75) return 'away'
  const bs: Record<string, number> = {}
  for (const c of f.faceBlendshapes?.[0]?.categories ?? []) bs[c.categoryName] = c.score
  // Глаза в одну сторону: один смотрит наружу, другой внутрь.
  const side = Math.max(
    Math.min(bs.eyeLookOutLeft ?? 0, bs.eyeLookInRight ?? 0),
    Math.min(bs.eyeLookInLeft ?? 0, bs.eyeLookOutRight ?? 0),
  )
  return side > 0.6 ? 'away' : 'ok'
}

export async function openCamera(video: HTMLVideoElement): Promise<MediaStream> {
  const stream = await navigator.mediaDevices.getUserMedia({
    video: { width: { ideal: 640 }, height: { ideal: 480 }, facingMode: 'user' },
    audio: false,
  })
  video.srcObject = stream
  await video.play()
  return stream
}

// watch — 4 кадра в секунду: этого хватает на «дольше 5 секунд» и не грузит слабый ноутбук.
export async function watch(video: HTMLVideoElement, onLook: (l: Look) => void): Promise<() => void> {
  const lm = await landmarker()
  const id = setInterval(() => {
    if (video.readyState >= 2) onLook(look(lm.detectForVideo(video, performance.now())))
  }, 250)
  return () => clearInterval(id)
}

export function stopCamera(stream: MediaStream | null) {
  stream?.getTracks().forEach((t) => t.stop())
}

// Проверка камеры пройдена в этой вкладке — mock можно начинать.
const CHECKED = 'ayzon.cam'

export function cameraChecked(): boolean {
  try {
    return sessionStorage.getItem(CHECKED) === '1'
  } catch {
    return false
  }
}

export function markCameraChecked() {
  try {
    sessionStorage.setItem(CHECKED, '1')
  } catch {
    // без хранилища — проверка только на этом экране
  }
}
