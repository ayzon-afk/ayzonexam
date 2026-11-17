// Запись голоса для Speaking: встроенный MediaRecorder, без библиотек. Сжатие opus/aac ~32 кбит/с —
// минута ответа ≈ 240 КБ, отправка не мучает мобильный интернет. Не вышло (нет микрофона, старый
// телефон) — запасной путь: голосовое боту (SPEC).

const TYPES = ['audio/ogg;codecs=opus', 'audio/webm;codecs=opus', 'audio/mp4', 'audio/webm']

export function canRecord(): boolean {
  return typeof MediaRecorder !== 'undefined' && !!navigator.mediaDevices?.getUserMedia
}

export type Recording = { blob: Blob; seconds: number }

export type Recorder = {
  stop(): Promise<Recording>
  cancel(): void
}

export async function startRecording(): Promise<Recorder> {
  const stream = await navigator.mediaDevices.getUserMedia({ audio: { echoCancellation: true, noiseSuppression: true } })
  const mimeType = TYPES.find((t) => MediaRecorder.isTypeSupported(t))
  const rec = new MediaRecorder(stream, { ...(mimeType ? { mimeType } : {}), audioBitsPerSecond: 32_000 })
  const chunks: Blob[] = []
  const started = Date.now()
  rec.ondataavailable = (e) => {
    if (e.data.size) chunks.push(e.data)
  }
  rec.start(1000)
  const release = () => stream.getTracks().forEach((t) => t.stop())
  return {
    stop: () =>
      new Promise((resolve) => {
        rec.onstop = () => {
          release()
          resolve({ blob: new Blob(chunks, { type: rec.mimeType || mimeType || 'audio/webm' }), seconds: (Date.now() - started) / 1000 })
        }
        rec.stop()
      }),
    cancel: () => {
      rec.onstop = null
      if (rec.state !== 'inactive') rec.stop()
      release()
    },
  }
}
