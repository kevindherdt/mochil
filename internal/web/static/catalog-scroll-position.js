/**
 * Walk a catalogue's cards to find the bucket owned by the first item
 * intersecting the content edge (the line just under the sticky masthead).
 *
 * The cards are in visual order, so a scroll only ever moves the answer a few
 * rows. Remembering the last match and walking from it in the direction of
 * travel keeps a scroll frame down to a couple of rect reads instead of one
 * per card — with 600 cards the difference is ~480 rects per frame versus a
 * handful.
 */
export function createBucketTracker() {
  let last = 0

  function find(items, top) {
    const count = items.length
    if (!count) {
      last = 0
      return ''
    }
    const edge = top + 8
    let index = Math.min(Math.max(last, 0), count - 1)
    let bottom = items[index].getBoundingClientRect().bottom
    if (bottom <= edge) {
      // Scrolled down: the edge has passed this card, so move forward.
      while (bottom <= edge && index + 1 < count) {
        index += 1
        bottom = items[index].getBoundingClientRect().bottom
      }
      // Everything is above the edge: no bucket owns the line.
      if (bottom <= edge) return ''
    } else {
      // Scrolled up: walk back while the previous card still covers the edge.
      while (index > 0 && items[index - 1].getBoundingClientRect().bottom > edge) {
        index -= 1
      }
    }
    last = index
    return items[index].dataset.bucket || '#'
  }

  return { find, reset() { last = 0 } }
}

/** Stateless call for callers that do not keep a catalogue position of their own. */
export function visibleCatalogueBucket(items, top) {
  for (const item of items) {
    if (item.getBoundingClientRect().bottom > top + 8) return item.dataset.bucket || '#'
  }
  return ''
}
