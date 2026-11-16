// Сжатие фото на телефоне перед отправкой: снимок камеры весит 3–8 МБ, после — 0.3–0.8 МБ.
// Сервер принимает до 1.5 МБ на фото (api.maxImageBytes). Без библиотек: <img> + canvas.

const MAX_BYTES = 1400 * 1024

export async function compressPhoto(file: Blob, maxSide = 2000): Promise<Blob> {
  const url = URL.createObjectURL(file)
  try {
    const img = new Image()
    img.src = url
    // <img> учитывает поворот из EXIF — фото с телефона не ляжет набок.
    await img.decode()
    const scale = Math.min(1, maxSide / Math.max(img.naturalWidth, img.naturalHeight))
    const canvas = document.createElement('canvas')
    canvas.width = Math.round(img.naturalWidth * scale)
    canvas.height = Math.round(img.naturalHeight * scale)
    const ctx = canvas.getContext('2d')
    if (!ctx) throw new Error('canvas')
    ctx.drawImage(img, 0, 0, canvas.width, canvas.height)
    for (const quality of [0.82, 0.7, 0.55]) {
      const blob = await new Promise<Blob | null>((ok) => canvas.toBlob(ok, 'image/jpeg', quality))
      if (blob && blob.size <= MAX_BYTES) return blob
    }
    throw new Error('photo_too_large')
  } finally {
    URL.revokeObjectURL(url)
  }
}
