import { dominantColour } from './audio-player-tint.js?v=3'

/** A colour taken from a cover, used to tint the page around it. The picking itself is the player's (audio-player-tint.js). */
function read(img, doc) {
  try {
    const canvas = doc.createElement('canvas')
    canvas.width = canvas.height = 24
    const ctx = canvas.getContext('2d', { willReadFrequently: true })
    ctx.drawImage(img, 0, 0, 24, 24)
    return dominantColour(ctx.getImageData(0, 0, 24, 24).data) || null
  } catch { return null } // cross-origin or no canvas: the page keeps its normal colours
}

/** Sets --tint on `target` once `img` has a picture. Without one, --tint is unset and the CSS falls back to the accent. */
export function applyCoverTint(target, img, doc = globalThis.document) {
  if (!target || !img) return
  const run = () => {
    if (!img.naturalWidth) return
    const color = read(img, doc)
    if (color) target.style.setProperty('--tint', color)
  }
  if (img.complete && img.naturalWidth) run()
  img.addEventListener('load', run)
}
