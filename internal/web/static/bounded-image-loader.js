// Grids show covers far smaller than the stored 600px image. Ask for the
// smallest server-made size that still looks sharp at this card's width.
export function sizedCoverSource(image, source) {
  if (!source.startsWith('/covers/') || /[?&]w=/.test(source)) return source
  const shown = image.clientWidth || image.parentElement?.clientWidth || 0
  const needed = shown * (globalThis.devicePixelRatio || 1)
  const width = needed > 0 ? [160, 240, 360].find(size => needed <= size) : undefined
  return width ? `${source}${source.includes('?') ? '&' : '?'}w=${width}` : source
}

export function createBoundedImageLoader({ concurrency = 8 } = {}) {
  const pending = []
  const queued = new Set()
  let active = 0

  function pump() {
    while (active < concurrency && pending.length) {
      const image = pending.shift()
      if (!queued.has(image)) continue
      queued.delete(image)
      if (!image?.isConnected || image.dataset.src === undefined) continue

      const source = image.dataset.src
      delete image.dataset.src
      active++
      let settled = false
      const finish = event => {
        if (settled) return
        settled = true
        image.classList?.remove('cover-pending')
        if (event?.type === 'error') image.classList?.add('cover-failed')
        active--
        pump()
      }
      image.addEventListener('load', finish, { once: true })
      image.addEventListener('error', finish, { once: true })
      image.classList?.add('cover-pending')
      image.src = sizedCoverSource(image, source)
      if (image.complete) queueMicrotask(finish)
    }
  }

  function enqueue(image) {
    if (!image || image.dataset.src === undefined || queued.has(image)) return
    queued.add(image)
    pending.push(image)
    pump()
  }

  function release(root) {
    root.querySelectorAll('img[data-src]').forEach(image => queued.delete(image))
  }

  return { enqueue, release }
}
