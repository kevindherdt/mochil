// A card takes its colour from its own artwork instead of from a blurred copy of it. The colour is
// the average of the cover's lively pixels (the same measurement the expanded player uses), mixed a
// little into the card as a glow behind the top-left corner, so a shelf reads as a row of coloured
// objects rather than a row of identical grey boxes.
import { dominantColour } from './audio-player-tint.js?v=4'

const SIZE = 16

function colour(image) {
  try {
    const canvas = document.createElement('canvas')
    canvas.width = canvas.height = SIZE
    const context = canvas.getContext('2d', { willReadFrequently: true })
    context.drawImage(image, 0, 0, SIZE, SIZE)
    return dominantColour(context.getImageData(0, 0, SIZE, SIZE).data)
  } catch {
    // A cross-origin or not-yet-decodable image simply keeps the card's plain surface.
    return ''
  }
}

export function createCardTint({
  root = document,
  // The same cards that wear the shared card look in card-surface.css: if a surface gets the padding,
  // the border and the top edge, it gets the glow from its own artwork too.
  selector = '.jump-card, .book-grid .book:not(.book-list-item), .series-cover-grid .series-card, '
    + '.music-group-grid:not(.list) .music-group-card, .podcast-card, .audiobook-series-card, .music-track-card',
  Observer = globalThis.IntersectionObserver,
  MutationObserverClass = globalThis.MutationObserver,
} = {}) {
  const done = new WeakSet()

  function paint(card, image = card.querySelector('img')) {
    if (!image || done.has(card)) return
    const apply = () => {
      const value = colour(image)
      if (!value) return
      done.add(card)
      card.style.setProperty('--art-tint', value)
    }
    if (image.complete && image.naturalWidth) apply()
    else image.addEventListener('load', apply, { once: true })
  }

  // Every shelf in this app is rendered by replacing innerHTML, so the image that carries the colour
  // arrives after the card does. Listening for image loads in the capture phase covers every render
  // path, including shelves drawn before this function was called, and costs nothing per card.
  (globalThis.document?.addEventListener)?.('load', (event) => {
    const image = event.target
    if (image?.tagName !== 'IMG') return
    const card = image.closest?.(selector)
    if (card) paint(card, image)
  }, true)

  // Only cards that have been on screen are sampled: a grid of three hundred books must not cost
  // three hundred canvas reads at boot.
  const watcher = Observer ? new Observer((entries) => {
    for (const entry of entries) if (entry.isIntersecting) { paint(entry.target); watcher.unobserve(entry.target) }
  }, { rootMargin: '200px' }) : null

  function scan(node) {
    if (!node || node.nodeType !== 1) return
    const cards = node.matches?.(selector) ? [node] : [...node.querySelectorAll(selector)]
    // A shelf holds a handful of cards, so its row can be painted at once and look whole; a grid of
    // three hundred books waits until each card is actually on screen.
    const paintAtOnce = cards.length <= 24
    for (const card of cards) {
      if (paintAtOnce || !watcher) paint(card)
      else watcher.observe(card)
    }
  }

  scan(root)
  if (MutationObserverClass && root) {
    new MutationObserverClass((records) => {
      for (const record of records) {
        for (const node of record.addedNodes) {
          // A cover loader may swap the card's <img> for a sized one after the card was painted;
          // that image is not a card, so the card around it is what has to be painted again.
          if (node.nodeType === 1) {
            const card = node.closest?.(selector)
            if (card) { paint(card, node.tagName === 'IMG' ? node : undefined); continue }
          }
          scan(node)
        }
      }
    }).observe(root, { childList: true, subtree: true })
  }

  return { scan }
}
