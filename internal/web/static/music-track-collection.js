/**
 * Renders one bounded music-track collection without owning playback or DOM
 * state. Favorites and Recently Played therefore share one cover/list contract.
 */
export function musicTrackCollectionHTML(
  items,
  { layout = 'list', row, card, empty = 'Nothing found here yet.' },
) {
  if (!items.length) return `<p class="empty-state">${empty}</p>`
  if (layout === 'covers') {
    return `<div class="music-track-grid">${items.map(item => card(item)).join('')}</div>`
  }
  return items.map(item => row(item)).join('')
}
