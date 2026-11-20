// The music "tracks" surface renders the same list two ways (replace on reload,
// append/prepend while scrolling) and keeps the live DOM bounded via a windowed
// container. These steps used to live in one long inline branch of render();
// pulling them out keeps each step named and independently testable.

/** Maps the render() append flag onto the insertion mode the window understands. */
export function musicTrackRenderMode(append) {
  return append === true ? 'append' : append === 'prepend' ? 'prepend' : 'replace'
}

/** Builds the markup for a page of tracks, as cover cards or as rows. */
export function musicTrackMarkup(items, covers, renderCard, renderRow) {
  return items.map(item => (covers ? renderCard(item) : renderRow(item))).join('')
}

const emptyTrackListMarkup = '<p class="empty-state">Nothing found here yet.</p>'

/**
 * Commits a page of track markup to the DOM: a full reset for "replace", or a
 * windowed insert for "append"/"prepend". Returns whatever the bounded window
 * reports as trimmed, so the caller can keep its backing array in sync.
 */
export function applyMusicTrackWindow({ mode, markup, container, trackWindow }) {
  if (mode === 'replace') {
    trackWindow.reset()
    container.innerHTML = markup || emptyTrackListMarkup
    return { start: 0, end: 0 }
  }
  const insertion = trackWindow.begin(mode)
  container.insertAdjacentHTML(mode === 'append' ? 'beforeend' : 'afterbegin', markup)
  return trackWindow.finish(insertion)
}
