// The expanded player takes a soft tint from the cover that is playing, the way Apple Music and Spotify do. The colour
// is the average of the cover's lively pixels; the stylesheet mixes only a little of it into the surface, so text stays
// readable whatever the cover looks like.
const SIZE = 16

/** Average colour of the pixels that carry colour; dull greys, blacks and whites are left out so a white-bordered cover still tints. */
export function dominantColour(data) {
  let red = 0, green = 0, blue = 0, weight = 0
  for (let i = 0; i + 3 < data.length; i += 4) {
    const [r, g, b, a] = [data[i], data[i + 1], data[i + 2], data[i + 3]]
    if (a < 128) continue
    const max = Math.max(r, g, b), min = Math.min(r, g, b)
    const saturation = max ? (max - min) / max : 0
    const lightness = (max + min) / 510
    const w = saturation * (1 - Math.abs(lightness - .5) * 1.4)
    if (w <= .05) continue
    red += r * w; green += g * w; blue += b * w; weight += w
  }
  if (!weight) return ''
  return `rgb(${Math.round(red / weight)} ${Math.round(green / weight)} ${Math.round(blue / weight)})`
}

/** A CSS url() for the cover, safe to put in a custom property. */
export const backdropURL = src => `url("${String(src).replaceAll('"', '%22').replaceAll('\\', '%5C')}")`

export function createPlayerTint({ player, art, document, MutationObserverClass = globalThis.MutationObserver }) {
  function paint(image) {
    try {
      const canvas = document.createElement('canvas')
      canvas.width = canvas.height = SIZE
      const context = canvas.getContext('2d', { willReadFrequently: true })
      context.drawImage(image, 0, 0, SIZE, SIZE)
      const colour = dominantColour(context.getImageData(0, 0, SIZE, SIZE).data)
      if (colour) player.style.setProperty('--art-tint', colour)
      else player.style.removeProperty('--art-tint')
      player.style.setProperty('--art-image', backdropURL(image.currentSrc || image.src))
    } catch {
      player.style.removeProperty('--art-tint')
      player.style.removeProperty('--art-image')
    }
  }
  function sync() {
    const image = art.querySelector('img')
    if (!image) { player.style.removeProperty('--art-tint'); player.style.removeProperty('--art-image'); return }
    if (image.complete && image.naturalWidth) paint(image)
    else image.addEventListener('load', () => paint(image), { once: true })
  }
  if (MutationObserverClass) new MutationObserverClass(sync).observe(art, { childList: true })
  sync()
  return { sync }
}
