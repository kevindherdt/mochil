import { applyCoverTint } from './cover-tint.js?v=2'

/** Tints an opened album's header from its cover. Image `load` does not bubble, so it is caught on the way down. */
export function watchAlbumTint(container) {
  if (!container) return
  container.addEventListener('load', event => {
    const img = event.target
    if (img?.tagName !== 'IMG') return
    const heading = img.closest?.('.music-group-heading.featured')
    if (heading) applyCoverTint(heading, img)
  }, true)
}
